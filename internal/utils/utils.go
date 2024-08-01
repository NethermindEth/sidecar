package utils

import (
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
