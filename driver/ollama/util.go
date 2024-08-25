// driver/ollama/util.go
package ollama

import (
	"fmt"
	"strings"

	"github.com/bagaking/botheater/utils"
	"github.com/ollama/ollama/api"
)

func Req2Str(req *api.ChatRequest) string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("REQUEST (%d)\n", len(req.Messages)))
	for i, msg := range req.Messages {
		sb.WriteString(Msg2Str(i, msg))
	}
	return sb.String()
}

func Msg2Str(ind int, msg api.Message) string {
	content := strings.TrimSpace(msg.Content)
	if content == "" {
		content = "!!got-empty-content!! all msg is:\n" + msg.Content
	}

	return utils.SPrintWithFrameCard(
		fmt.Sprintf(" %d. role[%s] (len:%d)", ind, msg.Role, len(content)),
		content, utils.PrintWidthL2, utils.StyMsgCard,
	)
}
