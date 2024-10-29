package numbers

import "C"
import (
	"math/big"
)

// NewBig257 returns a new big.Int with a size of 257 bits
// This allows us to fully support math on uint256 numbers
// as well as int256 numbers used for EigenPods.
func NewBig257() *big.Int {
	return big.NewInt(257)
}
