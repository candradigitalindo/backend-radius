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
