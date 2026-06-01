package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"
)

// RouterHeartbeatData holds the lightweight data fetched via SNMP for router liveness checks.
type RouterHeartbeatData struct {
	Identity string
	Uptime   string
	CPULoad  int
}

// PollRouterHeartbeat does a minimal 2-request SNMP poll against the given IP.
// It uses a 2-second timeout with zero retries to avoid blocking the worker.
// Returns an error if the router is unreachable or does not respond.
func (s *SNMPService) PollRouterHeartbeat(_ context.Context, ip, community string) (*RouterHeartbeatData, error) {
	if community == "" {
		community = "public"
	}
	client := &gosnmp.GoSNMP{
		Target:    ip,
		Port:      161,
		Community: community,
		Version:   gosnmp.Version2c,
		Timeout:   2 * time.Second,
		Retries:   0,
	}
	if err := client.Connect(); err != nil {
		return nil, err
	}
	defer client.Conn.Close()

	result := &RouterHeartbeatData{}

	// Request 1: sysUpTime + sysName (one round-trip)
	sysResp, err := client.Get([]string{oidSysUptime, oidSysName})
	if err != nil {
		return nil, err
	}
	for _, v := range sysResp.Variables {
		switch v.Name {
		case oidSysUptime:
			if ticks, ok := v.Value.(uint32); ok {
				result.Uptime = formatUptime(ticks)
			}
		case oidSysName:
			result.Identity = cleanSNMPString(snmpString(v))
		}
	}

	// Request 2: CPU load index 1 (best-effort, skip on error)
	cpuResp, err := client.Get([]string{oidHrProcessorLoad + ".1"})
	if err == nil {
		for _, v := range cpuResp.Variables {
			if val, ok := v.Value.(int); ok {
				result.CPULoad = val
			}
		}
	}

	return result, nil
}

type InterfaceBandwidth struct {
	Name   string `json:"name"`
	InBps  int64  `json:"in_bps"`
	OutBps int64  `json:"out_bps"`
}

func (s *SNMPService) GetRouterInterfaceBandwidth(ctx context.Context, ip string, community string, interfaceNames []string) ([]InterfaceBandwidth, error) {
	if community == "" {
		community = "public"
	}

	client := s.newSNMPClient(ip, community)
	if err := client.Connect(); err != nil {
		return []InterfaceBandwidth{}, fmt.Errorf("connect snmp: %w", err)
	}
	defer client.Conn.Close()

	sample1, err := s.pollInterfaceCounters(client, interfaceNames)
	if err != nil {
		return []InterfaceBandwidth{}, err
	}

	time.Sleep(1 * time.Second)

	sample2, err := s.pollInterfaceCounters(client, interfaceNames)
	if err != nil {
		return []InterfaceBandwidth{}, err
	}

	results := make([]InterfaceBandwidth, 0)
	for _, name := range interfaceNames {
		s2, ok2 := sample2[name]
		s1, ok1 := sample1[name]
		if ok1 && ok2 {
			inBps := int64(s2.In - s1.In) * 8
			outBps := int64(s2.Out - s1.Out) * 8
			results = append(results, InterfaceBandwidth{
				Name:   name,
				InBps:  inBps,
				OutBps: outBps,
			})
		}
	}

	return results, nil
}

type counters struct {
	In  uint64
	Out uint64
}

func (s *SNMPService) pollInterfaceCounters(client *gosnmp.GoSNMP, interfaceNames []string) (map[string]counters, error) {
	descResults, err := client.BulkWalkAll(".1.3.6.1.2.1.2.2.1.2")
	if err != nil {
		return nil, err
	}

	nameToIdx := make(map[string]int)
	for _, v := range descResults {
		name := cleanSNMPString(snmpString(v))
		idx := extractIfIndex(v.Name, ".1.3.6.1.2.1.2.2.1.2")
		nameToIdx[name] = idx
	}

	results := make(map[string]counters)
	for _, name := range interfaceNames {
		idx, ok := nameToIdx[name]
		if !ok {
			continue
		}

		oids := []string{
			fmt.Sprintf(".1.3.6.1.2.1.31.1.1.1.6.%d", idx),
			fmt.Sprintf(".1.3.6.1.2.1.31.1.1.1.10.%d", idx),
		}
		resp, err := client.Get(oids)
		if err != nil {
			continue
		}

		var c counters
		for _, v := range resp.Variables {
			if strings.HasPrefix(v.Name, ".1.3.6.1.2.1.31.1.1.1.6") {
				if val, ok := v.Value.(uint64); ok {
					c.In = val
				}
			} else if strings.HasPrefix(v.Name, ".1.3.6.1.2.1.31.1.1.1.10") {
				if val, ok := v.Value.(uint64); ok {
					c.Out = val
				}
			}
		}
		results[name] = c
	}

	return results, nil
}

func (s *SNMPService) GetRouterInterfaces(ctx context.Context, ip string, community string) ([]MonitorInterface, error) {
	if community == "" {
		community = "public"
	}

	client := s.newSNMPClient(ip, community)
	if err := client.Connect(); err != nil {
		return []MonitorInterface{}, fmt.Errorf("connect snmp: %w", err)
	}
	defer client.Conn.Close()

	ifaces, err := s.pollInterfaces(client)
	if err != nil {
		return []MonitorInterface{}, err
	}
	return ifaces, nil
}
