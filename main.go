package main

import (
	"context"
	"encoding/json"
	"math/big"

	// "encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
)

func main() {
	//client, err := ethclient.Dial("wss://mainnet.infura.io/ws/v3/257f8c940c9b49e0b1ecf4f58dddadae")
	client, err := ethclient.Dial("ws://3.21.167.130:8546")

	if err != nil {
		panic(err)
	}

	latestBlock, err := client.BlockNumber(context.Background())

	if err != nil {
		log.Fatalln("Failed to get latest block number", err)
	}

	fmt.Printf("latest block: %+v\n", latestBlock)

	block, err := client.BlockByNumber(context.Background(), big.NewInt(int64(latestBlock)))
	if err != nil {
		log.Fatalln("Failed to fetch block", err)
	}

	fmt.Printf("Got block: %+v\n", block.Transactions()[0])

	tx, _, err := client.TransactionByHash(context.Background(), block.Transactions()[0].Hash())

	jsonTx, _ := json.MarshalIndent(tx, "", "\t")
	fmt.Printf("Transaction: %+v\n", string(jsonTx))

	//parser := parsers.NewParser(client)

	//_, err = parser.ParseBlockToProto(block)

	//
	// if err != nil {
	// 	log.Fatalln("Failed to unpack block", err)
	// }

	// jsonBlock, err := json.MarshalIndent(parsedBlock, "", "\t")

	// fmt.Printf("JSON block: %+v\n", string(jsonBlock))

	//subch := make(chan types.Block)
	//
	//go func() {
	//	for i := 0; ; i++ {
	//		if i > 0 {
	//			time.Sleep(2 * time.Second)
	//		}
	//		subscribeToBlocks(client, subch)
	//	}
	//}()
	//
	//for block := range subch {
	//	fmt.Printf("latest block: %+v\n", block.Hash)
	//}
}
