package proc

import (
	"context"
	"os"
	"os/signal"
)

func NewContext() context.Context {
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	return ctx
}
