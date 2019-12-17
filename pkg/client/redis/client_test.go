package redis

import (
	"strings"
	"testing"

	rediscli "github.com/go-redis/redis"
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
	client := New()
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

func Test_slaveInfoFieldByName(t *testing.T) {
	slaveInfoBlobA := []interface{}{"name", "[xxxxA]:6379", "ip", "xxxxA", "port", "6379", "runid", "6f792839ab551e8dbec58e0eb3b3838d14f19a37", "flags", "slave", "link-pending-commands", "1", "link-refcount", "1", "last-ping-sent", "0", "last-ok-ping-reply", "1055", "last-ping-reply", "1055", "down-after-milliseconds", "5000", "info-refresh", "2074", "role-reported", "slave", "role-reported-time", "2983115", "master-link-down-time", "0", "master-link-status", "ok", "master-host", "xxxxA", "master-port", "6379", "slave-priority", "1", "slave-repl-offset", "124614695"}
	slaveInfoBlobB := []interface{}{"name", "[xxxxB]:6371", "ip", "xxxxB", "port", "6371", "runid", "fake_slave_8bb90711-8f37-44e8-b3b2-589af", "flags", "slave", "link-pending-commands", "1", "link-refcount", "1", "last-ping-sent", "0", "last-ok-ping-reply", "1055", "last-ping-reply", "1055", "down-after-milliseconds", "5000", "info-refresh", "2075", "role-reported", "slave", "role-reported-time", "2983114", "master-link-down-time", "0", "master-link-status", "ok", "master-host", "xxxxB", "master-port", "6379", "slave-priority", "0", "slave-repl-offset", "124614695"}
	slaveInfoBlobC := []interface{}{"name", "[xxxxB]:6371", "ip", "slave-priority", "slave-priority", "100"}
	slaveInfoBlobD := []interface{}{"name", "[xxxxB]:6371", "ip", "xxxxB", "slave-priority"}
	type args struct {
		name          string
		slaveInfoBlob interface{}
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "slaveA",
			args: args{
				name:          "slave-priority",
				slaveInfoBlob: slaveInfoBlobA,
			},
			want: "1",
		},
		{
			name: "slaveB",
			args: args{
				name:          "slave-priority",
				slaveInfoBlob: slaveInfoBlobB,
			},
			want: "0",
		},
		{
			name: "slaveC",
			args: args{
				name:          "slave-priority",
				slaveInfoBlob: slaveInfoBlobC,
			},
			want: "100",
		},
		{
			name: "slaveD",
			args: args{
				name:          "slave-priority",
				slaveInfoBlob: slaveInfoBlobD,
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := slaveInfoFieldByName(tt.args.name, tt.args.slaveInfoBlob); got != tt.want {
				t.Errorf("slaveInfoFieldByName() = %v, want %v", got, tt.want)
			}
		})
	}
}
