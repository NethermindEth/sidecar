package utils

import (
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

var (
	NullEthereumAddress    = "0000000000000000000000000000000000000000"
	NullEthereumAddressHex = fmt.Sprintf("0x%s", NullEthereumAddress)
)

func AreAddressesEqual(a, b string) bool {
	return strings.EqualFold(a, b)
}

func ConvertBytesToString(b []byte) string {
	return "0x" + hex.EncodeToString(b)
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

func SnakeCase(s string) string {
	notSnake := regexp.MustCompile(`[_-]`)
	return notSnake.ReplaceAllString(s, "_")
}
