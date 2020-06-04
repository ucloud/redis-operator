package util

import (
	"strconv"
	"strings"
)

const (
	// AnnotationScope annotation name for defining instance scope. Used for specifying cluster wide clusters.
	// A namespace-scoped operator watches and manages resources in a single namespace, whereas a cluster-scoped operator watches and manages resources cluster-wide.
	AnnotationScope = "redis.kun/scope"
	//AnnotationClusterScoped annotation value for cluster wide clusters.
	AnnotationClusterScoped = "cluster-scoped"
)

var isClusterScoped = true

func IsClusterScoped() bool {
	return isClusterScoped
}

func SetClusterScoped(namespace string) {
	if namespace != "" {
		isClusterScoped = false
	}
}

func ParseRedisMemConf(p string) (string, error) {
	var mul int64 = 1
	u := strings.ToLower(p)
	digits := u

	if strings.HasSuffix(u, "k") {
		digits = u[:len(u)-len("k")]
		mul = 1000
	} else if strings.HasSuffix(u, "kb") {
		digits = u[:len(u)-len("kb")]
		mul = 1024
	} else if strings.HasSuffix(u, "m") {
		digits = u[:len(u)-len("m")]
		mul = 1000 * 1000
	} else if strings.HasSuffix(u, "mb") {
		digits = u[:len(u)-len("mb")]
		mul = 1024 * 1024
	} else if strings.HasSuffix(u, "g") {
		digits = u[:len(u)-len("g")]
		mul = 1000 * 1000 * 1000
	} else if strings.HasSuffix(u, "gb") {
		digits = u[:len(u)-len("gb")]
		mul = 1024 * 1024 * 1024
	} else if strings.HasSuffix(u, "b") {
		digits = u[:len(u)-len("b")]
		mul = 1
	}

	val, err := strconv.ParseInt(digits, 10, 64)
	if err != nil {
		return "", err
	}

	return strconv.FormatInt(val*mul, 10), nil
}
