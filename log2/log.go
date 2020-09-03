package log2

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ctxMarker struct{}

var (
	defaultLogger = zap.NewNop()
	ctxKey        = &ctxMarker{}
)

// New creates a new zap logger with the given log level, service name and environment.
func New(level zapcore.Level, serviceName, environment string) (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(level)
	config.DisableStacktrace = false
	config.Sampling = nil
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}
	config.InitialFields = map[string]interface{}{
		"service": serviceName,
		"env":     environment,
	}
	return config.Build()
}

// NewDiscard creates logger which output to ioutil.Discard.
// This can be used for testing.
func NewDiscard() *zap.Logger {
	return zap.NewNop()
}

// SetDefaultLogger sets the specified zap.Logger as default logger.
// By default no-op logger is used, so any logs does not output.
func SetDefaultLogger(logger *zap.Logger) {
	defaultLogger = logger
}

// FromContext returns zap.Logger from context.
//
// The logger is populated into the context by WithLoggingContext in a library
// like a gRPC interceptor with a request context.
func FromContext(ctx context.Context) *zap.Logger {
	l, ok := ctx.Value(ctxKey).(*zap.Logger)
	if ok {
		return l
	}

	return defaultLogger
}

// WithContext creates a new context with a new zap.Logger specified by argument.
// The is used for the complicated usecase of structured logging with context.
// Please see the example below.
//
// 	import zaplog "github.com/kouzoh/go-microservices-kit/logging/zap"
//
// 	func Foo(ctx context.Context) {
// 		logger := zaplog.FromContext(ctx)
// 		logger = logger.With(zap.String("foo", "bar")) // add another field to the logger
// 		ctx = zaplog.WithContext(ctx, logger)          // replace context with the logger
// 		Bar(ctx)                                       // call succeeding functions
// 	}
//
// 	func Bar(ctx context.Context) {
// 		logger := zaplog.FromContext(ctx)
// 		logger.Info("message")                         // This message contains `"foo":"bar"` when called by Foo
// 	}
func WithContext(ctx context.Context, l *zap.Logger) context.Context {
	return context.WithValue(ctx, ctxKey, l)
}

// WithFields adds zap.Fields to the zap.Logger in the context directly and create a new context with it.
// This is a convenient function of FromContext and WithContext.
func WithFields(ctx context.Context, fields ...zap.Field) context.Context {
	return WithContext(ctx, FromContext(ctx).With(fields...))
}
