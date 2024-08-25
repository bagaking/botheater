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
		DriverConf driver.Config `yaml:",inline" json:",inline"`

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

		driver       driver.Driver
		tm           *tool.Manager
		argsReplacer map[string]any // 替换 prompt 中的占位符

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

// WithArgsReplacer 注入参数替换器，用于替换 prompt 中的占位符
func (b *Bot) WithArgsReplacer(argsReplacer map[string]any) *Bot {
	b.argsReplacer = argsReplacer
	return b
}

func (b *Bot) Logger(ctx context.Context, funcName string) (*logrus.Entry, context.Context) {
	ctx = utils.InjectAgentLogKey(ctx, b.PrefabName)
	ctx = utils.InjectAgentID(ctx, b.UUID)
	log, ctx := wlog.ByCtxAndCache(ctx, funcName)
	return log.Entry, ctx
}

func (b *Bot) MakeSystemMessage(ctx context.Context, appends ...string) *history.Message {
	ctx = utils.InjectAgentLogKey(ctx, b.PrefabName)
	msg := b.Prompt.
		BuildSystemMessage(ctx, b.tm, b.argsReplacer). // 注入 system prompt
		AppendContent(b.ActAsContext)
	for _, apd := range appends {
		msg = msg.AppendContent(apd)
	}
	return msg
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
	got, err = b.ExecuteFunctions(ctx, mergedHistory, got, &tempMessages)
	if err != nil {
		return "", irr.Wrap(err, "execute functions failed")
	}

	if len(tempMessages) <= 0 || b.Config.Prompt.FunctionMode != FunctionModeSampleOnly {
		return got, nil
	}

	// todo: 还是只在有函数的时候才做这个记录? 因为其他情况下都会回到原始上下文
	summarize, err := b.Summarize(ctx, append(tempMessages, history.NewBotMsg(got, b.PrefabName)))
	if err != nil {
		log.WithError(err).Warn("summarize failed")
	} else {
		got = fmt.Sprintf("# 结论\n%s\n\n# 过程\n%s\n", got, summarize) // todo: 测试中的机制, sample 模式下, 保留这些结论
		b.localHistory.Items = history.PushFunctionResultMSG(
			b.localHistory.Items,
			fmt.Sprintf("btw, 可以参考之前的结论: %s\n继续回答问题\n\n", got),
		)
	}
	return got, nil
}

func (b *Bot) ExecuteFunctions(ctx context.Context, historyBeforeFunctionCall history.Messages, trigger string, tempMessages *history.Messages) (string, error) {
	log, ctx := b.Logger(ctx, "E")
	// 如果没有新的函数调用，则将 trigger返回，否则将 trigger 推入临时队列
	if !tool.Caller.HasCall(trigger) {
		// 如果没有后续的函数调用就 **直接返回**
		log.Infof("\n%s",
			utils.SPrintWithFrameCard(
				fmt.Sprintf("agent %s response with no func calls", b.PrefabName),
				trigger, utils.PrintWidthL1, utils.StyNoFuncResult),
		)
		return trigger, nil
	}

	if tempMessages == nil {
		l := make(history.Messages, 0)
		tempMessages = &l
	}

	reqHistory := make(history.Messages, 0)
	if b.Config.Prompt.FunctionCtx == FunctionCtxAll {
		reqHistory = append(reqHistory, historyBeforeFunctionCall...)
	} else {
		reqHistory = append(reqHistory, b.MakeSystemMessage(ctx))

		introduce, err := b.Introduce(ctx, historyBeforeFunctionCall)
		if err != nil {
			log.WithError(err).Warn("introduce failed")
		} else {
			reqHistory = append(reqHistory, history.NewUserMsg(introduce, b.PrefabName))
		}
	}

	return b.executeFunctions(ctx, historyBeforeFunctionCall, tempMessages, trigger, 0)
}

func (b *Bot) executeFunctions(ctx context.Context, reqHistory history.Messages, tempMessages *history.Messages, funcCallMessage string, stackDepth int) (string, error) {
	log, ctx := b.Logger(ctx, fmt.Sprintf("ef-%d", stackDepth))

	// 考虑 trigger 是否要包含在临时队列，目前看效果不错
	//*tempMessages = append(*tempMessages, history.NewUserMsg(trigger, b.PrefabName))
	//history.PushFunctionResultMSG(*tempMessages, trigger) // 用 function 身份就看不懂需求了
	*tempMessages = append(*tempMessages, history.NewBotMsg(funcCallMessage, b.PrefabName))

	// 还有函数调用则进入递归 todo: 处理一次有多个的情况
	funcName, paramValues, err := tool.Caller.ParseCall(ctx, funcCallMessage)
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
	*tempMessages = history.PushFunctionResultMSG(*tempMessages, functionReturns) // 将函数调用结果推入临时队列

	req := append(make(history.Messages, 0), reqHistory...) // 注入当前历史
	req = append(req, *tempMessages...)                     // 注入临时指令
	req = append(req, history.MSGFunctionContinue)          // 注入驱动指令

	got, err := b.driver.Chat(ctx, req)
	if err != nil {
		return "", irr.Wrap(err, "function call failed, depth= %d", stackDepth)
	}

	log.Infof(
		utils.SPrintWithFrameCard(
			fmt.Sprintf("<-- function call stack --> %s(%v) [%d]", funcName, strings.Join(paramValues, ", "), stackDepth),
			functionReturns,
			utils.PrintWidthL1,
			utils.StyFunctionStack,
		),
	)

	// 如果没有后续的函数调用就 **直接返回**，否则使用 got 继续调用
	if !tool.Caller.HasCall(got) {
		return got, nil
	}
	log.WithField("stackDepth", stackDepth).Debugf("find function call, trigger= %s", got)

	return b.executeFunctions(ctx, reqHistory, tempMessages, got, stackDepth+1)
}

func (b *Bot) Summarize(ctx context.Context, messages2Summary history.Messages) (string, error) {
	log, ctx := b.Logger(ctx, "summarize")

	req := append(make(history.Messages, 0), b.MakeSystemMessage(ctx, `
# 补充说明
完成所有函数调用后，你会进行总结。
总结必须包括直接结论和足够多的支撑细节，不要遗漏关键信息。
总结时，你必须先回顾整个过程和结论，对其中错误的地方先进行修正，然后进行总结。
`))
	req = append(req, messages2Summary...)
	req = append(req, history.MSGFunctionSummarize) // 注入驱动指令
	got, err := b.driver.Chat(ctx, req)
	if err != nil {
		return "", irr.Wrap(err, "summarize failed")
	}

	log.Infof("\n%s\n", utils.SPrintWithCallStack("<-- function summarize-->", got, utils.PrintWidthL2))
	return got, nil
}

func (b *Bot) Introduce(ctx context.Context, historyMessages history.Messages) (string, error) {
	log, ctx := b.Logger(ctx, "introduce")

	req := append(historyMessages, history.MSGFunctionIntroduce) // 注入驱动指令
	got, err := b.driver.Chat(ctx, req)
	if err != nil {
		return "", irr.Wrap(err, "summarize failed")
	}

	log.Infof("\n%s\n", utils.SPrintWithCallStack("<-- function introduce -->", got, utils.PrintWidthL2))
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
