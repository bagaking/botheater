package coze

import (
	"fmt"
	"strings"

	"github.com/bagaking/goulp/jsonex"

	"github.com/volcengine/volc-sdk-golang/service/maas/models/api/v2"
)

func Req2Str(req *api.ChatReq) string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("REQUEST (%d)\n", len(req.Messages)))
	for i, msg := range req.Messages {
		sb.WriteString(Msg2Str(i, msg))
	}
	return sb.String()
}

func Msg2Str(ind int, msg *api.Message) string {
	content, ok := msg.Content.(string)
	if !ok {
		content = jsonex.MustMarshalToString(msg.Content)
	}
	content = strings.TrimSpace(content)
	if content == "" {
		content = "!!got-empty-content!! all msg is:\n" + jsonex.MustMarshalToString(msg)
	}

	return fmt.Sprintf("--- %d. role[%s] name[%s] --- \n%v\n--- %d. fin ---\n\n", ind, msg.Role, msg.Name, content, ind)
}

func Resp2Str(resp *api.ChatResp) string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("RESPONSE (choices:%d) (usage:%v) (err:%v)\n", len(resp.Choices), resp.Usage, resp.Error))
	for i, c := range resp.Choices {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, c.Message.Content))
	}
	return sb.String()
}

func RespMsg2Str(resp *api.ChatResp) string {
	sb := strings.Builder{}
	for _, c := range resp.Choices {
		sb.WriteString(fmt.Sprintf("%s\n", c.Message.Content))
	}
	return strings.TrimSpace(sb.String())
}
