package bot

import (
	"context"
	"encoding/base64"
	"fmt"
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
func (b *Bot) NormalReq(ctx context.Context, mergedHistory history.Messages) (string, error) {
	log, ctx := b.Logger(ctx, "normal_req")

	got, err := b.driver.Chat(ctx, mergedHistory)
	if err != nil {
		return "", irr.Wrap(err, "normal req failed")
	}

	got = strings.TrimSpace(got)
	if got == "" {
		return b.PrefabName + " 开小差了，请重试", nil
	}

	tempMessages := make(history.Messages, 0) // 创建函数调用过程的临时队列
	log.Debugf("try execute functions")
	got, err = b.ExecuteFunctions(ctx, mergedHistory, &tempMessages, got, 0)
	if err != nil {
		return "", irr.Wrap(err, "execute functions failed")
	}
	if b.Config.Prompt.FunctionMode == FunctionModeSampleOnly {
		summarize, err := b.Summarize(ctx, tempMessages)
		if err != nil {
			log.WithError(err).Warn("summarize failed")
		}
		got = fmt.Sprintf("#结论\n%s\n\n#过程\n%s\n", got, summarize)
	}

	return got, nil
}

func (b *Bot) ExecuteFunctions(ctx context.Context, historyBeforeFunctionCall history.Messages, tempMessages *history.Messages, trigger string, stackDepth int) (string, error) {
	log, ctx := b.Logger(ctx, "execute_functions")

	// 如果没有新的函数调用，则将 trigger返回，否则将 trigger 推入临时队列
	if !tool.Caller.HasCall(trigger) {
		// 如果没有后续的函数调用就 **直接返回**
		log.Info(
			utils.SPrintWithFrameCard(
				fmt.Sprintf("agent %s - %s，depth= %d, 不调用任何 Function 直接给出响应", b.PrefabName, b.UUID, stackDepth),
				trigger, 128, utils.SimpleStyle),
		)
		return trigger, nil
	} else {
		// 考虑 trigger 是否要包含在临时队列，目前看效果不错
		*tempMessages = history.PushFunctionCallMSG(*tempMessages, trigger)
	}

	// 还有函数调用则进入递归 todo: 处理一次有多个的情况
	funcName, paramValues, err := tool.Caller.ParseCall(ctx, trigger)
	functionReturns := ""
	if err != nil {
		log.WithError(err).Warnf("failed to parse function call")
		functionReturns = err.Error() + "，请检查后重试"
	} else {
		result := b.tm.Execute(ctx, funcName, paramValues)
		functionReturns = result.ToPrompt()
		// todo：要求错误修正的 prompt 在最终正确后可以去掉
	}

	// 将执行结果推入临时栈
	*tempMessages = history.PushFunctionCallMSG(*tempMessages, functionReturns) // 将函数调用结果推入临时队列

	req := append(historyBeforeFunctionCall, *tempMessages...)
	req = append(req, history.MSGFunctionContinue) // 注入驱动指令
	got, err := b.driver.Chat(ctx, req)
	if err != nil {
		return "", irr.Wrap(err, "function call failed, depth= %d", stackDepth)
	}

	log.Infof("\n%s\n", utils.SPrintWithCallStack(
		fmt.Sprintf("<-- function call stack [%d] --> %s(%v)", stackDepth, funcName, strings.Join(paramValues, ", ")),
		functionReturns, 120))

	return b.ExecuteFunctions(ctx, historyBeforeFunctionCall, tempMessages, got, stackDepth+1)
}

func (b *Bot) Summarize(ctx context.Context, messages2Summary history.Messages) (string, error) {
	log, ctx := b.Logger(ctx, "summarize")

	req := append(messages2Summary, history.MSGFunctionSummarize) // 注入驱动指令
	got, err := b.driver.Chat(ctx, req)
	if err != nil {
		return "", irr.Wrap(err, "summarize failed")
	}

	log.Infof("\n%s\n", utils.SPrintWithCallStack("<-- function summarize-->", got, 120))
	return got, nil
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
	got, err := b.NormalReq(ctx, messages)
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
