package log

import (
	"fmt"

	"go.uber.org/zap"
)

func NewLogger(debug bool) (*zap.Logger, error) {

	level := zap.InfoLevel

	if debug {
		level = zap.DebugLevel
	}

	loggerConfig := zap.Config{
		Level:             zap.NewAtomicLevelAt(level),
		Encoding:          "console",
		EncoderConfig:     zap.NewDevelopmentEncoderConfig(),
		DisableStacktrace: true,
		OutputPaths:       []string{"stderr"},
		ErrorOutputPaths:  []string{"stderr"},
	}

	return loggerConfig.Build()
}

func LogAndWrapError(err error, context string, args ...interface{}) error {
	if err == nil {
		return nil
	}

	contextMsg := fmt.Sprintf(context, args...)
	zap.S().Errorf("%s: %v", contextMsg, err)
	return fmt.Errorf("%s: %w", contextMsg, err)
}

func LogWarning(context string, args ...interface{}) {
	zap.S().Warnf(context, args...)
}
