package tools

import (
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/bagaking/botheater/call"
	"github.com/bagaking/botheater/call/tool"
	"github.com/khicago/irr"
)

type (
	Browser       struct{}
	BrowserParams struct {
		URL string `json:"url"`
	}
)

var _ tool.ITool = &Browser{}

func (b *Browser) Name() string {
	return "browser"
}

func (b *Browser) Usage() string {
	return "访问指定的 URL 并返回页面内容"
}

func (b *Browser) Examples() []string {
	return []string{"browser(\"https://www.google.com/search?q=golang\")"}
}

func (b *Browser) ParamNames() []string {
	return []string{"url"}
}

func (b *Browser) Execute(params map[string]string) (any, error) {
	urlStr, ok := params["url"]
	if !ok {
		return nil, irr.Wrap(call.ErrExecFailedInvalidParams, "parameter 'url' is required in %v", params)
	}
	if urlStr == "" {
		return nil, irr.Wrap(call.ErrExecFailedInvalidParams, "url cannot be empty")
	}

	_, err := url.ParseRequestURI(urlStr)
	if err != nil {
		return nil, irr.Wrap(call.ErrExecFailedInvalidParams, "invalid url format")
	}

	resp, err := http.Get(urlStr)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to fetch the URL")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return string(body), nil
}
