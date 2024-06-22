package bot

import (
	"context"
	"errors"

	"github.com/bagaking/goulp/jsonex"
	"github.com/bagaking/goulp/wlog"

	"github.com/bagaking/botheater/call/tool"
	"github.com/khicago/irr"

	"github.com/volcengine/volc-sdk-golang/service/maas/models/api/v2"
	client "github.com/volcengine/volc-sdk-golang/service/maas/v2"

	"github.com/bagaking/botheater/history"
)

type (
	Config struct {
		Endpoint string `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`

		PrefabName string `yaml:"prefab_name,omitempty" json:"prefab_name,omitempty"`
		Usage      string `yaml:"usage,omitempty" json:"usage,omitempty"`

		Prompt *Prompt `yaml:"prompt,omitempty" json:"prompt,omitempty"`

		// AckAs 表示这个 agent 的固有角色，用于支持多 Agent 模式
		// 根据不同的角色，调度系统将 1. 启用特殊流程 2. 注入信息到 prompt (类似于 function)
		AckAs        ActAs  `yaml:"ack_as,omitempty" json:"ack_as,omitempty"`
		ActAsContext string `yaml:"act_as_context,omitempty" json:"act_as_context,omitempty"`
	}

	Bot struct {
		*Config

		maas *client.MaaS
		tm   *tool.Manager

		localHistory *history.History
	}
)

func New(conf Config, maas *client.MaaS, tm *tool.Manager) *Bot {
	bot := &Bot{
		Config: &conf,
		maas:   maas,
		tm:     tm,
	}
	return bot
}

func (b *Bot) MakeSystemMessage(ctx context.Context) *api.Message {
	msg := b.Prompt.BuildSystemMessage(ctx, b.tm)

	if b.ActAsContext != "" {
		msg.Content = msg.Content.(string) + "\n\n" + b.ActAsContext
	}
	return msg
}

func (b *Bot) CreateRequestFromHistory(ctx context.Context, h *history.History) *api.ChatReq {

	// 创建这次交互的上下文，依次是 prompt、全局 history、本地 history
	messages := make([]*api.Message, 0, h.Len()+1)
	messages = append(messages, b.MakeSystemMessage(ctx))
	messages = append(messages, h.All()...)
	messages = append(messages, b.localHistory.All()...)

	req := &api.ChatReq{
		Messages: messages,
	}
	return req
}

func (b *Bot) NormalReq(ctx context.Context, h *history.History, req *api.ChatReq, depth int) (*api.ChatResp, error) {
	log, ctx := wlog.ByCtxAndCache(ctx, "normal_req")
	strReq := Req2Str(req)
	log.Infof("| REQ >>> (len:%d) %s", len(strReq), strReq)

	resp, status, err := b.maas.Chat(b.Endpoint, req)
	if err != nil {
		errVal := &api.Error{}
		if errors.As(err, &errVal) { // the returned error always type of *api.Error
			log.WithError(errVal).Errorf("meet maas error, status= %d\n", status)
		}
		return nil, irr.Wrap(err, "normal req failed, depth= %d", depth)
	}

	dataResp := jsonex.MustMarshalToString(resp)
	log.Infof("| RESP <<< (len:%d) %s", len(dataResp), dataResp)

	resp, err = b.TryHandleFunctionReq(ctx, h, req, resp, depth)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// TryHandleFunctionReq 如果有函数调用，则执行直到超出限制，否则返回结果
func (b *Bot) TryHandleFunctionReq(ctx context.Context, h *history.History, req *api.ChatReq, resp *api.ChatResp, depth int) (*api.ChatResp, error) {
	log := wlog.ByCtx(ctx, "handle_function_req")
	got := ""
	for _, c := range resp.Choices {
		if c.Message == nil || c.Message.Content == "" {
			continue
		}
		sContent, ok := c.Message.Content.(string)
		if !ok {
			continue
		}
		got = sContent
		break
	}

	if !tool.Caller.HasCall(got) {
		return resp, nil
	}

	// todo: 处理一次有多个的情况
	funcName, paramValues, err := tool.Caller.ParseCall(ctx, got)
	callResult := ""
	if err != nil {
		log.WithError(err).Warnf("failed to parse function call")
		callResult = err.Error() + "，请检查后重试"
	} else {
		result := b.tm.Execute(ctx, funcName, paramValues)
		callResult = result.ToPrompt()
		// todo：要求错误修正的 prompt 在最终正确后可以去掉
	}

	h.Items = history.PushFunctionCallMSG(h.Items, callResult)
	req.Messages = history.PushFunctionCallMSG(req.Messages, callResult)
	req.Messages = append(req.Messages, history.MSGContinue)

	return b.NormalReq(ctx, h, req, depth+1)
}

func (b *Bot) NormalChat(ctx context.Context, h *history.History, question string) (*api.ChatResp, error) {
	h.EnqueueUserMsg(question)
	return b.HistoryChat(ctx, h)
}

func (b *Bot) HistoryChat(ctx context.Context, h *history.History) (*api.ChatResp, error) {
	log := wlog.ByCtx(ctx, "history_chat")
	req := b.CreateRequestFromHistory(ctx, h)
	got, err := b.NormalReq(ctx, h, req, 0)
	if err != nil {
		log.WithError(err).Error("normal chat failed")
	}
	return got, nil
}

func (b *Bot) StreamChat(ctx context.Context, h *history.History, question string, handle func(resp *api.ChatResp)) error {
	log := wlog.ByCtx(ctx, "stream_chat")
	h.EnqueueUserMsg(question)
	req := b.CreateRequestFromHistory(ctx, h)

	log.Info("| REQ >>> %s", Req2Str(req))

	ch, err := b.maas.StreamChatWithCtx(ctx, b.Endpoint, req)
	if err != nil {
		errVal := &api.Error{}
		if errors.As(err, &errVal) { // the returned error always type of *api.Error
			log.WithError(errVal).Errorf("meet maas error")
		}
		return irr.Wrap(err, "stream chat failed")
	}

	for resp := range ch {
		handle(resp)
	}
	return nil
}

func (b *Bot) String() string {
	data := make(map[string]any)
	data["conf"] = b.Config
	return jsonex.MustMarshalToString(data)
}
