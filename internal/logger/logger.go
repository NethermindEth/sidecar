package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LoggerConfig struct {
	Debug bool
}

func NewLogger(cfg *LoggerConfig, options ...zap.Option) (*zap.Logger, error) {
	mergedOptions := []zap.Option{
		zap.WithCaller(true),
	}
	copy(mergedOptions, options)

	c := zap.NewProductionConfig()
	c.EncoderConfig = zap.NewProductionEncoderConfig()
	c.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	if cfg.Debug {
		c.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	} else {
		c.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	return c.Build(mergedOptions...)
}
