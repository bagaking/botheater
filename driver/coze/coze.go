package coze

import (
	"context"
	"errors"
	"fmt"

	"github.com/bagaking/botheater/driver"
	"github.com/khicago/irr"
	client "github.com/volcengine/volc-sdk-golang/service/maas/v2"

	"github.com/bagaking/goulp/wlog"
	"github.com/khicago/got/util/typer"

	"github.com/volcengine/volc-sdk-golang/service/maas/models/api/v2"

	"github.com/bagaking/botheater/history"
)

type Driver struct {
	EndpointID string
	maas       *client.MaaS
}

var _ driver.Driver = new(Driver)

func New(maas *client.MaaS, endpointID string) *Driver {
	return &Driver{
		maas:       maas,
		EndpointID: endpointID,
	}
}

func (d *Driver) Chat(ctx context.Context, messages []*history.Message) (got string, err error) {
	log, ctx := wlog.ByCtxAndCache(ctx, "coze.chat")
	req := d.buildRequest(messages)
	d.debugStart(req, log, messages)

	resp, status, err := d.maas.Chat(d.EndpointID, req)
	if err != nil {
		errVal := &api.Error{}
		if errors.As(err, &errVal) { // the returned error always type of *api.Error
			log.WithError(errVal).Errorf("meet maas error, status= %d\n", status)
		}
		return "", irr.Wrap(err, "chat failed")
	}

	got = RespMsg2Str(resp)

	d.debugFinish(log, got, messages)

	// todo: deal with unexpected situations
	return got, nil
}

func (d *Driver) debugFinish(log wlog.Log, got string, messages []*history.Message) {
	log.Debugf("coze driver | RESP (len:%d, history:%d) <<<\b%s\n<<<", len(got), len(messages), got)
}

func (d *Driver) debugStart(req *api.ChatReq, log wlog.Log, messages []*history.Message) {
	reqStr := Req2Str(req)
	log.Debugf("coze driver | REQ (len:%d, history:%d) >>> %s", len(reqStr), len(messages), reqStr)
}

func (d *Driver) StreamChat(ctx context.Context, messages []*history.Message, handle func(got string)) error {
	log, ctx := wlog.ByCtxAndCache(ctx, "coze.stream")
	req := d.buildRequest(messages)
	d.debugStart(req, log, messages)

	ch, err := d.maas.StreamChatWithCtx(ctx, d.EndpointID, req)
	if err != nil {
		errVal := &api.Error{}
		if errors.As(err, &errVal) { // the returned error always type of *api.Error
			log.WithError(errVal).Errorf("meet maas error")
		}
		return irr.Wrap(err, "stream chat failed")
	}

	round := 0
	for resp := range ch {
		round++
		got := RespMsg2Str(resp)
		d.debugFinish(log, fmt.Sprintf("\t -- stream(%d) --\n%s", round, got), messages)
		handle(got)

	}
	return nil
}

func (d *Driver) buildRequest(messages []*history.Message) *api.ChatReq {
	req := &api.ChatReq{
		Messages: typer.SliceMap(messages, func(m *history.Message) *api.Message {
			return &api.Message{
				Name:    m.Identity,
				Content: m.Content,
				Role:    MappingRole(m.Role),
			}
		}),
	}
	return req
}

func MappingRole(role history.Role) api.ChatRole {
	switch role {
	case history.RoleBot:
		return api.ChatRoleAssistant
	case history.RoleUser:
		return api.ChatRoleUser
	case history.RoleSystem:
		return api.ChatRoleSystem
	}
	// todo: 兜底用啥?
	return api.ChatRoleFunction
}
