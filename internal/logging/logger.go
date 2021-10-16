package logging

import (
	"context"
	"log"
	"os"
	"sync"
)

type contextKey string

const loggerKey = contextKey("logger")

var (
	defaultLogger     *log.Logger
	defaultLoggerOnce sync.Once
)

func DefaultLogger() *log.Logger {
	defaultLoggerOnce.Do(func() {
		defaultLogger = NewLogger("Gocy: ", log.Lmsgprefix)
	})
	return defaultLogger
}

func NewLogger(prefix string, flag int) *log.Logger {
	return log.New(os.Stderr, prefix, flag)
}

func WithLogger(ctx context.Context, logger *log.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

func FromContext(ctx context.Context) *log.Logger {
	if logger, ok := ctx.Value(loggerKey).(*log.Logger); ok {
		return logger
	}
	return DefaultLogger()
}
