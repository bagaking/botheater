package main

import (
	"context"

	"github.com/bagaking/goulp/wlog"
	"github.com/bagaking/goulp/yaml"

	"github.com/bagaking/botheater/bot"
)

type (
	Conf struct {
		BotPrefabs []*bot.Config `yaml:"bot_prefabs"`
		bots       map[string]*bot.Config
	}
)

const ConfigPath = "./conf.yml"

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
