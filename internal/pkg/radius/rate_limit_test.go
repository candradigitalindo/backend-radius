package radius

import (
	"bytes"
	"encoding/binary"
	"testing"

	"layeh.com/radius"
)

func TestRateLimitVendorAttributes(t *testing.T) {
	for _, vendor := range []string{"cisco", "ruijie", "juniper", "mikrotik", "vyos", "huawei"} {
		t.Run(vendor, func(t *testing.T) {
			packet := radius.New(radius.CodeAccessAccept, []byte("secret"))
			server := &Server{}
			server.applyRateLimit(packet, vendor, 10, 20, "", "Paket 20M")
			var filters, vsas []radius.Attribute
			for _, attr := range packet.Attributes {
				switch attr.Type {
				case 11:
					filters = append(filters, attr.Attribute)
				case 26:
					vsas = append(vsas, attr.Attribute)
				}
			}
			switch vendor {
			case "cisco":
				if len(filters) != 0 || len(vsas) != 2 {
					t.Fatalf("Cisco must receive two QoS AVPairs without an ACL Filter-Id: %#v", packet.Attributes)
				}
				for i, direction := range []string{"in", "out"} {
					want := EncodeVSA(CiscoVendorID, CiscoAttrAVPair, "ip:sub-qos-policy-"+direction+"=Paket_20M")
					if !bytes.Equal(vsas[i], want) {
						t.Fatalf("wrong Cisco %s policy: %v", direction, vsas[i])
					}
				}
			case "ruijie":
				if len(filters) != 0 || len(vsas) != 0 {
					t.Fatalf("Ruijie must not receive unimplemented rate attributes: %#v", packet.Attributes)
				}
			case "juniper":
				if len(filters) != 1 || string(filters[0]) != "Paket_20M" || len(vsas) != 0 {
					t.Fatalf("%s must receive its filter without Cisco VSAs: %#v", vendor, packet.Attributes)
				}
			case "huawei":
				if len(filters) != 0 || len(vsas) != 2 ||
					!bytes.Equal(vsas[0], EncodeVSAInt(HuaweiVendorID, HuaweiAttrInputAverageRate, 10_000_000)) ||
					!bytes.Equal(vsas[1], EncodeVSAInt(HuaweiVendorID, HuaweiAttrOutputAverageRate, 20_000_000)) {
					t.Fatalf("Huawei must receive bit/s rate attributes: %#v", packet.Attributes)
				}
			default:
				if len(filters) != 0 || len(vsas) != 1 || len(vsas[0]) < 6 || binary.BigEndian.Uint32(vsas[0][:4]) != MikroTikVendorID || vsas[0][4] != MikroTikAttrRateLimit || !bytes.HasPrefix(vsas[0][6:], []byte("10M/20M ")) {
					t.Fatalf("%s must retain its inline rate: %#v", vendor, packet.Attributes)
				}
			}
		})
	}
}
