package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(debug bool) *zap.Logger {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder

	config := zap.Config{
		Level:         zap.NewAtomicLevelAt(zap.InfoLevel),
		Development:   debug,
		Encoding:      "json",
		EncoderConfig: encoderConfig,
	}

	log, err := config.Build()
	if err != nil {
		panic(err)
	}
	defer log.Sync()

	return log
}
