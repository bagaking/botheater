package main

import (
	"context"

	"github.com/bagaking/goulp/wlog"
	"github.com/bagaking/goulp/yaml"
	"github.com/khicago/irr"

	"github.com/bagaking/botheater/bot"
	"github.com/bagaking/botheater/call/tool"
	"github.com/bagaking/botheater/driver/coze"
)

type (
	Conf struct {
		BotPrefabs []*bot.Config `yaml:"bot_prefabs"`
		bots       map[string]*bot.Config
	}
)

const ConfigPath = "./conf.yml"

var ErrPrefabNotFound = irr.Error("prefab not found")

func LoadConf(ctx context.Context) Conf {
	log := wlog.ByCtx(ctx, "load_conf")
	// 读取 YAML 文件
	c := Conf{
		bots: make(map[string]*bot.Config),
	}
	err := yaml.LoadYAMLFile(ConfigPath, &c)
	if err != nil {
		log.WithError(err).Warnf("Failed to read config file")
		return Conf{}
	}

	// 初始化 Bots map
	for _, botConfig := range c.BotPrefabs {
		c.bots[botConfig.PrefabName] = botConfig
	}

	return c
}

func (c *Conf) NewBot(ctx context.Context, prefabName string, tm *tool.Manager) (*bot.Bot, error) {
	botConf, exists := c.bots[prefabName]
	if !exists {
		return nil, ErrPrefabNotFound
	}
	driver := coze.New(coze.NewClient(ctx), botConf.Endpoint)
	return bot.New(*botConf, driver, tm), nil
}
