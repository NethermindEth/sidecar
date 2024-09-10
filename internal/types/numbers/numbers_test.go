package numbers

import (
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func Test_numbers(t *testing.T) {
	t.Run("Test that big.Int can produce negative numbers", func(t *testing.T) {
		startingNum := big.Int{}
		startingNum.SetString("10", 10)

		amountToSubtract := big.Int{}
		amountToSubtract.SetString("20", 10)

		assert.Equal(t, "-10", amountToSubtract.Sub(&startingNum, &amountToSubtract).String())
	})
	t.Run("Test that big.NewInt(257) can produce negative numbers", func(t *testing.T) {
		startingNum := big.NewInt(257)
		startingNum.SetString("10", 10)

		amountToSubtract := big.NewInt(257)
		amountToSubtract.SetString("20", 10)

		assert.Equal(t, "-10", amountToSubtract.Sub(startingNum, amountToSubtract).String())
	})
	t.Run("Test that Big257 can produce negative numbers", func(t *testing.T) {
		startingNum, success := NewBig257().SetString("10", 10)
		assert.True(t, success)

		amountToSubtract, success := NewBig257().SetString("20", 10)
		assert.True(t, success)

		assert.Equal(t, "-10", amountToSubtract.Sub(startingNum, amountToSubtract).String())
	})
	t.Run("Test a really big number", func(t *testing.T) {
		startingNum, success := NewBig257().SetString("13389173346000000000000000", 10)
		assert.True(t, success)

		assert.Equal(t, "13389173346000000000000000", startingNum.String())

		amountToSubtract, success := NewBig257().SetString("20", 10)
		assert.True(t, success)

		assert.Equal(t, "13389173345999999999999980", amountToSubtract.Sub(startingNum, amountToSubtract).String())
	})
}
