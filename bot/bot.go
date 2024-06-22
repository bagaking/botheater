package bot

import (
	"context"
	"encoding/base64"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/bagaking/botheater/utils"
	"github.com/google/uuid"

	"github.com/bagaking/botheater/driver"
	"github.com/bagaking/goulp/jsonex"
	"github.com/bagaking/goulp/wlog"
	"github.com/khicago/irr"

	"github.com/bagaking/botheater/call/tool"
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
		UUID string `yaml:"uuid" json:"uuid"`
		*Config

		driver driver.Driver
		tm     *tool.Manager

		// localHistory 用于跨任务记忆，目前没在用
		// runtime 的解决，目前看临时 history 就够了
		localHistory *history.History
	}
)

func New(conf Config, driver driver.Driver, tm *tool.Manager) *Bot {
	bot := &Bot{
		Config:       &conf,
		driver:       driver,
		tm:           tm,
		localHistory: history.NewHistory(),
		UUID:         base64.StdEncoding.EncodeToString([]byte(uuid.New().String())),
	}
	return bot
}

func (b *Bot) Logger(ctx context.Context, funcName string) (*logrus.Entry, context.Context) {
	ctx = utils.InjectAgentLogKey(ctx, b.PrefabName)
	ctx = utils.InjectAgentID(ctx, b.UUID)
	log, ctx := wlog.ByCtxAndCache(ctx, funcName)
	return log.Entry, ctx
}

func (b *Bot) MakeSystemMessage(ctx context.Context) *history.Message {
	ctx = utils.InjectAgentLogKey(ctx, b.PrefabName)
	return b.Prompt.BuildSystemMessage(ctx, b.tm).AppendContent(b.ActAsContext)
}

func (b *Bot) Messages(ctx context.Context, globalHistory *history.History) history.Messages {
	ctx = utils.InjectAgentLogKey(ctx, b.PrefabName)
	// 创建这次交互的上下文，依次是 prompt、全局 history、本地 history
	messages := make([]*history.Message, 0, globalHistory.Len()+1)
	messages = append(messages, b.MakeSystemMessage(ctx))
	messages = append(messages, globalHistory.All()...)
	messages = append(messages, b.localHistory.All()...)
	return messages
}

// NormalReq 递归结构，会处理函数调用，不会改变 History
func (b *Bot) NormalReq(ctx context.Context, staticMessages history.Messages, tempMessages *history.Messages, depth int) (string, error) {
	log, ctx := b.Logger(ctx, "normal_req")

	if tempMessages == nil {
		tl := make(history.Messages, 0)
		tempMessages = &tl
	}

	got, err := b.driver.Chat(ctx, staticMessages)
	if err != nil {
		return "", irr.Wrap(err, "normal req failed, depth= %d", depth)
	}

	got = strings.TrimSpace(got)
	if got == "" {
		return b.PrefabName + " 开小差了，请重试", nil
	}

	// 如果没有后续的函数调用就直接返回
	if !tool.Caller.HasCall(got) {
		log.Infof("-- 在 depth= %d 的调用中，agent 不调用任何 Function 直接给出响应：`%s`", depth, got)
		return got, nil
	}

	// 还有函数调用则进入递归 todo: 处理一次有多个的情况
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

	// 在临时栈中操作
	*tempMessages = history.PushFunctionCallMSG(*tempMessages, callResult)
	log.Infof("--> call %s(%v) result %s", funcName, paramValues, callResult)
	return b.NormalReq(ctx, staticMessages, tempMessages, depth+1)
}

// Question - 只是一个和 bot 聊天的快捷方式
func (b *Bot) Question(ctx context.Context, h *history.History, question string) (string, error) {
	h.EnqueueUserMsg(question)
	return b.SendChat(ctx, h)
}

func (b *Bot) SendChat(ctx context.Context, globalHistory *history.History) (string, error) {
	log, ctx := b.Logger(ctx, "send_chat")
	// 创建临时聊天队列
	messages := b.Messages(ctx, globalHistory)
	tempMessages := make(history.Messages, 0) // 要保存结果就在调用前创建，不保存的话就是不保留 function 调用过程的模式
	got, err := b.NormalReq(ctx, messages, &tempMessages, 0)
	if err != nil {
		log.WithError(err).Error("normal chat failed")
	}
	// 最终结果返回，由外部决定是否组装到全局历史中
	return got, nil
}

func (b *Bot) String() string {
	data := make(map[string]any)
	data["conf"] = b.Config
	return jsonex.MustMarshalToString(data)
}
