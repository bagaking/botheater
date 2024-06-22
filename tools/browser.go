package tools

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/khicago/irr"

	"github.com/bagaking/botheater/call"
	"github.com/bagaking/botheater/call/tool"
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
	return "访问指定的 URL 并返回页面内容中的文本和链接"
}

func (b *Browser) Examples() []string {
	return []string{"browser(\"https://www.google.com/search?q=golang\")", "browser(\"https://en.wikipedia.org/wiki/vector_database\")"}
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

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var result strings.Builder
	doc.Find("body").Each(func(i int, s *goquery.Selection) {
		s.Find("a").Each(func(j int, a *goquery.Selection) {
			text := strings.TrimSpace(a.Text())
			if text != "" {
				return
			}
			href, exists := a.Attr("href")
			if exists {
				decodedHref, _ := url.QueryUnescape(href)
				decodedHref, _ = strconv.Unquote(`"` + decodedHref + `"`)
				result.WriteString(fmt.Sprintf("[%s](%s)\n", text, decodedHref))
			} else {
				result.WriteString(text)
				result.WriteRune('\n')
			}
		})
		s.Find("p, h1, h2, h3, h4, h5, h6, li, span, div").Each(func(j int, p *goquery.Selection) {
			text := strings.TrimSpace(p.Text())
			if text != "" {
				return
			}
			result.WriteString(text)
			result.WriteRune('\n')
		})
	})

	cleanedResult := strings.TrimSpace(result.String())
	cleanedResult = removeExtraNewlines(cleanedResult)
	return result.String(), nil
}

func removeExtraNewlines(input string) string {
	var result strings.Builder
	previousWasSpace := false

	for _, char := range input {
		if char == '\n' || char == ' ' || char == '\t' || char == '\r' {
			if !previousWasSpace {
				result.WriteRune(char)
				previousWasSpace = true
			}
		} else {
			result.WriteRune(char)
			previousWasSpace = false
		}
	}

	return result.String()
}
