package driver

import (
	"context"

	"github.com/bagaking/botheater/history"
)

type (
	Driver interface {
		Chat(ctx context.Context, messages []*history.Message) (resp string, err error)
		StreamChat(ctx context.Context, messages []*history.Message, handle func(got string)) error
	}
)
