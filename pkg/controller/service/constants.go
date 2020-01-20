package service

// variables refering to the redis exporter port
const (
	exporterPort                 = 9121
	exporterPortName             = "http-metrics"
	exporterContainerName        = "redis-exporter"
	exporterDefaultRequestCPU    = "50m"
	exporterDefaultLimitCPU      = "100m"
	exporterDefaultRequestMemory = "50Mi"
	exporterDefaultLimitMemory   = "200Mi"

	redisPasswordEnv = "REDIS_PASSWORD"
)
