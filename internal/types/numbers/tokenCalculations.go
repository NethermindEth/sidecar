package numbers

import (
	"errors"
	"fmt"
	"github.com/shopspring/decimal"
)

/*
#cgo darwin CFLAGS: -I/opt/homebrew/opt/python@3.12/Frameworks/Python.framework/Versions/3.12/include/python3.12 -I/Users/seanmcgary/Code/sidecar/sqlite-extensions
#cgo darwin LDFLAGS: -L/opt/homebrew/opt/python@3.12/Frameworks/Python.framework/Versions/3.12/lib -lpython3.12 -L/Users/seanmcgary/Code/sidecar/sqlite-extensions -lcalculations
#cgo darwin LDFLAGS: -Wl,-rpath,/Users/seanmcgary/Code/sidecar/sqlite-extensions
#include <stdlib.h>
#include "calculations.h"
*/
import "C"
import "unsafe"

func InitPython() error {
	if C.init_python() == 0 {
		return errors.New("failed to initialize python")
	}
	return nil
}

func FinalizePython() {
	C.finalize_python()
}

// CalculateAmazonStakerTokenRewards calculates the Amazon token rewards for a given staker proportion and tokens per day
// cast(staker_proportion * tokens_per_day AS DECIMAL(38,0))
func CalculateAmazonStakerTokenRewards(stakerProportion string, tokensPerDay string) (string, error) {
	cSp := C.CString(stakerProportion)
	cTpd := C.CString(tokensPerDay)
	defer C.free(unsafe.Pointer(cSp))
	defer C.free(unsafe.Pointer(cTpd))

	cResult := C._amazon_staker_token_rewards(cSp, cTpd)
	defer C.free(unsafe.Pointer(cResult))

	return C.GoString(cResult), nil
}

// CalculateNileStakerTokenRewards calculates the tokens to be rewarded for a given staker proportion and tokens per day
// (staker_proportion * tokens_per_day)::text::decimal(38,0)
func CalculateNileStakerTokenRewards(stakerProportion string, tokensPerDay string) (string, error) {
	cSp := C.CString(stakerProportion)
	cTpd := C.CString(tokensPerDay)
	defer C.free(unsafe.Pointer(cSp))
	defer C.free(unsafe.Pointer(cTpd))

	cResult := C._nile_staker_token_rewards(cSp, cTpd)
	defer C.free(unsafe.Pointer(cResult))

	return C.GoString(cResult), nil
}

// CalculatePostNileStakerTokenRewards calculates the tokens to be rewarded for a given staker proportion and tokens per day
// FLOOR(staker_proportion * tokens_per_day_decimal)
func CalculatePostNileStakerTokenRewards(stakerProportion string, tokensPerDay string) (string, error) {
	cSp := C.CString(stakerProportion)
	cTpd := C.CString(tokensPerDay)
	defer C.free(unsafe.Pointer(cSp))
	defer C.free(unsafe.Pointer(cTpd))

	cResult := C._staker_token_rewards(cSp, cTpd)
	defer C.free(unsafe.Pointer(cResult))

	return C.GoString(cResult), nil
}

// CalculateAmazonOperatorTokens calculates the operator payout portion for rewards (10% of total)
//
// cast(total_staker_operator_payout * 0.10 AS DECIMAL(38,0))
func CalculateAmazonOperatorTokens(totalStakerPayout string) (string, error) {
	tsp := C.CString(totalStakerPayout)
	defer C.free(unsafe.Pointer(tsp))

	cResult := C._amazon_operator_token_rewards(tsp)
	defer C.free(unsafe.Pointer(cResult))

	return C.GoString(cResult), nil
}

// CalculateNileOperatorTokens calculates the operator payout portion for rewards (10% of total)
//
// (total_staker_operator_payout * 0.10)::text::decimal(38,0)
func CalculateNileOperatorTokens(totalStakerPayout string) (string, error) {
	tsp := C.CString(totalStakerPayout)
	defer C.free(unsafe.Pointer(tsp))

	cResult := C._nile_operator_token_rewards(tsp)
	defer C.free(unsafe.Pointer(cResult))

	return C.GoString(cResult), nil
}

// CalculatePostNileOperatorTokens calculates the operator payout portion for rewards (10% of total)
//
// floor(total_staker_operator_payout * 0.10)
func CalculatePostNileOperatorTokens(totalStakerPayout string) (string, error) {
	tpd, err := decimal.NewFromString(totalStakerPayout)
	if err != nil {
		return "", err
	}

	return tpd.Mul(decimal.NewFromFloat(0.10)).Floor().String(), nil
}

// PreNileTokensPerDay calculates the tokens per day for pre-nile rewards, rounded to 15 sigfigs
//
// Not gonna lie, this is pretty annoying that it has to be this way, but in order to support backwards compatibility
// with the current/old rewards system where postgres was lossy, we have to do this.
func PreNileTokensPerDay(tokensPerDay string) (string, error) {
	fmt.Printf("PreNileTokensPerDay: %s\n", tokensPerDay)
	cTokens := C.CString(tokensPerDay)
	defer C.free(unsafe.Pointer(cTokens))

	fmt.Printf("Calling pre_nile_tokens_per_day\n")
	result := C._pre_nile_tokens_per_day(cTokens)
	fmt.Printf("Successfully called")
	defer C.free(unsafe.Pointer(result))

	resultStr := C.GoString(result)
	fmt.Printf("PreNileTokensPerDay Result: %+v\n", resultStr)
	return resultStr, nil
}
