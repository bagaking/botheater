package call

import (
	"context"
	"regexp"
	"strings"

	"github.com/bagaking/goulp/wlog"
	"github.com/khicago/irr"
)

type Caller struct {
	Prefix string
	Regex  *regexp.Regexp
}

func (ct *Caller) ParseCall(ctx context.Context, content string) (name string, params []string, err error) {
	log := wlog.ByCtx(ctx, "parse_call")
	matches := ct.Regex.FindStringSubmatch(content)
	if len(matches) < 2 {
		return "", nil, irr.Error("invalid call format")
	}

	name = matches[1]
	paramsStr := matches[2]
	if paramsStr == "" {
		return name, []string{}, nil
	}
	params = strings.Split(paramsStr, ",")

	log.Infof("find caller %s%s ( %v )", ct.Prefix, name, strings.Join(params, ","))
	return name, params, nil
}

func (ct *Caller) HasCall(content string) bool {
	return ct.Regex.MatchString(content)
}
