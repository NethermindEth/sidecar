package sidecar

import (
	"context"
	"errors"
	"fmt"
	"github.com/Layr-Labs/sidecar/internal/clients/ethereum"
	"github.com/Layr-Labs/sidecar/internal/config"
	"github.com/Layr-Labs/sidecar/internal/eigenState/stateManager"
	"github.com/Layr-Labs/sidecar/internal/pipeline"
	"github.com/Layr-Labs/sidecar/internal/rpcServer"
	"github.com/Layr-Labs/sidecar/internal/storage"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/cors"
	"go.uber.org/zap"
	"golang.org/x/xerrors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
	"net/http"
	"regexp"
	"time"
)

type SidecarConfig struct {
	GenesisBlockNumber uint64
}

type Sidecar struct {
	Logger         *zap.Logger
	Config         *SidecarConfig
	GlobalConfig   *config.Config
	Storage        storage.BlockStore
	Pipeline       *pipeline.Pipeline
	EthereumClient *ethereum.Client
	ShutdownChan   chan bool
}

func NewSidecar(
	cfg *SidecarConfig,
	gCfg *config.Config,
	s storage.BlockStore,
	p *pipeline.Pipeline,
	l *zap.Logger,
	ethClient *ethereum.Client,
) *Sidecar {
	return &Sidecar{
		Logger:         l,
		Config:         cfg,
		GlobalConfig:   gCfg,
		Storage:        s,
		Pipeline:       p,
		EthereumClient: ethClient,
		ShutdownChan:   make(chan bool),
	}
}

func (s *Sidecar) Start(ctx context.Context) {

	s.Logger.Info("Starting sidecar")

	s.StartIndexing(ctx)
	/*
		Main loop:

		- Get current indexed block
			- If no blocks, start from the genesis block
			- If some blocks, start from last indexed block
		- Once at tip, begin listening for new blocks
	*/
}

func isHealthCheckRoute(fullMethodName string) bool {
	r := regexp.MustCompile(`HealthCheck$`)

	return r.MatchString(fullMethodName)
}

func HttpLoggerMiddleware(next http.Handler, l *zap.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		healthRegex := regexp.MustCompile(`v1\/health$`)
		readyRegex := regexp.MustCompile(`v1\/ready$`)

		if !healthRegex.MatchString(r.URL.Path) && !readyRegex.MatchString(r.URL.Path) {
			l.Sugar().Infow("http_request",
				zap.String("system", "http"),
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Duration("duration", time.Since(start)),
			)
		}
	})
}

func (s *Sidecar) WithRpcServer(
	ctx context.Context,
	bs storage.BlockStore,
	sm *stateManager.EigenStateManager,
	gracefulShutdown chan bool,
) error {
	grpc_zap.ReplaceGrpcLoggerV2(s.Logger)

	opts := []grpc_zap.Option{
		grpc_zap.WithDecider(func(fullMethodName string, err error) bool {
			if err == nil && isHealthCheckRoute(fullMethodName) {
				return false
			}
			// by default everything else will be logged
			return true
		}),
	}

	grpcPort := s.GlobalConfig.RpcConfig.GrpcPort
	grpcLis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		s.Logger.Sugar().Errorw("Failed to listen", zap.Error(err), zap.Int("port", grpcPort))
		return xerrors.Errorf("failed to listen: %w", err)
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpc_ctxtags.UnaryServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
			grpc_zap.UnaryServerInterceptor(s.Logger, opts...),
		),
	)
	reflection.Register(grpcServer)
	mux := runtime.NewServeMux()

	_, err = rpcServer.NewRpcServer(ctx, grpcServer, mux, bs, sm, s.Logger)
	if err != nil {
		s.Logger.Sugar().Errorw("Failed to create rpc server", zap.Error(err))
		return err
	}

	ctx, cancelCtx := context.WithCancel(ctx)
	httpPort := s.GlobalConfig.RpcConfig.HttpPort
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", httpPort),
		Handler: cors.AllowAll().Handler(HttpLoggerMiddleware(mux, s.Logger)),
		BaseContext: func(listener net.Listener) context.Context {
			ctx = context.WithValue(ctx, "httpServer", listener.Addr().String())
			return ctx
		},
	}

	s.Logger.Sugar().Infow("Starting servers...",
		zap.Int("grpcPort", grpcPort),
		zap.Int("httpPort", httpPort),
	)
	go func() {
		s.Logger.Sugar().Infow("Starting HTTP server")
		err := httpServer.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			s.Logger.Sugar().Infow("HTTP server closed")
		} else {
			s.Logger.Sugar().Fatal("failed to serve", zap.Error(err))
		}
		cancelCtx()
	}()
	go func() {
		if err := grpcServer.Serve(grpcLis); err != nil {
			s.Logger.Sugar().Fatal("failed to serve reload server", zap.Error(err))
		}
	}()

	go func() {
		for {
			select {
			case <-gracefulShutdown:
				s.Logger.Sugar().Info("Shutting down servers")
				grpcServer.GracefulStop()
				httpServer.Shutdown(ctx)
			}
		}
	}()

	return nil
}
