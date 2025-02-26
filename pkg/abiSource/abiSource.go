package abiSource

type AbiSource interface {
	FetchAbi(address string, bytecode string) (string, error)
}
