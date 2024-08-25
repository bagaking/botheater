// driver/ollama/init.go
package ollama

import (
	"context"
	"log"

	"github.com/ollama/ollama/api"
)

func NewClient(ctx context.Context) *api.Client {
	client, err := api.ClientFromEnvironment()
	if err != nil {
		log.Fatal(err)
	}
	return client
}
