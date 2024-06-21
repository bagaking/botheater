package bot

import (
	"fmt"
	"strings"

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
	return fmt.Sprintf("--- %d. [%s-%s] --- \n%v\n--- %d. fin ---\n\n", ind, msg.Role, msg.Name, msg.Content, ind)
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
	return sb.String()
}
