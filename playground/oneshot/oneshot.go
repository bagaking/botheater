package oneshot

import (
	"context"
	"crypto/md5"
	"encoding/base64"

	"github.com/bagaking/botheater/bot"
	"github.com/bagaking/botheater/history"
	"github.com/bagaking/goulp/wlog"
)

func Play(ctx context.Context, loader *bot.Loader, prompt, question string) {

	_, _ = SimpleQuestion(ctx, loader, prompt, question)

}

func SimpleQuestion(ctx context.Context, loader *bot.Loader, prompt string, question string) (string, error) {
	uid := md5.Sum([]byte(prompt))
	prefabName := "__sq__" + base64.StdEncoding.EncodeToString(uid[0:])
	existBot, err := loader.GetBot(prefabName)
	if err != nil {
		wlog.ByCtx(ctx).Debugf("Failed to get exist bot: %v", err)

		basic, err := loader.GetBot("botheater_basic")
		if err != nil {
			wlog.ByCtx(ctx).Fatalf("Failed to get bot: %v", err)
		}

		conf := *basic.Config
		conf.Prompt = &bot.Prompt{
			Content:   prompt,
			Functions: make([]string, 0),
		}
		conf.PrefabName = prefabName
		if err = loader.LoadBot(ctx, &conf).Error(); err != nil {
			wlog.ByCtx(ctx).Fatalf("Failed to load new bot: %v", err)
		}
		existBot, err = loader.GetBot(prefabName)
		if err != nil {
			wlog.ByCtx(ctx).Fatalf("Failed to get new bot: %v", err)
		}
	}
	if existBot == nil {
		wlog.ByCtx(ctx).Fatalf("Failed to get new bot")
	}

	return existBot.Question(ctx, history.NewHistory(), question)
}
