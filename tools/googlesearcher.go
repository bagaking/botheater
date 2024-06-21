package tools

import (
	"net/url"

	"github.com/bagaking/botheater/call"
	"github.com/bagaking/botheater/call/tool"
	"github.com/khicago/irr"
)

type (
	GoogleSearcher       struct{}
	GoogleSearcherParams struct {
		Query string `json:"query"`
	}
)

var _ tool.ITool = &GoogleSearcher{}

func (g *GoogleSearcher) Name() string {
	return "google_searcher"
}

func (g *GoogleSearcher) Usage() string {
	return "使用 Google 搜索引擎进行搜索，并返回搜索结果"
}

func (g *GoogleSearcher) Examples() []string {
	return []string{"google_searcher(\"golang tutorial\")"}
}

func (g *GoogleSearcher) ParamNames() []string {
	return []string{"query"}
}

func (g *GoogleSearcher) Execute(params map[string]string) (any, error) {
	query, ok := params["query"]
	if !ok {
		return nil, irr.Wrap(call.ErrExecFailedInvalidParams, "parameter 'query' is required in %v", params)
	}
	if query == "" {
		return nil, irr.Wrap(call.ErrExecFailedInvalidParams, "query cannot be empty")
	}

	searchURL := "https://www.google.com/search?q=" + url.QueryEscape(query)
	browser := &Browser{}
	result, err := browser.Execute(map[string]string{"url": searchURL})
	if err != nil {
		return nil, err
	}

	return result, nil
}
