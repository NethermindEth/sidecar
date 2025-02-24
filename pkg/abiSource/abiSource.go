package abiSource

import (
	"encoding/json"
)

type AbiSource interface {
	FetchAbi(address string, bytecode string) (string, error)
}

type Response struct {
	Output struct {
		ABI json.RawMessage `json:"abi"` // Use json.RawMessage to capture the ABI JSON
	} `json:"output"`
}
