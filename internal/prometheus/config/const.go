/*
@Author : WuWeiJian
@Date : 2020-12-03 14:29
*/

package config

// 一些默认配置
const (
	DefaultPrometheusCfgFile = "prometheus_install.ini"

	DefaultPrometheusPort       = 9090
	DefaultPrometheusPortArm    = 9092
	DefaultGrafanaPort          = 3000
	DefaultConsulPort           = 8500
	DefaultNodeExporterPort     = 9100
	DefaultPostgresExporterPort = 9187
	DefaultRedisExporterPort    = 9121
	DefaultMongodbExporterPort  = 9216
	DefaultMariadbExporterPort  = 9104
	DefaultPrometheusDir        = "/opt/prometheus"
	DefaultExportersDir         = "/opt/exporters"
	DefaultPrometheusVersion    = "2.23.0"
	DefaultPrometheusConfDir    = "/opt/prometheus/prometheus/conf"
)

// 版本
const (
	PrometheusV230MD5 = "854b6aa16d4efe3fae8626d7d5d452de"
)

// install 默认配置
const (
	Kinds            = "prometheus"
	PackageFile      = "%s_%s_%s.tar.gz"
	ServerFileName   = "prometheus"
	ConfDir          = "conf"
	ConfFileName     = "prometheus.yml"
	DataDir          = "data"
	ServiceFileName  = "prometheus%d.service"
	Consul           = "consul"
	Grafana          = "grafana"
	Prometheus       = "prometheus"
	NodeExporter     = "node_exporter"
	PostgresExporter = "postgres_exporter"
	RedisExporter    = "redis_exporter"
	MongodbExporter  = "mongodb_exporter"
	MariadbExporter  = "mariadb_exporter"
)
