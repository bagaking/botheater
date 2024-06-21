package main

import (
	"context"
	"os"

	"github.com/bagaking/botheater/bot"
	"github.com/bagaking/botheater/tool"
	"github.com/bagaking/goulp/wlog"
	client "github.com/volcengine/volc-sdk-golang/service/maas/v2"

	"gopkg.in/yaml.v2"

	"github.com/khicago/irr"
)

type (
	Conf struct {
		BotPrefabs []*bot.BotConfig `yaml:"bot_prefabs"`
		bots       map[string]*bot.BotConfig
	}
)

const ConfigPath = "./conf.yml"

var (
	VOLC_ACCESSKEY = os.Getenv("VOLC_ACCESSKEY")
	VOLC_SECRETKEY = os.Getenv("VOLC_SECRETKEY")

	ErrPrefabNotFound = irr.Error("prefab not found")
)

func LoadConf(ctx context.Context) Conf {
	log := wlog.ByCtx(ctx, "load_conf")
	// 读取 YAML 文件
	data, err := os.ReadFile(ConfigPath)
	if err != nil {
		log.WithError(err).Warnf("Failed to read config file")
		return Conf{}
	}

	c := Conf{
		bots: make(map[string]*bot.BotConfig),
	}
	// 解析 YAML 数据
	err = yaml.Unmarshal(data, &c)
	if err != nil {
		log.WithError(err).Warnf("Failed to unmarshal config")
		return Conf{}
	}

	// 初始化 Bots map
	for _, botConfig := range c.BotPrefabs {
		c.bots[botConfig.PrefabName] = botConfig
	}

	return c
}

func (c *Conf) NewBot(cli *client.MaaS, prefabName string, tm *tool.ToolManager) (*bot.Bot, error) {
	botConf, exists := c.bots[prefabName]
	if !exists {
		return nil, ErrPrefabNotFound
	}
	return bot.New(*botConf, cli, tm), nil
}

func initClient(ctx context.Context) *client.MaaS {
	r := client.NewInstance("maas-api.ml-platform-cn-beijing.volces.com", "cn-beijing")

	wlog.ByCtx(ctx, "initClient").Infof("init client with IAM Keys: VOLC_SECRETKEY= %s, VOLC_SECRETKEY= %s", VOLC_ACCESSKEY, VOLC_SECRETKEY)

	// fetch ak&sk from environmental variables
	r.SetAccessKey(VOLC_ACCESSKEY)
	r.SetSecretKey(VOLC_SECRETKEY)

	return r
}
