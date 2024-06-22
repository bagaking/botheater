package tools

import (
	"math/rand"
	"time"

	"github.com/bagaking/botheater/call/tool"
)

type (
	RandomIdeaGenerator       struct{}
	RandomIdeaGeneratorParams struct{}
)

var _ tool.ITool = &RandomIdeaGenerator{}

func (r *RandomIdeaGenerator) Name() string {
	return "random_idea_generator"
}

func (r *RandomIdeaGenerator) Usage() string {
	return "调用一次，获得一个点子"
}

func (r *RandomIdeaGenerator) Examples() []string {
	return []string{"random_idea_generator()"}
}

func (r *RandomIdeaGenerator) ParamNames() []string {
	return []string{}
}

// Execute 生成随机想法
func (r *RandomIdeaGenerator) Execute(data map[string]string) (any, error) {
	ideas := []string{
		"去公园散步",
		"读一本新书",
		"学习一门新技能",
		"写一篇博客",
		"尝试新的食谱",
		"继续想想",
		"找一个相似的东西",
		"想想背后的道理",
	}

	rand.Seed(time.Now().UnixNano())
	randomIndex := rand.Intn(len(ideas))
	return ideas[randomIndex], nil
}
