package testutils

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewTestLogger() (*zap.Logger, error)  {
	loggerConfig := zap.NewDevelopmentConfig()
	loggerConfig.EncoderConfig.TimeKey = "timestamp"
	loggerConfig.Encoding = "json"
	loggerConfig.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("Jan 02 15:04:05.000000000")

	return loggerConfig.Build()
}
