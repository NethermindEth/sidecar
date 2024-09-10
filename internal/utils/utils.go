package utils

import (
	"encoding/hex"
	"fmt"
	"strings"
)

var (
	NullEthereumAddress    = "0000000000000000000000000000000000000000"
	NullEthereumAddressHex = fmt.Sprintf("0x%s", NullEthereumAddress)
)

func AreAddressesEqual(a, b string) bool {
	return strings.ToLower(a) == strings.ToLower(b)
}

func ConvertBytesToString(b []byte) string {
	return "0x" + hex.EncodeToString(b[:])
}

func StripLeading0x(s string) string {
	if len(s) < 2 {
		return s
	}
	if s[0:2] == "0x" {
		s = s[2:]
	}
	return s
}
