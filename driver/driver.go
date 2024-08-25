package driver

import (
	"context"

	"github.com/bagaking/botheater/history"
)

type (
	Config struct {
		Driver   string `yaml:"driver,omitempty" json:"driver,omitempty"`
		Endpoint string `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`
	}

	Driver interface {
		Chat(ctx context.Context, messages []*history.Message) (resp string, err error)
		StreamChat(ctx context.Context, messages []*history.Message, handle func(got string)) error
	}
)
