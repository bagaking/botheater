package bot

import (
	"context"
	"errors"

	"github.com/bagaking/goulp/jsonex"
	"github.com/bagaking/goulp/wlog"

	"github.com/khicago/got/util/typer"
	"github.com/khicago/irr"

	"github.com/volcengine/volc-sdk-golang/service/maas/models/api/v2"
	client "github.com/volcengine/volc-sdk-golang/service/maas/v2"

	"github.com/bagaking/botheater/history"
	"github.com/bagaking/botheater/tool"
)

type (
	BotConfig struct {
		Endpoint   string `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`
		PrefabName string `yaml:"prefab_name,omitempty" json:"prefab_name,omitempty"`

		Prompt *Prompt `yaml:"prompt,omitempty" json:"prompt,omitempty"`
	}

	Bot struct {
		*BotConfig

		maas *client.MaaS
		tm   *tool.Manager
	}
)

func New(conf BotConfig, maas *client.MaaS, tm *tool.Manager) *Bot {
	return &Bot{
		BotConfig: &conf,
		maas:      maas,
		tm:        tm,
	}
}

// PushFunctionCallMSG 如果队列中最后一个是 MSGContinue，则查到其之前
func PushFunctionCallMSG(msgs []*api.Message, callResult string) []*api.Message {
	mCall := &api.Message{
		Content: callResult,
		Name:    tool.CallPrefix,
		Role:    api.ChatRoleAssistant,
	}

	for len(msgs) > 0 && typer.SliceLast(msgs) == MSGContinue { // remote continue cmd
		msgs = msgs[:len(msgs)-1]
	}

	// todo: merge 规则可以调整
	for l := len(msgs); len(msgs) > 0 && msgs[l-1].Name == tool.CallPrefix; l = len(msgs) { // merge calls
		if str, ok := msgs[l-1].Content.(string); ok {
			mCall.Content = str + "\n\n" + callResult
		}
		msgs = msgs[:l-1]
	}
	return append(msgs, mCall)
}

var MSGContinue = &api.Message{
	Role:    api.ChatRoleUser,
	Content: "根据 function 调用结果，继续解决我的问题",
}

func (b *Bot) MakeSystemMessage() *api.Message {
	return b.Prompt.BuildSystemMessage(b.tm)
}

func (b *Bot) CreateRequestFromHistory(h *history.History) *api.ChatReq {
	req := &api.ChatReq{
		Messages: b.CreateMessagesFromHistory(h),
	}
	return req
}

func (b *Bot) CreateMessagesFromHistory(h *history.History) []*api.Message {
	messages := make([]*api.Message, 0, h.Len()+1)
	messages = append(messages, b.MakeSystemMessage())
	messages = append(messages, h.All()...)
	return messages
}

func (b *Bot) NormalReq(ctx context.Context, h *history.History, req *api.ChatReq, depth int) (*api.ChatResp, error) {
	log, ctx := wlog.ByCtxAndCache(ctx, "normal_req")
	log.Infof("| REQ >>> %s", Req2Str(req))

	resp, status, err := b.maas.Chat(b.Endpoint, req)
	if err != nil {
		errVal := &api.Error{}
		if errors.As(err, &errVal) { // the returned error always type of *api.Error
			log.WithError(errVal).Errorf("meet maas error, status= %d\n", status)
		}
		return nil, irr.Wrap(err, "normal req failed, depth= %d", depth)
	}

	log.Infof("| RESP <<< %s", jsonex.MustMarshalToString(resp))

	return b.TryHandleFunctionReq(ctx, h, req, resp, depth)
}

// TryHandleFunctionReq 如果有函数调用，则执行直到超出限制，否则返回结果
func (b *Bot) TryHandleFunctionReq(ctx context.Context, h *history.History, req *api.ChatReq, resp *api.ChatResp, depth int) (*api.ChatResp, error) {
	log, ctx := wlog.ByCtxAndCache(ctx, "handle_function_req")
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

	if !tool.HasFunctionCall(got) {
		return resp, nil
	}

	funcName, paramValues, err := tool.ParseFunctionCall(ctx, got)
	callResult := ""
	if err != nil {
		log.WithError(err).Warnf("failed to parse function call")
		callResult = err.Error() + "，请检查后重试"
	} else {
		result := b.tm.Execute(ctx, funcName, paramValues)
		callResult = result.ToPrompt()
		// todo：要求错误修正的 prompt 在最终正确后可以去掉
	}

	h.Items = PushFunctionCallMSG(h.Items, callResult)
	req.Messages = PushFunctionCallMSG(req.Messages, callResult)
	req.Messages = append(req.Messages, MSGContinue)

	return b.NormalReq(ctx, h, req, depth+1)
}

func (b *Bot) NormalChat(ctx context.Context, h *history.History, question string) (*api.ChatResp, error) {
	log := wlog.ByCtx(ctx, "normal_chat")

	h.EnqueueUserMsg(question)
	req := b.CreateRequestFromHistory(h)
	got, err := b.NormalReq(ctx, h, req, 0)
	if err != nil {
		log.WithError(err).Error("normal chat failed")
	}

	return got, nil
}

func (b *Bot) StreamChat(ctx context.Context, h *history.History, question string, handle func(resp *api.ChatResp)) error {
	log := wlog.ByCtx(ctx, "stream_chat")
	h.EnqueueUserMsg(question)
	req := b.CreateRequestFromHistory(h)

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
	data["conf"] = b.BotConfig
	return jsonex.MustMarshalToString(data)
}
