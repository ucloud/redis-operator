package redis_test

import (
	rediscli "github.com/go-redis/redis"
	"github.com/ucloud/redis-operator/pkg/client/redis"
	"strings"
	"testing"
)

func newClient() *rediscli.Client {
	return rediscli.NewClient(&rediscli.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}

func TestGetAllRedisConfig(t *testing.T) {
	cli := newClient()
	//var client redis.Client
	client := redis.New()
	result, err := client.GetAllRedisConfig(cli)
	if err != nil {
		t.Fatal(err)
	}

	if v, ok := result["client-output-buffer-limit"]; !ok {
		t.Fatal("must have config 'client-output-buffer-limit'")
	} else {
		if !strings.Contains(v, "normal") {
			t.Fatal("'client-output-buffer-limit' is empty")
		}
	}

}
