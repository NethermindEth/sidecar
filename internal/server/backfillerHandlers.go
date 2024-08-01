package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/Layr-Labs/sidecar/internal/backfiller"
	v1 "github.com/Layr-Labs/sidecar/protos/eigenlayer/blocklake/v1"
)

func (bs *BackfillServer) StartBackfill(ctx context.Context, req *v1.BackfillRequest) (*v1.BackfillResponse, error) {
	bfConfig := &backfiller.BackfillConfig{
		BackfillType: backfiller.BackfillType(req.GetBackfillType()),
		Reindex:      req.GetReindex(),
	}

	if bfConfig.BackfillType == backfiller.BackfillType_Range {
		bfConfig.BackfillRange = &backfiller.BackfillRange{
			From: req.GetRange().GetFrom(),
			To:   req.GetRange().GetTo(),
		}

		if bfConfig.BackfillRange.To == 0 {
			return nil, errors.New("Range 'to' is required")
		}
		if bfConfig.BackfillRange.From == 0 {
			return nil, errors.New("Range 'from' is required")
		}
		if bfConfig.BackfillRange.From > bfConfig.BackfillRange.To {
			return nil, errors.New("Range 'from' must be less than 'to'")
		}
	} else if bfConfig.BackfillType == backfiller.BackfillType_ReprocessLogsRange {
		bfConfig.BackfillRange = &backfiller.BackfillRange{
			From: req.GetRange().GetFrom(),
			To:   req.GetRange().GetTo(),
		}
	} else if bfConfig.BackfillType == backfiller.BackfillType_RestakedStrategiesRange {
		bfConfig.BackfillRange = &backfiller.BackfillRange{
			From: req.GetRange().GetFrom(),
			To:   req.GetRange().GetTo(),
		}
	}

	err := bs.Backfiller.Backfill(bfConfig)
	if err != nil {
		return nil, err
	}

	return &v1.BackfillResponse{}, nil
}

func (bs *BackfillServer) PurgeQueues(ctx context.Context, req *v1.PurgeQueuesRequest) (*v1.PurgeQueuesResponse, error) {
	conn, err := bs.Backfiller.RabbitMQ.Connect()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	bs.Backfiller.RabbitMQ.PurgeAllQueues()

	return &v1.PurgeQueuesResponse{}, nil
}

func (bs *BackfillServer) IndexContracts(ctx context.Context, req *v1.IndexContractsRequest) (*v1.IndexContractsResponse, error) {
	requestRange := req.GetRange()
	if requestRange == nil {
		return nil, fmt.Errorf("Range is required")
	}
	backfillRange := &backfiller.BackfillRange{
		From: requestRange.GetFrom(),
		To:   requestRange.GetTo(),
	}

	err := bs.Backfiller.BackfillContracts(backfillRange)
	if err != nil {
		return nil, err
	}

	return &v1.IndexContractsResponse{}, nil
}

func (bs *BackfillServer) ReIndexTransactionsForContract(
	ctx context.Context,
	req *v1.ReIndexTransactionsForContractRequest,
) (*v1.ReIndexTransactionsForContractResponse, error) {
	err := bs.Backfiller.ReIndexTransactionsForContract(req.GetContractAddress())

	if err != nil {
		return nil, err
	}

	return &v1.ReIndexTransactionsForContractResponse{}, nil
}

func (bs *BackfillServer) ReIndexRestakedStrategies(
	ctx context.Context,
	req *v1.ReIndexRestakedStrategiesRequest,
) (*v1.ReIndexRestakedStrategiesResponse, error) {
	r := &backfiller.BackfillRange{
		From: req.GetRange().GetFrom(),
		To:   req.GetRange().GetTo(),
	}

	err := bs.Backfiller.BackfillRestakedStrategies(r, req.GetMod())

	if err != nil {
		return nil, err
	}

	return &v1.ReIndexRestakedStrategiesResponse{}, nil
}
