package bot

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bagaking/goulp/jsonex"

	"github.com/bagaking/botheater/tool"
	"github.com/bagaking/goulp/wlog"
	"github.com/khicago/irr"
	"github.com/volcengine/volc-sdk-golang/service/maas/models/api/v2"
	client "github.com/volcengine/volc-sdk-golang/service/maas/v2"
)

type (
	BotConfig struct {
		Endpoint   string `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`
		PrefabName string `yaml:"prefab_name,omitempty" json:"prefab_name,omitempty"`

		Prompt *Prompt `yaml:"prompt,omitempty" json:"prompt,omitempty"`
	}

	Bot struct {
		*BotConfig

		chatHistory []*api.Message
		maas        *client.MaaS
		tm          *tool.ToolManager
	}
)

func New(conf BotConfig, maas *client.MaaS, tm *tool.ToolManager) *Bot {
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
		Role:    api.ChatRoleAssistant,
	}
	l := len(msgs)
	if l == 0 {
		return []*api.Message{mCall}
	}
	if msgs[l-1] == MSGContinue {
		msgs[l-1] = mCall
	}
	return msgs
}

var MSGContinue = &api.Message{
	Role:    api.ChatRoleUser,
	Content: "根据 function 调用结果，继续解决我的问题",
}

func (b *Bot) MakeSystemMessage() *api.Message {
	return b.Prompt.BuildSystemMessage(b.tm)
}

func (b *Bot) MakeUserMessage(question string) *api.Message {
	return &api.Message{
		Content: question,
		Role:    api.ChatRoleUser,
	}
}

func (b *Bot) EnqueueHistory(msg *api.Message) *Bot {
	b.chatHistory = append(b.chatHistory, msg)
	return b
}

func (b *Bot) CreateRequestFromHistory() *api.ChatReq {
	req := &api.ChatReq{
		Messages: b.CreateMessagesFromHistory(),
	}
	return req
}

func (b *Bot) CreateMessagesFromHistory() []*api.Message {
	messages := make([]*api.Message, 0, len(b.chatHistory)+1)
	messages = append(messages, b.MakeSystemMessage())
	messages = append(messages, b.chatHistory...)
	return messages
}

func (b *Bot) NormalReq(ctx context.Context, req *api.ChatReq, depth int) (*api.ChatResp, error) {
	log, ctx := wlog.ByCtxAndCache(ctx, "normal_req")
	log.Infof("| REQ >>> %s", req2Str(req))

	got, status, err := b.maas.Chat(b.Endpoint, req)
	if err != nil {
		errVal := &api.Error{}
		if errors.As(err, &errVal) { // the returned error always type of *api.Error
			log.WithError(errVal).Errorf("meet maas error, status= %d\n", status)
		}
		return nil, irr.Wrap(err, "normal req failed, depth= %d", depth)
	}

	log.Infof("| RESP <<< %s", jsonex.MustMarshalToString(got))

	for _, c := range got.Choices {
		if c.Message == nil || c.Message.Content == "" {
			continue
		}
		sContent, ok := c.Message.Content.(string)
		if !ok {
			continue
		}

		log.Infof("=== analysis %s", sContent)
		if tool.HasFunctionCall(sContent) {
			funcName, paramValues, err := tool.ParseFunctionCall(ctx, sContent)
			if err != nil {
				log.WithError(err).Errorf("failed to parse function call")
				continue
			}
			result, err := b.tm.Execute(ctx, funcName, paramValues)
			if err != nil {
				result = fmt.Sprintf("调用失败: %v", err)
			}
			// plainTXT := strings.Replace(sContent, fmt.Sprintf("func_call::%s(%s)", funcName, paramValues), "", -1)
			callResult := fmt.Sprintf("func_call::%s(%s) 调用的结果为: %s", funcName, strings.Join(paramValues, ","), jsonex.MustMarshalToString(result))

			b.chatHistory = PushFunctionCallMSG(req.Messages, callResult)
			req.Messages = PushFunctionCallMSG(req.Messages, callResult)
			req.Messages = append(req.Messages, MSGContinue)

			return b.NormalReq(ctx, req, depth+1)
		}
	}

	return got, nil
}

func (b *Bot) NormalChat(ctx context.Context, question string) (*api.ChatResp, error) {
	log := wlog.ByCtx(ctx, "normal_chat")
	msg := b.MakeUserMessage(question)
	req := b.EnqueueHistory(msg).CreateRequestFromHistory()
	got, err := b.NormalReq(ctx, req, 0)
	if err != nil {
		log.WithError(err).Error("normal chat failed")
	}
	return got, nil
}

func (b *Bot) StreamChat(ctx context.Context, question string, handle func(resp *api.ChatResp)) error {
	log := wlog.ByCtx(ctx, "stream_chat")

	msg := b.MakeUserMessage(question)
	req := b.EnqueueHistory(msg).CreateRequestFromHistory()

	log.Info("| REQ >>> %s", req2Str(req))

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
	data["history"] = b.chatHistory
	return jsonex.MustMarshalToString(data)
}

func req2Str(req *api.ChatReq) string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("REQUEST (%d)\n", len(req.Messages)))
	for i, msg := range req.Messages {
		sb.WriteString(msg2Str(i, msg))
	}
	return sb.String()
}

func msg2Str(ind int, msg *api.Message) string {
	return fmt.Sprintf("--- %d. [%s] --- \n%s\n--- %d. fin ---\n\n", ind, msg.Role, msg.Content, ind)
}
