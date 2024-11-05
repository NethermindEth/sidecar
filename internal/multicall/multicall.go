package multicall

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

//go:embed multicallAbi.json
var multicallAbi string

type MultiCallMetaData[T interface{}] struct {
	Address      common.Address
	Data         []byte
	FunctionName string
	Deserialize  func([]byte) (*T, error)
}

type Multicall3Result struct {
	Success    bool
	ReturnData []byte
}

type TypedMulticall3Result[A any] struct {
	Success bool
	Value   A
	Error   error
}

type DeserializedMulticall3Result struct {
	Success bool
	Value   any
}

func (md *MultiCallMetaData[T]) Raw() RawMulticall {
	return RawMulticall{
		Address:      md.Address,
		Data:         md.Data,
		FunctionName: md.FunctionName,
		Deserialize: func(data []byte) (any, error) {
			res, err := md.Deserialize(data)
			return any(res), err
		},
	}
}

type RawMulticall struct {
	Address      common.Address
	Data         []byte
	FunctionName string
	Deserialize  func([]byte) (any, error)
}

type MulticallClient struct {
	Contract            *bind.BoundContract
	Address             common.Address
	ABI                 *abi.ABI
	Context             context.Context
	MaxBatchSize        uint64
	OverrideCallOptions *bind.CallOpts
	IgnoreErrors        bool
}

type ParamMulticall3Call3 struct {
	Target       common.Address
	AllowFailure bool
	CallData     []byte
}

type TMulticallClientOptions struct {
	OverrideContractAddress *common.Address
	MaxBatchSizeBytes       uint64
	OverrideCallOptions     *bind.CallOpts
	IgnoreErrors            bool
}

func panicIfError[T any](val T, err error) T {
	if err != nil {
		panic(err)
	}
	return val
}

/**
 * Initializes a multicall client. You'll need one of these to make any calls.
 *	ctx: network context for operations
 *	eth: the geth client to use for interacting with your node
 *	options [optional]: additional options to specify when making your request.
 *			- WARNING: these parameters take sceond precedence to any `overrideOptions` provided by the `Do*WithInfo()` functions.
 */
func NewMulticallClient(ctx context.Context, eth *ethclient.Client, options *TMulticallClientOptions) (*MulticallClient, error) {
	if eth == nil {
		return nil, errors.New("no ethclient passed")
	}

	// taken from: https://www.multicall3.com/
	parsed := panicIfError(abi.JSON(strings.NewReader(multicallAbi)))

	contractAddress := func() common.Address {
		if options == nil || options.OverrideContractAddress == nil {
			// also taken from: https://www.multicall3.com/ -- it's deployed at the same addr on most chains
			return common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")
		}
		return *options.OverrideContractAddress
	}()

	maxBatchSize := func() uint64 {
		if options == nil || options.MaxBatchSizeBytes == 0 {
			return 8192 // default batch size.
		} else {
			return options.MaxBatchSizeBytes
		}
	}()

	callOptions := func() *bind.CallOpts {
		if options != nil {
			return options.OverrideCallOptions
		}
		return nil
	}()

	return &MulticallClient{
		Address:             contractAddress,
		OverrideCallOptions: callOptions,
		MaxBatchSize:        maxBatchSize,
		Context:             ctx,
		ABI:                 &parsed,
		Contract:            bind.NewBoundContract(contractAddress, parsed, eth, eth, eth),
		IgnoreErrors:        options.IgnoreErrors,
	}, nil
}

func DescribeWithDeserialize[T any](contractAddress common.Address, abi abi.ABI, deserialize func([]byte) (*T, error), method string, params ...interface{}) (*MultiCallMetaData[T], error) {
	callData, err := abi.Pack(method, params...)
	if err != nil {
		return nil, fmt.Errorf("error packing multicall: %s", err.Error())
	}
	return &MultiCallMetaData[T]{
		Address:      contractAddress,
		Data:         callData,
		FunctionName: method,
		Deserialize:  deserialize,
	}, nil
}

// Describes a call that invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func Describe[T any](contractAddress common.Address, contractAbi abi.ABI, method string, params ...interface{}) (*MultiCallMetaData[T], error) {
	return DescribeWithDeserialize(
		contractAddress,
		contractAbi,
		func(b []byte) (*T, error) {
			res, err := contractAbi.Unpack(method, b)
			if err != nil {
				return nil, err
			}
			output, _ := abi.ConvertType(res[0], new(T)).(*T)
			return output, nil
		},
		method,
		params...,
	)
}

func Do[A any, B any](mc *MulticallClient, a *MultiCallMetaData[A], b *MultiCallMetaData[B]) (*A, *B, error) {
	res, err := doMultiCallMany(mc, nil, a.Raw(), b.Raw())
	if err != nil {
		return nil, nil, fmt.Errorf("error performing multicall: %s", err.Error())
	}
	return any(res[0].Value).(*A), any(res[1].Value).(*B), nil
}

func Do3[A any, B any, C any](mc *MulticallClient, a *MultiCallMetaData[A], b *MultiCallMetaData[B], c *MultiCallMetaData[C]) (*A, *B, *C, error) {
	res, err := doMultiCallMany(mc, nil, a.Raw(), b.Raw(), c.Raw())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error performing multicall: %s", err.Error())
	}
	return any(res[0].Value).(*A), any(res[1].Value).(*B), any(res[2].Value).(*C), nil
}

func Do4[A any, B any, C any, D any](mc *MulticallClient, a *MultiCallMetaData[A], b *MultiCallMetaData[B], c *MultiCallMetaData[C], d *MultiCallMetaData[D]) (*A, *B, *C, *D, error) {
	res, err := doMultiCallMany(mc, nil, a.Raw(), b.Raw(), c.Raw(), d.Raw())
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("error performing multicall: %s", err.Error())
	}
	return any(res[0].Value).(*A), any(res[1].Value).(*B), any(res[2].Value).(*C), any(res[3].Value).(*D), nil
}

func Do5[A any, B any, C any, D any, E any](mc *MulticallClient, a *MultiCallMetaData[A], b *MultiCallMetaData[B], c *MultiCallMetaData[C], d *MultiCallMetaData[D], e *MultiCallMetaData[E]) (*A, *B, *C, *D, *E, error) {
	res, err := doMultiCallMany(mc, nil, a.Raw(), b.Raw(), c.Raw(), d.Raw(), e.Raw())
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("error performing multicall: %s", err.Error())
	}
	return any(res[0].Value).(*A), any(res[1].Value).(*B), any(res[2].Value).(*C), any(res[3].Value).(*D), any(res[4].Value).(*E), nil
}

func Do6[A any, B any, C any, D any, E any, F any](mc *MulticallClient, a *MultiCallMetaData[A], b *MultiCallMetaData[B], c *MultiCallMetaData[C], d *MultiCallMetaData[D], e *MultiCallMetaData[E], f *MultiCallMetaData[F]) (*A, *B, *C, *D, *E, *F, error) {
	res, err := doMultiCallMany(mc, nil, a.Raw(), b.Raw(), c.Raw(), d.Raw(), e.Raw(), f.Raw())
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("error performing multicall: %s", err.Error())
	}
	return any(res[0].Value).(*A), any(res[1].Value).(*B), any(res[2].Value).(*C), any(res[3].Value).(*D), any(res[4].Value).(*E), any(res[5].Value).(*F), nil
}

func DoMany[A any](mc *MulticallClient, requests ...*MultiCallMetaData[A]) (*[]*A, error) {
	return DoManyWithOptions(mc, nil, requests...)
}

func DoManyWithOptions[A any](mc *MulticallClient, options *bind.CallOpts, requests ...*MultiCallMetaData[A]) (*[]*A, error) {
	res, err := doMultiCallMany(mc, options, mapCollection(requests, func(mc *MultiCallMetaData[A], index uint64) RawMulticall {
		return mc.Raw()
	})...)
	if err != nil {
		return nil, fmt.Errorf("multicall failed: %s", err.Error())
	}

	anyFailures := filterCollection(res, func(cur DeserializedMulticall3Result) bool {
		return !cur.Success
	})
	if !mc.IgnoreErrors && len(anyFailures) > 0 {
		return nil, errors.New("1 or more calls failed")
	}

	unwoundResults := mapCollection(res, func(d DeserializedMulticall3Result, i uint64) *A {
		if mc.IgnoreErrors && !d.Success {
			return nil
		}
		// force these back to A
		return any(d.Value).(*A)
	})

	return &unwoundResults, nil
}

// ////////////////// Other transactions you can run at the same time as your multicall.
func (mc *MulticallClient) GetBalance(address common.Address) *MultiCallMetaData[big.Int] {
	call, _ := Describe[big.Int](
		mc.Address,
		*mc.ABI,
		"getEthBalance",
		address,
	)
	return call
}

func (mc *MulticallClient) GetBlockNumber() *MultiCallMetaData[big.Int] {
	call, _ := Describe[big.Int](
		mc.Address,
		*mc.ABI,
		"getBlockNumber",
	)
	return call
}

// //////////////////////
func DoManyAllowFailures[A any](mc *MulticallClient, requests ...*MultiCallMetaData[A]) (*[]TypedMulticall3Result[*A], error) {
	return DoManyAllowFailuresWithOptions(mc, nil, requests...)
}

func DoManyAllowFailuresWithOptions[A any](mc *MulticallClient, overrideOpts *bind.CallOpts, requests ...*MultiCallMetaData[A]) (*[]TypedMulticall3Result[*A], error) {
	res, err := doMultiCallMany(mc, overrideOpts, mapCollection(requests, func(mc *MultiCallMetaData[A], index uint64) RawMulticall {
		return mc.Raw()
	})...)
	if err != nil {
		return nil, fmt.Errorf("multicall failed: %s", err.Error())
	}

	// unwind results
	unwoundResults := mapCollection(res, func(d DeserializedMulticall3Result, i uint64) TypedMulticall3Result[*A] {
		val, ok := any(d.Value).(*A)
		if !ok {
			return TypedMulticall3Result[*A]{
				Value:   val,
				Success: false,
			}
		}

		return TypedMulticall3Result[*A]{
			Value:   val,
			Success: d.Success,
		}
	})
	return &unwoundResults, nil
}

func doMultiCallMany(mc *MulticallClient, overrideOpts *bind.CallOpts, calls ...RawMulticall) ([]DeserializedMulticall3Result, error) {
	typedCalls := make([]ParamMulticall3Call3, len(calls))
	for i, call := range calls {
		typedCalls[i] = ParamMulticall3Call3{
			Target:       call.Address,
			AllowFailure: true,
			CallData:     call.Data,
		}
	}

	// see if we need to chunk them now
	chunkedCalls := chunkCalls(typedCalls, int(mc.MaxBatchSize))
	var results = make([]interface{}, len(calls))
	var totalResults = 0

	callOptions := func() *bind.CallOpts {
		if overrideOpts != nil {
			return overrideOpts
		}
		if mc.OverrideCallOptions != nil {
			return mc.OverrideCallOptions
		}
		return nil
	}()

	chunkNumber := 1
	for _, multicalls := range chunkedCalls {
		var res []interface{}
		chunkNumber++
		err := mc.Contract.Call(callOptions, &res, "aggregate3", multicalls)
		if err != nil {
			return nil, fmt.Errorf("aggregate3 failed: %s", err)
		}

		multicallResults := *abi.ConvertType(res[0], new([]Multicall3Result)).(*[]Multicall3Result)
		for i := 0; i < len(multicallResults); i++ {
			results[totalResults+i] = multicallResults[i]
		}
		totalResults += len(multicallResults)
	}

	outputs := make([]DeserializedMulticall3Result, len(calls))
	for i, call := range calls {
		res := results[i].(Multicall3Result)
		if res.Success {
			if res.ReturnData != nil {
				val, err := call.Deserialize(res.ReturnData)
				if err != nil {
					outputs[i] = DeserializedMulticall3Result{
						Value:   err,
						Success: false,
					}
				} else {
					outputs[i] = DeserializedMulticall3Result{
						Value:   val,
						Success: res.Success,
					}
				}
			} else {
				outputs[i] = DeserializedMulticall3Result{
					Value:   errors.New("no data returned"),
					Success: false,
				}
			}
		} else {
			outputs[i] = DeserializedMulticall3Result{
				Success: false,
				Value:   errors.New("call failed"),
			}
		}
	}

	return outputs, nil
}
