// driver/ollama/ollama.go
package ollama

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bagaking/goulp/wlog"
	"github.com/khicago/got/util/proretry"
	"github.com/khicago/irr"
	"github.com/ollama/ollama/api"

	"github.com/bagaking/botheater/driver"
	"github.com/bagaking/botheater/history"
	"github.com/bagaking/botheater/utils"
)

type Driver struct {
	client *api.Client
	model  string
}

var _ driver.Driver = new(Driver)

func New(client *api.Client, model string) *Driver {
	return &Driver{
		client: client,
		model:  model,
	}
}

// Chat implements driver.Driver, provide a chat interface to ollama
// @see https://pkg.go.dev/github.com/ollama/ollama/api#hdr-Examples
func (d *Driver) Chat(ctx context.Context, messages []*history.Message) (string, error) {
	log, ctx := wlog.ByCtxAndCache(ctx, "ollama.chat")
	req := d.buildRequest(messages)
	d.debugStart(req, log, len(messages))

	var got string

	err := proretry.Run(func() error {
		err := d.client.Chat(ctx, req, func(resp api.ChatResponse) error {
			got += resp.Message.Content
			return nil
		})
		if err != nil {
			return err
		}
		return nil
	}, 3,
		proretry.WithInitInterval(time.Second*2),
		proretry.WithBackoff(proretry.LinearBackoff(time.Second*2)),
	)
	if err != nil {
		if errors.Is(err, &proretry.RetryError{}) {
			log.WithError(err).Errorf("meet ollama error")
		}
		return "", irr.Wrap(err, "chat failed")
	}

	if got == "" {
		return "", irr.Error("got empty content")
	}

	d.debugFinish(log, got, len(messages))

	return got, nil
}

func (d *Driver) StreamChat(ctx context.Context, messages []*history.Message, handle func(got string)) error {
	log, ctx := wlog.ByCtxAndCache(ctx, "ollama.stream")
	req := d.buildRequest(messages)
	d.debugStart(req, log, len(messages))

	err := d.client.Chat(ctx, req, func(resp api.ChatResponse) error {
		got := resp.Message.Content
		d.debugFinish(log, got, len(messages))
		handle(got)
		return nil
	})
	if err != nil {
		return irr.Wrap(err, "stream chat failed")
	}

	return nil
}

func (d *Driver) buildRequest(messages []*history.Message) *api.ChatRequest {
	apiMessages := make([]api.Message, len(messages))
	for i, m := range messages {
		apiMessages[i] = api.Message{
			Role:    string(m.Role),
			Content: m.Content,
		}
	}

	req := &api.ChatRequest{
		Model:    "llama3.1", // 假设使用 llama3.1 模型
		Messages: apiMessages,
	}
	return req
}

func (d *Driver) debugFinish(log wlog.Log, got string, lenHistory int) {
	log.Debugf("\n%s\n",
		utils.SPrintWithFrameCard(
			fmt.Sprintf("ollama driver <<< RESP (len:%d, history:%d)", len(got), lenHistory),
			got, utils.PrintWidthL1, utils.StyTalk,
		),
	)
}

func (d *Driver) debugStart(req *api.ChatRequest, log wlog.Log, lenHistory int) {
	reqStr := Req2Str(req)
	log.Debugf("\n%s\n",
		utils.SPrintWithFrameCard(
			fmt.Sprintf("ollama driver >>> REQ (len:%d, history:%d)", len(reqStr), lenHistory),
			reqStr, utils.PrintWidthL1, utils.StyTalk,
		),
	)
}
