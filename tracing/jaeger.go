package tracing

import (
	"log/slog"

	"github.com/opentracing/opentracing-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
)

var (
	downFn func() error
)

func Teardown() error {
	if downFn != nil {
		return downFn()
	}
	return nil
}

func SetUp() error {
	// Sample configuration for testing. Use constant sampling to sample every trace
	// and enable LogSpan to log every span via configured Logger.
	cfg, err := jaegercfg.FromEnv()
	if err != nil {
		return err
	}

	tracer, closer, err := cfg.NewTracer(
		jaegercfg.Logger(l{}),
	)
	downFn = closer.Close
	opentracing.SetGlobalTracer(tracer)
	return err
}

type l struct {
}

func (l) Error(msg string) {
	slog.Error(msg)
}

func (l) Infof(msg string, args ...interface{}) {
	slog.Info(msg, args...)
}
