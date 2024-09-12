package numbers

import "C"
import (
	"fmt"
	"github.com/shopspring/decimal"
	"math/big"
)

// NewBig257 returns a new big.Int with a size of 257 bits
// This allows us to fully support math on uint256 numbers
// as well as int256 numbers used for EigenPods.
func NewBig257() *big.Int {
	return big.NewInt(257)
}

// NumericMultiply take two huge numbers, stored as strings, and multiplies them
func NumericMultiply(a, b string) (string, error) {
	na, err := decimal.NewFromString(a)
	if err != nil {
		return "", err
	}
	nb, err := decimal.NewFromString(b)
	if err != nil {
		return "", err
	}

	return na.Mul(nb).String(), nil
}

func SubtractBig(a, b string) (string, error) {
	na, err := decimal.NewFromString(a)
	if err != nil {
		return "", err
	}
	nb, err := decimal.NewFromString(b)
	if err != nil {
		return "", err
	}

	return na.Sub(nb).String(), nil
}

func BigGreaterThan(a, b string) (bool, error) {
	na, err := decimal.NewFromString(a)
	if err != nil {
		return false, err
	}
	nb, err := decimal.NewFromString(b)
	if err != nil {
		return false, err
	}

	return na.GreaterThan(nb), nil
}

// CalcRawTokensPerDay calculates the raw tokens per day for a given amount and duration
// Returns the raw tokens per day in decimal format as a string
func CalcRawTokensPerDay(amountStr string, duration uint64) (string, error) {
	amount, err := decimal.NewFromString(amountStr)
	if err != nil {
		fmt.Printf("CalcRawTokensPerDay Error: %s\n", err)
		return "", err
	}

	rawTokensPerDay := amount.Div(decimal.NewFromFloat(float64(duration) / 86400))

	return rawTokensPerDay.String(), nil
}

// PostNileTokensPerDay calculates the tokens per day for post-nile rewards
// Simply truncates the decimal portion of the of the raw tokens per day
func PostNileTokensPerDay(tokensPerDay string) (string, error) {
	fmt.Printf("PostNileTokensPerDay: %s\n", tokensPerDay)
	tpd, err := decimal.NewFromString(tokensPerDay)
	if err != nil {
		fmt.Printf("PostNileTokensPerDay Error: %s\n", err)
		return "", err
	}

	return tpd.BigInt().String(), nil
}

// CalculateStakerProportion calculates the staker weight for a given staker and total weight
func CalculateStakerProportion(stakerWeightStr string, totalWeightStr string) (string, error) {
	stakerWeight, err := decimal.NewFromString(stakerWeightStr)
	if err != nil {
		return "", err
	}
	totalWeight, err := decimal.NewFromString(totalWeightStr)
	if err != nil {
		return "", err
	}

	res := ((stakerWeight.Div(totalWeight)).Mul(decimal.NewFromInt(1000000000000000))).
		Div(decimal.NewFromInt(1000000000000000)).
		Floor()
	return res.String(), nil
}

func CalculateStakerWeight(multiplier string, shares string) (string, error) {
	m, err := decimal.NewFromString(multiplier)
	if err != nil {
		return "", err
	}
	s, err := decimal.NewFromString(shares)
	if err != nil {
		return "", err
	}

	return m.Mul(s).String(), nil
}
