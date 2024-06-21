package bot

import (
	"context"
	"fmt"

	"github.com/khicago/got/util/typer"
)

type ActAs string

const (
	ActAsCoordinator ActAs = "coordinator"
	ActAsEvaluator   ActAs = "evaluator"
)

// InitAllActAs 初始化所有 ActAs
func InitAllActAs(ctx context.Context, allBots ...*Bot) {
	configs := typer.SliceMap(allBots, func(from *Bot) *Config {
		return from.Config
	})
	for i := range allBots {
		b := allBots[i]
		// wlog.ByCtx(ctx, "InitAllActAs").Infof("bot %d.%s act_as= %s, conf=%v", i, b.PrefabName, b.AckAs, b.Config)
		if b.AckAs == ActAsCoordinator {
			b.InjectCoordinatorPrompt(configs)
			// wlog.ByCtx(ctx, "InitAllActAs").Infof("find coordinator at %s, with context %s", b.Config.PrefabName, b.ActAsContext)
		}
	}
}

const (
	ActAsTellStart = `# 现在支持了以下 Agents
`
	ActAsTellTail = `
当且仅当要使用 agents 时，回复 agents_call::name，比如：
agents_call::botheater_basic
注意:
- 要调用 agent 时不要回复除调用 agent 以外的内容
- 如果不需要调用 agent, 你的回复一定不要包含这种格式
`
)

// InjectCoordinatorPrompt 注入所有机器人的信息到 Coordinator 的 prompt
func (b *Bot) InjectCoordinatorPrompt(allBotConfigs []*Config) {
	info := ActAsTellStart
	for i, botConfig := range allBotConfigs {
		info += fmt.Sprintf("%d. %s\n    Usage: %s\n", i+1, botConfig.PrefabName, botConfig.Usage)
	}
	b.ActAsContext += info + ActAsTellTail
}
