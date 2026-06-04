package service

import (
	"testing"

	"github.com/candrasyahputra/radius-server/internal/model"
)

func TestUpdateLastKnownWiFiFromSetRequestStoresLastSentPassword(t *testing.T) {
	ont := &model.ONT{}

	changed := updateLastKnownWiFiFromSetRequest(ont, "BINJAI-NET", "rahasia123", "11i")
	if !changed {
		t.Fatalf("expected wifi cache to change")
	}
	if ont.LastKnownWiFiPassword == nil || *ont.LastKnownWiFiPassword != "rahasia123" {
		t.Fatalf("last sent password not stored: %#v", ont.LastKnownWiFiPassword)
	}
	if ont.LastKnownWiFiSource == nil || *ont.LastKnownWiFiSource != "last_sent" {
		t.Fatalf("unexpected wifi source: %#v", ont.LastKnownWiFiSource)
	}
}

func TestApplyCachedWiFiPasswordUsesDatabaseValue(t *testing.T) {
	password := "rahasia123"
	ssid := "BINJAI-NET"
	security := "11i"
	source := "last_sent"

	ont := &model.ONT{
		LastKnownWiFiPassword: &password,
		LastKnownWiFiSSID:     &ssid,
		LastKnownWiFiSecurity: &security,
		LastKnownWiFiSource:   &source,
	}
	info := &DeviceInfo{
		WiFiSSID:     ssid,
		WiFiSecurity: security,
	}

	if ok := applyCachedWiFiPassword(ont, info); !ok {
		t.Fatalf("expected cached wifi password to be applied")
	}
	if info.WiFiPassword != password {
		t.Fatalf("unexpected wifi password: %q", info.WiFiPassword)
	}
	if info.WiFiPasswordReported {
		t.Fatalf("cached password must not be marked as reported")
	}
	if info.WiFiPasswordSource != "last_sent" {
		t.Fatalf("unexpected wifi password source: %q", info.WiFiPasswordSource)
	}
}
