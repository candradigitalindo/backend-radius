package radius

import (
	"encoding/binary"
	"fmt"
)

const MikroTikVendorID = 14988

// MikroTik VSA Attribute IDs
const (
	MikroTikAttrRateLimit   = 8
	MikroTikAttrRecvLimit   = 9
	MikroTikAttrXmitLimit   = 10
	MikroTikAttrGroup       = 11
	MikroTikAttrAddressList = 17
)

// Huawei VSA (vendor 2011). Rates are integers in bit/s.
//   Input  = traffic from subscriber → NAS (upload)
//   Output = traffic from NAS → subscriber (download)
const HuaweiVendorID = 2011

const (
	HuaweiAttrInputAverageRate  = 2
	HuaweiAttrOutputAverageRate = 5
)

// Cisco VSA (vendor 9). Cisco-AVPair carries free-form "key=value" strings.
const CiscoVendorID = 9

const CiscoAttrAVPair = 1

// EncodeVSAInt encodes a 32-bit integer Vendor-Specific Attribute (e.g. Huawei rates).
func EncodeVSAInt(vendorID uint32, attrType byte, value uint32) []byte {
	vendorAttrLen := byte(2 + 4)
	buf := make([]byte, 4+int(vendorAttrLen))
	binary.BigEndian.PutUint32(buf[0:4], vendorID)
	buf[4] = attrType
	buf[5] = vendorAttrLen
	binary.BigEndian.PutUint32(buf[6:], value)
	return buf
}

// EncodeVSA encodes a Vendor-Specific Attribute (VSA) for inclusion in a RADIUS packet.
func EncodeVSA(vendorID uint32, attrType byte, value string) []byte {
	valBytes := []byte(value)
	if len(valBytes) > 247 {
		panic(fmt.Sprintf("VSA value too long: %d bytes (max 247)", len(valBytes)))
	}
	vendorAttrLen := byte(2 + len(valBytes))

	buf := make([]byte, 4+int(vendorAttrLen))
	binary.BigEndian.PutUint32(buf[0:4], vendorID)
	buf[4] = attrType
	buf[5] = vendorAttrLen
	copy(buf[6:], valBytes)

	return buf
}
