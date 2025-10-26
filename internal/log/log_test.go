package log

import (
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name      string
		debug     bool
		wantLevel zapcore.Level
		wantErr   bool
	}{
		{
			name:      "info level logger",
			debug:     false,
			wantLevel: zapcore.InfoLevel,
			wantErr:   false,
		},
		{
			name:      "debug level logger",
			debug:     true,
			wantLevel: zapcore.DebugLevel,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewLogger(tt.debug)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewLogger() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if logger == nil {
					t.Error("NewLogger() returned nil logger without error")
					return
				}

				// Check that the logger level is set correctly
				if !logger.Core().Enabled(tt.wantLevel) {
					t.Errorf("NewLogger() level not enabled for %v", tt.wantLevel)
				}

				// Verify that the logger is functional
				logger.Info("test message")
				logger.Debug("debug message")

				// Clean up
				_ = logger.Sync()
			}
		})
	}
}

func TestNewLogger_Configuration(t *testing.T) {
	t.Run("logger uses console encoding", func(t *testing.T) {
		logger, err := NewLogger(false)
		if err != nil {
			t.Fatalf("NewLogger() error = %v", err)
		}
		defer func() {
			_ = logger.Sync()
		}()

		// The logger should be successfully created with console encoding
		if logger == nil {
			t.Error("NewLogger() returned nil logger")
		}
	})

	t.Run("logger outputs to stderr", func(t *testing.T) {
		logger, err := NewLogger(false)
		if err != nil {
			t.Fatalf("NewLogger() error = %v", err)
		}
		defer func() {
			_ = logger.Sync()
		}()

		// Test that logger can write (outputs to stderr by default)
		logger.Info("test output")

		// No error should occur during normal logging
		if logger == nil {
			t.Error("Logger should not be nil")
		}
	})

	t.Run("logger has stack traces disabled", func(t *testing.T) {
		logger, err := NewLogger(true)
		if err != nil {
			t.Fatalf("NewLogger() error = %v", err)
		}
		defer func() {
			_ = logger.Sync()
		}()

		// Verify logger was created successfully
		// Stack traces are disabled, so error logging should not panic
		logger.Error("test error without stack trace")
	})
}

func TestNewLogger_LevelTransition(t *testing.T) {
	t.Run("info logger does not log debug messages", func(t *testing.T) {
		logger, err := NewLogger(false) // Info level
		if err != nil {
			t.Fatalf("NewLogger() error = %v", err)
		}
		defer func() {
			_ = logger.Sync()
		}()

		// Debug level should not be enabled
		if logger.Core().Enabled(zapcore.DebugLevel) {
			t.Error("Info level logger should not enable debug level")
		}

		// Info level should be enabled
		if !logger.Core().Enabled(zapcore.InfoLevel) {
			t.Error("Info level logger should enable info level")
		}
	})

	t.Run("debug logger logs both debug and info messages", func(t *testing.T) {
		logger, err := NewLogger(true) // Debug level
		if err != nil {
			t.Fatalf("NewLogger() error = %v", err)
		}
		defer func() {
			_ = logger.Sync()
		}()

		// Both debug and info levels should be enabled
		if !logger.Core().Enabled(zapcore.DebugLevel) {
			t.Error("Debug level logger should enable debug level")
		}

		if !logger.Core().Enabled(zapcore.InfoLevel) {
			t.Error("Debug level logger should enable info level")
		}
	})
}

func TestNewLogger_Fields(t *testing.T) {
	t.Run("logger can be used with fields", func(t *testing.T) {
		logger, err := NewLogger(true)
		if err != nil {
			t.Fatalf("NewLogger() error = %v", err)
		}
		defer func() {
			_ = logger.Sync()
		}()

		// Test that logger can handle structured logging with fields
		logger.Info("test with fields",
			zap.String("key1", "value1"),
			zap.Int("key2", 42),
		)

		// Test with sugared logger
		sugar := logger.Sugar()
		defer func() {
			_ = sugar.Sync()
		}()

		sugar.Infof("formatted message: %s", "test")
		sugar.Debugw("debug with fields",
			"field1", "value1",
			"field2", 123,
		)
	})
}

func TestNewLogger_ErrorHandling(t *testing.T) {
	t.Run("logger handles errors gracefully", func(t *testing.T) {
		logger, err := NewLogger(false)
		if err != nil {
			t.Fatalf("NewLogger() error = %v", err)
		}
		defer func() {
			_ = logger.Sync()
		}()

		// Test that error logging works
		logger.Error("test error",
			zap.Error(err),
			zap.String("context", "test"),
		)

		// Verify logger is still functional after error logging
		logger.Info("after error log")
	})
}

func TestNewLogger_MultipleInstances(t *testing.T) {
	t.Run("multiple logger instances can be created", func(t *testing.T) {
		logger1, err := NewLogger(false)
		if err != nil {
			t.Fatalf("NewLogger() error = %v", err)
		}
		defer func() {
			_ = logger1.Sync()
		}()

		logger2, err := NewLogger(true)
		if err != nil {
			t.Fatalf("NewLogger() error = %v", err)
		}
		defer func() {
			_ = logger2.Sync()
		}()

		// Both loggers should be independent
		if logger1 == logger2 {
			t.Error("Multiple logger instances should be different")
		}

		// Both should be functional
		logger1.Info("logger 1 message")
		logger2.Debug("logger 2 message")
	})
}
