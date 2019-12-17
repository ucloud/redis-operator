package redis

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	rediscli "github.com/go-redis/redis"

	"github.com/ucloud/redis-operator/pkg/util"
)

// Client defines the functions necessary to connect to redis and sentinel to get or set what we need
type Client interface {
	GetNumberSentinelsInMemory(ip string, auth *util.AuthConfig) (int32, error)
	GetNumberSentinelSlavesInMemory(ip string, auth *util.AuthConfig) (int32, error)
	ResetSentinel(ip string, auth *util.AuthConfig) error
	GetSlaveMasterIP(ip string, auth *util.AuthConfig) (string, error)
	IsMaster(ip string, auth *util.AuthConfig) (bool, error)
	MonitorRedis(ip string, monitor string, quorum string, auth *util.AuthConfig) error
	MakeMaster(ip string, auth *util.AuthConfig) error
	MakeSlaveOf(ip string, masterIP string, auth *util.AuthConfig) error
	GetSentinelMonitor(ip string, auth *util.AuthConfig) (string, error)
	SetCustomSentinelConfig(ip string, configs []string, auth *util.AuthConfig) error
	SetCustomRedisConfig(ip string, configs map[string]string, auth *util.AuthConfig) error
	GetAllRedisConfig(rClient *rediscli.Client) (map[string]string, error)
}

type client struct {
}

// New returns a redis client
func New() Client {
	return &client{}
}

const (
	sentinelsNumberREString = "sentinels=([0-9]+)"
	slaveNumberREString     = "slaves=([0-9]+)"
	sentinelStatusREString  = "status=([a-z]+)"
	redisMasterHostREString = "master_host:([0-9a-zA-Z:.]+)"
	redisRoleMaster         = "role:master"
	redisPort               = "6379"
	sentinelPort            = "26379"
	masterName              = "mymaster"

	defaultDownAfterMilliseconds = "5000"
	defaultFailovertimeout       = "3000"
	defaultParallelSyncs         = "2"
)

var (
	sentinelNumberRE  = regexp.MustCompile(sentinelsNumberREString)
	sentinelStatusRE  = regexp.MustCompile(sentinelStatusREString)
	slaveNumberRE     = regexp.MustCompile(slaveNumberREString)
	redisMasterHostRE = regexp.MustCompile(redisMasterHostREString)
)

// GetNumberSentinelsInMemory return the number of sentinels that the requested sentinel has
func (c *client) GetNumberSentinelsInMemory(ip string, auth *util.AuthConfig) (int32, error) {
	options := c.setOptions(ip, sentinelPort, auth)
	rClient := rediscli.NewClient(options)
	defer rClient.Close()
	info, err := rClient.Info("sentinel").Result()
	if err != nil {
		return 0, err
	}
	if err2 := isSentinelReady(info); err2 != nil {
		return 0, err2
	}
	match := sentinelNumberRE.FindStringSubmatch(info)
	if len(match) == 0 {
		return 0, errors.New("seninel regex not found")
	}
	nSentinels, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, err
	}
	return int32(nSentinels), nil
}

// GetNumberSentinelsInMemory return the number of sentinels that the requested sentinel has
func (c *client) GetNumberSentinelSlavesInMemory(ip string, auth *util.AuthConfig) (int32, error) {
	options := c.setOptions(ip, sentinelPort, auth)
	rClient := rediscli.NewClient(options)
	defer rClient.Close()
	info, err := rClient.Info("sentinel").Result()
	if err != nil {
		return 0, err
	}

	if err = isSentinelReady(info); err != nil {
		return 0, err
	}

	cmd := rediscli.NewSliceCmd("sentinel", "slaves", masterName)
	rClient.Process(cmd)
	slaveInfoBlobs, err := cmd.Result()
	if err != nil {
		return 0, err
	}
	nSlaves := len(slaveInfoBlobs)
	for _, slaveInfoBlob := range slaveInfoBlobs {
		slavePriority := slaveInfoFieldByName("slave-priority", slaveInfoBlob)
		if slavePriority == "0" {
			nSlaves -= 1
		}
	}

	return int32(nSlaves), nil
}

func slaveInfoFieldByName(name string, slaveInfoBlob interface{}) string {
	slaveInfo := slaveInfoBlob.([]interface{})
	infoLens := len(slaveInfo)
	i := 0
	for i+1 < infoLens {
		stringValue := slaveInfo[i].(string)
		if stringValue == name {
			return slaveInfo[i+1].(string)
		}
		i += 2
	}
	return ""
}

func isSentinelReady(info string) error {
	matchStatus := sentinelStatusRE.FindStringSubmatch(info)
	if len(matchStatus) == 0 || matchStatus[1] != "ok" {
		return errors.New("sentinel not ready")
	}
	return nil
}

// ResetSentinel sends a sentinel reset * for the given sentinel
func (c *client) ResetSentinel(ip string, auth *util.AuthConfig) error {
	options := c.setOptions(ip, sentinelPort, auth)
	rClient := rediscli.NewClient(options)
	defer rClient.Close()
	cmd := rediscli.NewIntCmd("SENTINEL", "reset", "*")
	rClient.Process(cmd)
	_, err := cmd.Result()
	if err != nil {
		return err
	}
	return nil
}

// GetSlaveMasterIP returns the master of the given redis, or nil if it's master
func (c *client) GetSlaveMasterIP(ip string, auth *util.AuthConfig) (string, error) {
	options := c.setOptions(ip, redisPort, auth)
	rClient := rediscli.NewClient(options)
	defer rClient.Close()
	info, err := rClient.Info("replication").Result()
	if err != nil {
		return "", err
	}
	match := redisMasterHostRE.FindStringSubmatch(info)
	if len(match) == 0 {
		return "", nil
	}
	return match[1], nil
}

func (c *client) IsMaster(ip string, auth *util.AuthConfig) (bool, error) {
	options := c.setOptions(ip, redisPort, auth)
	rClient := rediscli.NewClient(options)
	defer rClient.Close()
	info, err := rClient.Info("replication").Result()
	if err != nil {
		return false, err
	}
	return strings.Contains(info, redisRoleMaster), nil
}

func (c *client) MonitorRedis(ip string, monitor string, quorum string, auth *util.AuthConfig) error {
	options := c.setOptions(ip, sentinelPort, auth)
	rClient := rediscli.NewClient(options)
	defer rClient.Close()
	cmd := rediscli.NewBoolCmd("SENTINEL", "REMOVE", masterName)
	rClient.Process(cmd)
	// We'll continue even if it fails, the priority is to have the redises monitored
	cmd = rediscli.NewBoolCmd("SENTINEL", "MONITOR", masterName, monitor, redisPort, quorum)
	rClient.Process(cmd)
	_, err := cmd.Result()
	if err != nil {
		return err
	}
	if auth.Password != "" {
		sCmd := rediscli.NewStatusCmd("SENTINEL", "SET", masterName, "auth-pass", auth.Password)
		rClient.Process(sCmd)
		if err = sCmd.Err(); err != nil {
			return err
		}
	}

	sCmd := rediscli.NewStatusCmd("SENTINEL", "SET", masterName, "down-after-milliseconds", defaultDownAfterMilliseconds)
	rClient.Process(sCmd)
	if err = sCmd.Err(); err != nil {
		return err
	}
	sCmd = rediscli.NewStatusCmd("SENTINEL", "SET", masterName, "failover-timeout", defaultFailovertimeout)
	rClient.Process(sCmd)
	if err = sCmd.Err(); err != nil {
		return err
	}
	sCmd = rediscli.NewStatusCmd("SENTINEL", "SET", masterName, "parallel-syncs", defaultParallelSyncs)
	rClient.Process(sCmd)
	if err = sCmd.Err(); err != nil {
		return err
	}

	return nil
}

func (c *client) MakeMaster(ip string, auth *util.AuthConfig) error {
	options := c.setOptions(ip, redisPort, auth)
	rClient := rediscli.NewClient(options)
	defer rClient.Close()
	if res := rClient.SlaveOf("NO", "ONE"); res.Err() != nil {
		return res.Err()
	}
	return nil
}

func (c *client) MakeSlaveOf(ip string, masterIP string, auth *util.AuthConfig) error {
	options := c.setOptions(ip, redisPort, auth)
	rClient := rediscli.NewClient(options)
	defer rClient.Close()
	if res := rClient.SlaveOf(masterIP, redisPort); res.Err() != nil {
		return res.Err()
	}
	return nil
}

func (c *client) GetSentinelMonitor(ip string, auth *util.AuthConfig) (string, error) {
	options := c.setOptions(ip, sentinelPort, auth)
	rClient := rediscli.NewClient(options)
	defer rClient.Close()
	cmd := rediscli.NewSliceCmd("SENTINEL", "master", masterName)
	rClient.Process(cmd)
	res, err := cmd.Result()
	if err != nil {
		return "", err
	}
	masterIP := res[3].(string)
	return masterIP, nil
}

func (c *client) SetCustomSentinelConfig(ip string, configs []string, auth *util.AuthConfig) error {
	options := c.setOptions(ip, sentinelPort, auth)
	rClient := rediscli.NewClient(options)
	defer rClient.Close()

	for _, config := range configs {
		param, value, err := c.getConfigParameters(config)
		if err != nil {
			return err
		}
		if err := c.applySentinelConfig(param, value, rClient); err != nil {
			return err
		}
	}
	return nil
}

func (c *client) SetCustomRedisConfig(ip string, configs map[string]string, auth *util.AuthConfig) error {
	options := c.setOptions(ip, redisPort, auth)
	rClient := rediscli.NewClient(options)
	defer rClient.Close()

	for param, value := range configs {
		//param, value, err := c.getConfigParameters(config)
		//if err != nil {
		//	return err
		//}
		if err := c.applyRedisConfig(param, value, rClient); err != nil {
			return err
		}
	}
	return nil
}

func (c *client) GetAllRedisConfig(rClient *rediscli.Client) (map[string]string, error) {
	result := rClient.ConfigGet("*")
	if result.Err() != nil {
		return nil, result.Err()
	}

	valMap := make(map[string]string)
	val := result.Val()
	for i := 0; i < len(val); i += 2 {
		valMap[val[i].(string)] = val[i+1].(string)
	}

	return valMap, nil
}

func (c *client) applyRedisConfig(parameter string, value string, rClient *rediscli.Client) error {
	result := rClient.ConfigSet(parameter, value)
	return result.Err()
}

func (c *client) applySentinelConfig(parameter string, value string, rClient *rediscli.Client) error {
	cmd := rediscli.NewStatusCmd("SENTINEL", "set", masterName, parameter, value)
	rClient.Process(cmd)
	return cmd.Err()
}

func (c *client) getConfigParameters(config string) (parameter string, value string, err error) {
	s := strings.Split(config, " ")
	if len(s) < 2 {
		return "", "", fmt.Errorf("configuration '%s' malformed", config)
	}
	return s[0], strings.Join(s[1:], " "), nil
}

func (c *client) setOptions(ip, port string, auth *util.AuthConfig) *rediscli.Options {
	passwd := auth.Password
	if port == sentinelPort {
		passwd = ""
	}
	return &rediscli.Options{
		Addr:     net.JoinHostPort(ip, port),
		Password: passwd,
		DB:       0,
	}
}
