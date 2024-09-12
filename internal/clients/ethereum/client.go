package ethereum

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
	"golang.org/x/xerrors"
)

const (
	EIP1967_STORAGE_SLOT = "0x360894A13BA1A3210667C828492DB98DCA3E2076CC3735A920A3CA505D382BBC"
)

type RequestMethod struct {
	Name    string
	Timeout time.Duration
}

type RPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
	ID      uint   `json:"id"`
}

type RPCError struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
}

type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *uint           `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

type GetBlockByNumberResponse struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      uint           `json:"id"`
	Result  *EthereumBlock `json:"result"`
}

type ClientParams struct {
}

var jsonRPCVersion = "2.0"

type Client struct {
	BaseURL string
	Logger  *zap.Logger

	httpClient *http.Client
}

func NewClient(baseUrl string, l *zap.Logger) *Client {
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	l.Sugar().Debugw(fmt.Sprintf("Creating new Ethereum client with url '%s'", baseUrl))

	return &Client{
		BaseURL:    baseUrl,
		httpClient: client,
		Logger:     l,
	}
}

func (c *Client) GetWebsocketConnection(wsUrl string) (*ethclient.Client, error) {
	d, err := ethclient.Dial(wsUrl)
	if err != nil {
		return nil, err
	}

	return d, nil
}

func (c *Client) GetEthereumContractCaller() (*ethclient.Client, error) {
	d, err := ethclient.Dial(c.BaseURL)
	if err != nil {
		c.Logger.Sugar().Error("Failed to create new eth client", zap.Error(err))
		return nil, err
	}
	return d, nil
}

func (c *Client) ListenForNewBlocks(
	ctx context.Context,
	wsc *ethclient.Client,
	quitChan chan struct{},
	recvBlockHandler func(block *types.Header) error,
) error {
	ch := make(chan *types.Header)
	sub, err := wsc.SubscribeNewHead(ctx, ch)
	if err != nil {
		return err
	}

	defer close(ch)
	for {
		select {
		case err := <-sub.Err():
			c.Logger.Sugar().Errorw("Received error", zap.Error(err))
		case header := <-ch:
			recvBlockHandler(header)
		case <-quitChan:
			c.Logger.Sugar().Infow("Received quit")
			return nil
		}
	}
}

func (c *Client) buildUrlWithPath(path string) string {
	return fmt.Sprintf("%s%s", c.BaseURL, path)
}

func (c *Client) GetBlockNumber(ctx context.Context) (string, error) {
	res, err := c.Call(ctx, GetBlockRequest(1))

	if err != nil {
		return "", err
	}

	return strings.ReplaceAll(string(res.Result), "\"", ""), nil
}

func (c *Client) GetBlockNumberUint64(ctx context.Context) (uint64, error) {
	blockNumber, err := c.GetBlockNumber(ctx)
	if err != nil {
		return 0, err
	}

	blockNumberUint64, err := hexutil.DecodeUint64(blockNumber)
	if err != nil {
		return 0, err
	}

	return blockNumberUint64, nil
}

func (c *Client) GetBlockByNumber(ctx context.Context, blockNumber uint64) (*EthereumBlock, error) {
	rpcRequest := GetBlockByNumberRequest(blockNumber, 1)

	res, err := c.Call(ctx, rpcRequest)
	if err != nil {
		return nil, err
	}
	ethBlock, err := RPCMethod_getBlockByNumber.ResponseParser(res.Result)
	if err != nil {
		c.Logger.Sugar().Errorw("failed to parse block",
			zap.Error(err),
			zap.Any("raw response", res.Result),
		)
		return nil, err
	}
	return ethBlock, nil
}

func (c *Client) GetTransactionByHash(ctx context.Context, txHash string) (*EthereumTransaction, error) {
	rpcRequest := GetTransactionByHashRequest(txHash, 1)

	res, err := c.Call(ctx, rpcRequest)
	if err != nil {
		return nil, err
	}
	txReceipt, err := RPCMethod_getTransactionByHash.ResponseParser(res.Result)
	if err != nil {
		c.Logger.Sugar().Errorw("failed to parse transaction",
			zap.Error(err),
			zap.Any("raw response", res.Result),
		)
		return nil, err
	}
	return txReceipt, nil
}

func (c *Client) GetTransactionReceipt(ctx context.Context, txHash string) (*EthereumTransactionReceipt, error) {
	rpcRequest := GetTransactionReceiptRequest(txHash, 1)

	res, err := c.Call(ctx, rpcRequest)
	if err != nil {
		return nil, err
	}
	txReceipt, err := RPCMethod_getTransactionReceipt.ResponseParser(res.Result)
	if err != nil {
		c.Logger.Sugar().Errorw("failed to parse transaction receipt",
			zap.Error(err),
			zap.Any("raw response", res.Result),
		)
		return nil, err
	}
	return txReceipt, nil
}

func (c *Client) GetStorageAt(ctx context.Context, address string, storagePosition string, block string) (string, error) {
	rpcRequest := GetStorageAtRequest(address, storagePosition, block, 1)

	res, err := c.Call(ctx, rpcRequest)
	if err != nil {
		return "", err
	}
	storageValue, err := RPCMethod_getStorageAt.ResponseParser(res.Result)
	if err != nil {
		c.Logger.Sugar().Errorw("failed to get storage value",
			zap.Error(err),
			zap.Any("raw response", res.Result),
		)
		return "", err
	}
	return storageValue, nil
}

func (c *Client) GetCode(ctx context.Context, address string) (string, error) {
	rpcRequest := GetCodeRequest(address, 1)

	res, err := c.Call(ctx, rpcRequest)
	if err != nil {
		return "", err
	}
	bytecode, err := RPCMethod_getCode.ResponseParser(res.Result)
	if err != nil {
		c.Logger.Sugar().Errorw("failed to get contract bytecode",
			zap.Error(err),
			zap.Any("raw response", res.Result),
		)
		return "", err
	}
	return bytecode, nil
}

type BatchRPCRequest[T any] struct {
	Request *RPCRequest
	Handler ResponseParserFunc[T]
}

func (c *Client) batchCall(ctx context.Context, requests []*RPCRequest) ([]*RPCResponse, error) {
	if len(requests) == 0 {
		return make([]*RPCResponse, 0), nil
	}
	requestBody, err := json.Marshal(requests)
	if err != nil {
		return nil, xerrors.Errorf("Failed to marshal requests", err)
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL, bytes.NewReader(requestBody))
	if err != nil {
		return nil, xerrors.Errorf("Failed to make request", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, xerrors.Errorf("Request failed %v", err)
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, xerrors.Errorf("Failed to read body %v", err)
	}

	if response.StatusCode != http.StatusOK {
		return nil, xerrors.Errorf("received http error code %+v", response.StatusCode)
	}

	destination := []*RPCResponse{}

	if strings.HasPrefix(string(responseBody), "{") {
		errorResponse := RPCResponse{}
		if err := json.Unmarshal(responseBody, &errorResponse); err != nil {
			return nil, xerrors.Errorf("failed to unmarshal error response", err)
		}
		c.Logger.Sugar().Debugw("Error payload returned from batch call",
			zap.String("error", string(responseBody)),
		)
		if errorResponse.Error.Message != "empty batch" {
			return nil, xerrors.Errorf("Error payload returned from batch call", string(responseBody))
		}
	} else {
		if err := json.Unmarshal(responseBody, &destination); err != nil {
			c.Logger.Sugar().Errorw("failed to unmarshal batch call response",
				zap.Error(err),
				zap.String("response", string(responseBody)),
			)
			return nil, xerrors.Errorf("failed to unmarshal response", err)
		}
	}
	response.Body.Close()

	return destination, nil
}

const batchSize = 200

func (c *Client) BatchCall(ctx context.Context, requests []*RPCRequest) ([]*RPCResponse, error) {
	batches := [][]*RPCRequest{}

	currentIndex := 0
	for true {
		endIndex := currentIndex + batchSize
		if endIndex >= len(requests) {
			endIndex = len(requests)
		}
		batches = append(batches, requests[currentIndex:endIndex])
		currentIndex = currentIndex + batchSize
		if currentIndex >= len(requests) {
			break
		}
	}
	c.Logger.Sugar().Debugw(fmt.Sprintf("Batching '%v' requests into '%v' batches", len(requests), len(batches)))

	results := []*RPCResponse{}
	wg := sync.WaitGroup{}
	for i, batch := range batches {
		wg.Add(1)

		go func(b []*RPCRequest) {
			defer wg.Done()

			c.Logger.Sugar().Debugw(fmt.Sprintf("[batch %d] Fetching batch with '%d' requests", i, len(b)))
			res, err := c.batchCall(ctx, b)
			if err != nil {
				c.Logger.Sugar().Errorw("failed to batch call", zap.Error(err))
				return
			}
			c.Logger.Sugar().Debugw(fmt.Sprintf("[batch %d] Received '%d' results", i, len(res)))
			for _, r := range res {
				results = append(results, r)
			}
		}(batch)
	}
	wg.Wait()
	c.Logger.Sugar().Debugw(fmt.Sprintf("Received '%d' results", len(results)))
	return results, nil
}

func (c *Client) call(ctx context.Context, rpcRequest *RPCRequest) (*RPCResponse, error) {
	requestBody, err := json.Marshal(rpcRequest)

	c.Logger.Sugar().Debug("Request body", zap.String("requestBody", string(requestBody)))
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, RPCMethod_GetBlock.RequestMethod.Timeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL, bytes.NewReader(requestBody))
	if err != nil {
		return nil, xerrors.Errorf("Failed to make request", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, xerrors.Errorf("Request failed", err)
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, xerrors.Errorf("Failed to read body", err)
	}
	if response.StatusCode != http.StatusOK {
		return nil, xerrors.Errorf("received http error code %+v", response.StatusCode)
	}

	destination := &RPCResponse{}
	if err := json.Unmarshal(responseBody, destination); err != nil {
		return nil, xerrors.Errorf("failed to unmarshal response", err)
	}

	if destination.Error != nil {
		return nil, xerrors.Errorf("received error response: %+v", destination.Error)
	}

	response.Body.Close()

	return destination, nil
}

func (c *Client) Call(ctx context.Context, rpcRequest *RPCRequest) (*RPCResponse, error) {
	backoffs := []int{1, 3, 5, 10, 20, 30, 60}

	for _, backoff := range backoffs {
		res, err := c.call(ctx, rpcRequest)
		if err == nil {
			return res, nil
		}
		c.Logger.Sugar().Errorw("Failed to call", zap.Error(err), zap.Int("backoffSecs", backoff))
		time.Sleep(time.Second * time.Duration(backoff))
	}
	c.Logger.Sugar().Errorw("Exceeded retries for Call", zap.Any("rpcRequest", rpcRequest))
	return nil, xerrors.Errorf("Exceeded retries for Call")
}
