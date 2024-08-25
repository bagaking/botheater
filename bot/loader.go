package bot

import (
	"context"
	"github.com/bagaking/botheater/driver"
	"github.com/bagaking/botheater/driver/coze"
	"github.com/bagaking/botheater/driver/ollama"
	"reflect"

	"github.com/bagaking/botheater/call/tool"
	"github.com/bagaking/goulp/wlog"
	"github.com/khicago/got/util/typer"
	"github.com/khicago/irr"
)

var (
	ErrPrefabNotFound = irr.Error("prefab not found")
	ErrBotNotFound    = irr.Error("bot not found")
)

// Loader is a struct that helps to load bots with chainable methods and error handling.
type Loader struct {
	tm   *tool.Manager
	bots []*Bot
	err  error
}

// NewBotLoader creates a new Loader instance.
func NewBotLoader(tm *tool.Manager) *Loader {
	return &Loader{
		tm: tm,
	}
}

func (bl *Loader) Error() error {
	return bl.err
}

// LoadBots loads bots
func (bl *Loader) LoadBots(ctx context.Context, configs map[string]*Config) *Loader {
	if bl.err != nil {
		return bl
	}
	logger := wlog.ByCtx(ctx, "load_bot")

	for prefabName := range configs {
		botConf, exists := configs[prefabName]
		if !exists {
			logger.Errorf("prefab %s not found", prefabName)
			bl.err = ErrPrefabNotFound
			return bl
		}
		bl.LoadBot(ctx, botConf)
	}
	return bl
}

// LoadBot loads a bot and adds it to the Loader.
func (bl *Loader) LoadBot(ctx context.Context, conf *Config) *Loader {
	if bl.err != nil {
		return bl
	}

	var d driver.Driver
	switch conf.DriverConf.Driver {
	case "ollama":
		d = ollama.New(ollama.NewClient(ctx), conf.DriverConf.Endpoint)
	case "coze":
		fallthrough
	default:
		d = coze.New(coze.NewClient(ctx), conf.DriverConf.Endpoint)
	}

	b := New(*conf, d, bl.tm)
	bl.bots = append(bl.bots, b)
	return bl
}

// GetBots returns the loaded bots and any error encountered.
func (bl *Loader) GetBots() ([]*Bot, error) {
	return bl.bots, bl.err
}

// GetBot returns the loaded bots and any error encountered.
func (bl *Loader) GetBot(name string) (*Bot, error) {
	i := typer.SliceFirstMatch(bl.bots, func(b *Bot) bool {
		return b.PrefabName == name
	})
	if i < 0 {
		return nil, ErrBotNotFound
	}
	return bl.bots[i], bl.err
}

// StaplingBots loads bots based on struct tags.
func (bl *Loader) StaplingBots(ctx context.Context, botsStructure any) error {
	if bl.err != nil {
		return irr.Wrap(bl.err, "loader error")
	}
	val := reflect.ValueOf(botsStructure).Elem()
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		tag := typ.Field(i).Tag.Get("bot")
		if tag == "" {
			continue
		}
		bot, err := bl.GetBot(tag)
		if err != nil {
			return irr.Wrap(err, "failed to get bot %s", tag)
		}
		val.Field(i).Set(reflect.ValueOf(bot))
	}
	return nil
}
