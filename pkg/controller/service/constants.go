package service

// variables refering to the redis exporter port
const (
	exporterPort                 = 9121
	exporterPortName             = "http-metrics"
	exporterContainerName        = "redis-exporter"
	exporterDefaultRequestCPU    = "25m"
	exporterDefaultLimitCPU      = "50m"
	exporterDefaultRequestMemory = "50Mi"
	exporterDefaultLimitMemory   = "100Mi"
)
