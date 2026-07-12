package llm

import (
	"context"
	"errors"
	"time"
)

var ErrLLMClientNotConfigured = errors.New("llm client is not configured")

const defaultLLMTimeout = 30 * time.Second

func contextWithDefaultTimeout(ctx context.Context) (context.Context, context.CancelFunc) {

	if _, ok := ctx.Deadline(); ok {
		return context.WithCancel(ctx)
	}

	return context.WithTimeout(
		ctx,
		defaultLLMTimeout,
	)
}
