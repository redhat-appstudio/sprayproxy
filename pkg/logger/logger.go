package logger

import (
	"log"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Get() *zap.Logger {
	config := zap.NewProductionConfig()
	config.DisableStacktrace = true
	config.InitialFields = map[string]any{"service": "sprayproxy"}
	config.EncoderConfig.EncodeTime = utcRFC3339TimeEncoder
	logger, err := config.Build()
	if err != nil {
		log.Fatalf("Failed to initialize zap logger: %v", err)
	}
	return logger
}

// custom encoder to log timestamp in UTC
func utcRFC3339TimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	zapcore.RFC3339TimeEncoder(t.UTC(), enc)
}
