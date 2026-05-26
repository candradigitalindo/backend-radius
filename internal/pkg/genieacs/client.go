package genieacs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/candrasyahputra/radius-server/internal/config"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	username   string
	password   string
}

func NewClient(cfg config.GenieACSConfig) *Client {
	return &Client{
		baseURL: strings.TrimRight(cfg.URL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		username: cfg.Username,
		password: cfg.Password,
	}
}

type Device struct {
	ID           string           `json:"_id"`
	SerialNumber string           `json:"serial_number,omitempty"`
	ProductClass string           `json:"product_class,omitempty"`
	Manufacturer string           `json:"manufacturer,omitempty"`
	Tags         []string         `json:"_tags,omitempty"`
	LastInform   string           `json:"_lastInform,omitempty"`
	Parameters   map[string]Param `json:"parameters,omitempty"`
	Raw          map[string]json.RawMessage `json:"-"`
}

// HasNestedParam returns true if the given dot-separated path exists in the device data,
// regardless of whether it has a _value. Useful for detecting supported parameters.
func (d *Device) HasNestedParam(path string) bool {
	if d.Raw == nil {
		return false
	}
	parts := strings.Split(path, ".")
	var current interface{} = map[string]json.RawMessage(d.Raw)
	for i, part := range parts {
		switch m := current.(type) {
		case map[string]json.RawMessage:
			raw, ok := m[part]
			if !ok {
				return false
			}
			// Last segment — exists
			if i == len(parts)-1 {
				return true
			}
			var nested map[string]json.RawMessage
			if err := json.Unmarshal(raw, &nested); err == nil {
				current = nested
			} else {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// GetNestedValue traverses the GenieACS nested JSON structure using a dot-separated path.
// e.g. "InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPPPConnection.1.Username"
func (d *Device) GetNestedValue(path string) interface{} {
	if d.Raw == nil {
		return nil
	}
	parts := strings.Split(path, ".")
	var current interface{} = map[string]json.RawMessage(d.Raw)
	for _, part := range parts {
		switch m := current.(type) {
		case map[string]json.RawMessage:
			raw, ok := m[part]
			if !ok {
				return nil
			}
			// Try to unmarshal as nested object first
			var nested map[string]json.RawMessage
			if err := json.Unmarshal(raw, &nested); err == nil {
				current = nested
			} else {
				// Try as Param
				var p Param
				if err := json.Unmarshal(raw, &p); err == nil {
					return p.Value
				}
				return nil
			}
		case map[string]interface{}:
			v, ok := m[part]
			if !ok {
				return nil
			}
			current = v
		default:
			return nil
		}
	}
	// If we ended at a nested map, try to get _value
	if m, ok := current.(map[string]json.RawMessage); ok {
		if v, ok := m["_value"]; ok {
			var val interface{}
			if err := json.Unmarshal(v, &val); err == nil {
				return val
			}
		}
	}
	return nil
}

// UnmarshalJSON handles the flat GenieACS NBI device format where
// _deviceId is an object and TR-069 parameters are top-level keys.
func (d *Device) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	d.Raw = raw // store for nested traversal

	if v, ok := raw["_id"]; ok {
		json.Unmarshal(v, &d.ID)
	}
	if v, ok := raw["_tags"]; ok {
		json.Unmarshal(v, &d.Tags)
	}
	if v, ok := raw["_lastInform"]; ok {
		json.Unmarshal(v, &d.LastInform)
	}

	// Parse _deviceId object
	if v, ok := raw["_deviceId"]; ok {
		var deviceID struct {
			SerialNumber string `json:"_SerialNumber"`
			Manufacturer string `json:"_Manufacturer"`
			ProductClass string `json:"_ProductClass"`
		}
		if err := json.Unmarshal(v, &deviceID); err == nil {
			d.SerialNumber = deviceID.SerialNumber
			d.Manufacturer = deviceID.Manufacturer
			d.ProductClass = deviceID.ProductClass
		}
	}

	// Collect TR-069 parameters (keys containing a dot)
	d.Parameters = make(map[string]Param)
	skip := map[string]bool{"_id": true, "_tags": true, "_deviceId": true, "_lastInform": true, "_registered": true, "_lastBoot": true, "_lastBootstrap": true}
	for key, val := range raw {
		if skip[key] || len(key) == 0 || key[0] == '_' {
			continue
		}
		var p Param
		if err := json.Unmarshal(val, &p); err == nil && (p.Value != nil || p.Type != "") {
			d.Parameters[key] = p
		}
	}

	return nil
}

type Param struct {
	Value    interface{} `json:"_value"`
	Type     string      `json:"_type"`
	Writable bool        `json:"_writable"`
}

type Task struct {
	ID              string      `json:"_id,omitempty"`
	Name            string      `json:"name"`
	Device          string      `json:"device,omitempty"`
	Status          string      `json:"status,omitempty"`
	Parametervalues []TaskParam `json:"parameterValues,omitempty"`
	ParameterNames  []string    `json:"parameterNames,omitempty"`
	// Completed is true when GenieACS executed the task on the ONT (HTTP 200).
	// False means the task is queued (HTTP 202) — ONT didn't respond in time.
	Completed       bool        `json:"-"`
}

type TaskParam struct {
	Name  string
	Value interface{}
	Type  string
}

// MarshalJSON encodes TaskParam as a JSON array [name, value, type]
// as expected by the GenieACS NBI API.
func (p TaskParam) MarshalJSON() ([]byte, error) {
	return json.Marshal([]interface{}{p.Name, p.Value, p.Type})
}

// UnmarshalJSON decodes a JSON array [name, value, type] into TaskParam.
func (p *TaskParam) UnmarshalJSON(data []byte) error {
	var arr []interface{}
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}
	if len(arr) >= 1 {
		if v, ok := arr[0].(string); ok {
			p.Name = v
		}
	}
	if len(arr) >= 2 {
		p.Value = arr[1]
	}
	if len(arr) >= 3 {
		if v, ok := arr[2].(string); ok {
			p.Type = v
		}
	}
	return nil
}

func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	reqURL := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("genieacs: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.username != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("genieacs: request failed: %w", err)
	}
	return resp, nil
}

func (c *Client) ListDevices(ctx context.Context, query string) ([]Device, error) {
	path := "/devices"
	if query != "" {
		path += "?query=" + url.QueryEscape(query)
	}
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("genieacs: list devices returned %d", resp.StatusCode)
	}
	var devices []Device
	if err := json.NewDecoder(resp.Body).Decode(&devices); err != nil {
		return nil, fmt.Errorf("genieacs: decode devices: %w", err)
	}
	return devices, nil
}

func (c *Client) GetDevice(ctx context.Context, deviceID string) (*Device, error) {
	query := fmt.Sprintf(`{"_id":"%s"}`, deviceID)
	devices, err := c.ListDevices(ctx, query)
	if err != nil {
		return nil, err
	}
	if len(devices) == 0 {
		return nil, nil
	}
	return &devices[0], nil
}

func (c *Client) FindDeviceBySerialNumber(ctx context.Context, serialNumber string) (*Device, error) {
	query := fmt.Sprintf(`{"_deviceId._SerialNumber":"%s"}`, serialNumber)
	devices, err := c.ListDevices(ctx, query)
	if err != nil {
		return nil, err
	}
	if len(devices) == 0 {
		return nil, nil
	}
	return &devices[0], nil
}

func (c *Client) DeleteDevice(ctx context.Context, deviceID string) error {
	path := "/devices/" + url.PathEscape(deviceID)
	resp, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("genieacs: delete device returned %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) CreateTask(ctx context.Context, deviceID string, task Task, timeoutMs int) (*Task, error) {
	body, err := json.Marshal(task)
	if err != nil {
		return nil, fmt.Errorf("genieacs: marshal task: %w", err)
	}
	path := fmt.Sprintf("/devices/%s/tasks?timeout=%d", url.PathEscape(deviceID), timeoutMs)
	resp, err := c.doRequest(ctx, http.MethodPost, path, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("genieacs: create task returned %d: %s", resp.StatusCode, string(raw))
	}
	var result Task
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("genieacs: decode task response: %w", err)
	}
	// HTTP 200 = task completed (ONT responded). HTTP 202 = queued (ONT unreachable).
	result.Completed = resp.StatusCode == http.StatusOK
	return &result, nil
}

func (c *Client) ListTasks(ctx context.Context, query string) ([]Task, error) {
	path := "/tasks"
	if query != "" {
		path += "?query=" + url.QueryEscape(query)
	}
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("genieacs: list tasks returned %d", resp.StatusCode)
	}
	var tasks []Task
	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		return nil, fmt.Errorf("genieacs: decode tasks: %w", err)
	}
	return tasks, nil
}

func (c *Client) RefreshObject(ctx context.Context, deviceID, objectPath string) (*Task, error) {
	task := Task{
		Name:           "getParameterValues",
		ParameterNames: []string{objectPath},
	}
	return c.CreateTask(ctx, deviceID, task, 5000)
}

// RefreshObjects refreshes multiple parameter paths from the ONT in a single task.
// timeoutMs is how long GenieACS waits for the ONT to respond.
func (c *Client) RefreshObjects(ctx context.Context, deviceID string, objectPaths []string, timeoutMs int) (*Task, error) {
	task := Task{
		Name:           "getParameterValues",
		ParameterNames: objectPaths,
	}
	return c.CreateTask(ctx, deviceID, task, timeoutMs)
}

// RefreshSpecificParams fetches a list of specific parameter paths (leaf or object)
// from the ONT. Returns true if the ONT responded (data is realtime), false if queued.
func (c *Client) RefreshSpecificParams(ctx context.Context, deviceID string, params []string, timeoutMs int) bool {
	task, err := c.RefreshObjects(ctx, deviceID, params, timeoutMs)
	if err != nil {
		return false
	}
	return task.Completed
}

func (c *Client) SetParameterValues(ctx context.Context, deviceID string, params []TaskParam) (*Task, error) {
	task := Task{
		Name:            "setParameterValues",
		Parametervalues: params,
	}
	return c.CreateTask(ctx, deviceID, task, 5000)
}

func (c *Client) Reboot(ctx context.Context, deviceID string) (*Task, error) {
	task := Task{Name: "reboot"}
	return c.CreateTask(ctx, deviceID, task, 0)
}

func (c *Client) FactoryReset(ctx context.Context, deviceID string) (*Task, error) {
	task := Task{Name: "factoryReset"}
	return c.CreateTask(ctx, deviceID, task, 0)
}

func (c *Client) GetParameterValue(ctx context.Context, deviceID, paramName string) (interface{}, error) {
	_, err := c.RefreshObject(ctx, deviceID, paramName)
	if err != nil {
		return nil, err
	}
	device, err := c.GetDevice(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	if device == nil {
		return nil, fmt.Errorf("genieacs: device %s not found", deviceID)
	}
	if param, ok := device.Parameters[paramName]; ok {
		return param.Value, nil
	}
	return nil, nil
}

func (c *Client) AddTag(ctx context.Context, deviceID, tag string) error {
	path := fmt.Sprintf("/devices/%s/tags/%s", url.PathEscape(deviceID), url.PathEscape(tag))
	resp, err := c.doRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (c *Client) RemoveTag(ctx context.Context, deviceID, tag string) error {
	path := fmt.Sprintf("/devices/%s/tags/%s", url.PathEscape(deviceID), url.PathEscape(tag))
	resp, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
