package log

import (
	"context"
	"fmt"

	"cloud.google.com/go/logging"
	"github.com/pkg/errors"
)

type (
	Logger interface {
		Info(payload interface{})
		Debug(payload interface{})
		Warn(payload interface{})
		Error(payload interface{})
		Close()
	}

	stackdriverLogger struct {
		Client *logging.Client
		Logger *logging.Logger
	}

	nopLogger struct{}

	standardLogger struct{}
)

var (
	defaultLogger Logger
)

func DefaultLogger() Logger {
	if defaultLogger == nil {
		defaultLogger = NewNopLogger()
	}
	return defaultLogger
}

func SetDefaultLogger(logger Logger) {
	defaultLogger = logger
}

func NewStackdriverLogger(ctx context.Context, projectID string, logID string) (Logger, error) {
	client, err := logging.NewClient(ctx, projectID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed logging.NewClient")
	}

	lg := client.Logger(logID)

	return &stackdriverLogger{
		Client: client,
		Logger: lg,
	}, nil
}

// NewStandardLogger creates logger logs to std/out
func NewStandardLogger() Logger {
	return &standardLogger{}
}

// NewNopLogger creates logger do nothing (useful for testing)
func NewNopLogger() Logger {
	return &nopLogger{}
}

func (s *stackdriverLogger) Info(payload interface{}) {
	s.Logger.Log(logging.Entry{
		Payload:  payload,
		Severity: logging.Info,
	})
}

func (s *stackdriverLogger) Debug(payload interface{}) {
	s.Logger.Log(logging.Entry{
		Payload:  payload,
		Severity: logging.Debug,
	})
}

func (s *stackdriverLogger) Warn(payload interface{}) {
	s.Logger.Log(logging.Entry{
		Payload:  payload,
		Severity: logging.Warning,
	})
}

func (s *stackdriverLogger) Error(payload interface{}) {
	s.Logger.Log(logging.Entry{
		Payload:  payload,
		Severity: logging.Error,
	})
}

func (s *stackdriverLogger) Close() {
	s.Client.Close()
}

func (s *nopLogger) Info(payload interface{}) {}

func (s *nopLogger) Debug(payload interface{}) {}

func (s *nopLogger) Warn(payload interface{}) {}

func (s *nopLogger) Error(payload interface{}) {}

func (s *nopLogger) Close() {}

func (s *standardLogger) Info(payload interface{}) {
	fmt.Printf("[Info] %+v\n", payload)
}

func (s *standardLogger) Debug(payload interface{}) {
	fmt.Printf("[Debug] %+v\n", payload)
}

func (s *standardLogger) Warn(payload interface{}) {
	fmt.Printf("[Warn] %+v\n", payload)
}

func (s *standardLogger) Error(payload interface{}) {
	fmt.Printf("[Error] %+v\n", payload)
}

func (s *standardLogger) Close() {}
