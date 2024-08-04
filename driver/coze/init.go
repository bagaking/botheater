// Usage:
//
// 1. go get -u github.com/volcengine/volc-sdk-golang
// 2. VOLC_ACCESSKEY=XXXXX VOLC_SECRETKEY=YYYYY go run main.go
package coze

import (
	"context"

	"github.com/bagaking/botheater/utils"
	"github.com/bagaking/goulp/wlog"

	client "github.com/volcengine/volc-sdk-golang/service/maas/v2"
)

const (
	EnvKeyVOLCAccessKey  utils.EnvKey = "VOLC_ACCESSKEY"
	EnvKeyVOLCSecretKey  utils.EnvKey = "VOLC_SECRETKEY"
	EnvKeyDoubaoEndpoint utils.EnvKey = "DOUBAO_ENDPOINT"
)

var (
	VOLC_ACCESSKEY = EnvKeyVOLCAccessKey.Read()
	VOLC_SECRETKEY = EnvKeyVOLCSecretKey.Read()
)

func NewClient(ctx context.Context) *client.MaaS {
	r := client.NewInstance("maas-api.ml-platform-cn-beijing.volces.com", "cn-beijing")
	wlog.ByCtx(ctx, "coze.init").Debugf("init client with IAM Keys: VOLC_SECRETKEY= %s, VOLC_SECRETKEY= %s", VOLC_ACCESSKEY, VOLC_SECRETKEY)

	// fetch ak&sk from environmental variables
	r.SetAccessKey(VOLC_ACCESSKEY)
	r.SetSecretKey(VOLC_SECRETKEY)

	return r
}
