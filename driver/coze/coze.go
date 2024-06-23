package coze

import (
	"context"
	"errors"
	"fmt"

	"github.com/bagaking/botheater/utils"
	"github.com/bagaking/goulp/wlog"
	"github.com/khicago/got/util/typer"
	"github.com/khicago/irr"
	"github.com/volcengine/volc-sdk-golang/service/maas/models/api/v2"
	client "github.com/volcengine/volc-sdk-golang/service/maas/v2"

	"github.com/bagaking/botheater/driver"
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
	d.debugStart(req, log, len(messages))

	resp, status, err := d.maas.Chat(d.EndpointID, req)
	if err != nil {
		errVal := &api.Error{}
		if errors.As(err, &errVal) { // the returned error always type of *api.Error
			log.WithError(errVal).Errorf("meet maas error, status= %d\n", status)
		}
		return "", irr.Wrap(err, "chat failed")
	}

	// todo: deal with unexpected situations
	if resp.Error != nil {
		return "", irr.Wrap(resp.Error, "response failed")
	}

	got = RespMsg2Str(resp)
	if got == "" { // 空白的情况抛出错误，由上游兜底
		return "", irr.Error("got empty content")
		// got = "got empty message，fallback to all:\n" + jsonex.MustMarshalToString(resp)
	}

	d.debugFinish(log, got, len(messages))

	return got, nil
}

func (d *Driver) debugFinish(log wlog.Log, got string, lenHistory int) {
	log.Debugf("\n%s\n",
		utils.SPrintWithFrameCard(
			fmt.Sprintf("coze driver <<< RESP (len:%d, history:%d)", len(got), lenHistory),
			got, 120,
		),
	)
}

func (d *Driver) debugStart(req *api.ChatReq, log wlog.Log, lenHistory int) {
	reqStr := Req2Str(req)
	log.Debugf("\n%s\n",
		utils.SPrintWithFrameCard(
			fmt.Sprintf("coze driver >>> REQ (len:%d, history:%d)", len(reqStr), lenHistory),
			reqStr, 120,
		),
	)
}

func (d *Driver) StreamChat(ctx context.Context, messages []*history.Message, handle func(got string)) error {
	log, ctx := wlog.ByCtxAndCache(ctx, "coze.stream")
	req := d.buildRequest(messages)
	d.debugStart(req, log, len(messages))

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
		d.debugFinish(log, fmt.Sprintf("\t -- stream(%d) --\n%s", round, got), len(messages))
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
