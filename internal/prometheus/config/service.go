/*
@Author : WuWeiJian
@Date : 2020-12-09 15:56
*/

package config

var ConsulService = `
[Unit]
Description=consul-service
After=network-online.target

[Service]
User=root
Restart=on-failure
ExecStart=%s/consul/consul agent -data-dir=%s/consul/data -config-dir=%s/consul/config -server -bind=127.0.0.1 -client=0.0.0.0 -bootstrap-expect=1 -ui -datacenter=consul01
LimitNOFILE=10000
TimeoutStopSec=20

[Run]
WantedBy=multi-user.target`

var PrometheusService = `
[Unit]
Description=Prometheus
Wants=network-online.target
After=network-online.target

[Service]
Type=simple
User=root
Group=root
ExecStart=%s/prometheus/prometheus \
    --config.file=%s/prometheus/prometheus.yml \
    --storage.tsdb.path=%s/prometheus/data \
    --storage.tsdb.retention.time=30d \
    --web.console.templates=%s/prometheus/consoles \
    --web.console.libraries=%s/prometheus/console_libraries \
    --web.listen-address=0.0.0.0:%d \
    --web.enable-lifecycle
Restart=on-failure

[Install]
WantedBy=multi-user.target`

var GrafanaService = `
[Unit]
Description=Grafana instance
Documentation=http://docs.grafana.org
Wants=network-online.target
After=network-online.target
After=postgresql.service mariadb.service mysqld.service

[Service]
Environment=CONF_FILE=%s/grafana/conf/defaults.ini
WorkingDirectory=%s/grafana
User=root
Group=root
Type=notify
Restart=on-failure
RuntimeDirectory=root
RuntimeDirectoryMode=0750
ExecStart=%s/grafana/bin/grafana-server
                            --config=${CONF_FILE} 

LimitNOFILE=10000
TimeoutStopSec=20

[Install]
WantedBy=multi-user.target`

var NodeExporterService = `
[Unit]
Description=node exporter
After=network.target
Wants=network-online.target

[Service]
Type=simple
User=root
ExecStart=%s/node_exporter_dbup/node_exporter --web.listen-address=:%d
Restart=on-failure
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target`

// Config represent the data to generate systemd config
//type PrometheusService struct {
//	Body string
//}
//
//func NewPrometheusService() *PrometheusService {
//	var body = `[Unit]
//Description=Prometheus Monitoring System
//Documentation=Prometheus Monitoring System
//
//[Service]
//Type=simple
//User=root
//ExecStart=%s --config.file=%s --web.listen-address=%d --storage.tsdb.path=%s --web.enable-lifecycle
//ExecReload=/bin/kill -HUP $MAINPID
//KillMode=mixed
//Restart=on-failure
//RestartSec=5s
//
//[Run]
//WantedBy=multi-user.target`
//	return &PrometheusService{
//		Body: body,
//	}
//}
//
//func (s *PrometheusService) FormatBody(port int, serverName, confFileName, dataPath string) {
//	s.Body = fmt.Sprintf(s.Body,
//		serverName,
//		confFileName,
//		port,
//		dataPath)
//}
//
//func (s *PrometheusService) SaveTo(filename string) error {
//	return ioutil.WriteFile(filename, []byte(s.Body), 0755)
//}

var PostgresExporterService = `
[Unit]
Description=postgres_exporter
Documentation=postgres_exporter

[Service]
Type=simple
User=root
Environment="DATA_SOURCE_NAME=postgresql://postgres:%s@%s/?sslmode=disable"
ExecStart=%s/postgres_exporter%d/postgres_exporter --web.listen-address=:%d --auto-discover-databases
KillMode=mixed
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target`

var RedisExporterService = `
[Unit]
Description=redis_exporter
Documentation=redis_exporter

[Service]
Type=simple
User=root
ExecStart=%s/redis_exporter%d/redis_exporter -redis.addr=%s -redis.password=%s  -web.listen-address=:%d
KillMode=mixed
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target`

var MongodbExporterService = `
[Unit]
Description=mongodb_exporter
Documentation=mongodb_exporter

[Service]
Type=simple
User=root
Environment="MONGODB_URI=mongodb://%s:%s@%s/admin"
ExecStart=%s/mongodb_exporter%d/mongodb_exporter --web.listen-address=:%d
KillMode=mixed
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
`

var MariadbExporterService = `
[Unit]
Description=mariadb_exporter
Documentation=mariadb_exporter

[Service]
Type=simple
User=root
ExecStart=%s/mariadb_exporter%d/mysql_exporter --web.listen-address=:%d --config.my-cnf=%s
KillMode=mixed
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
`

var MariadbExporterConfig = `
[client]
user=%s
password=%s
port=%d
host=%s
`
