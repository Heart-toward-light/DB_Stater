/*
@Author : WuWeiJian
@Date : 2020-12-15 19:47
*/

package cmd

import (
	"dbup/internal/prometheus"
	"dbup/internal/prometheus/config"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func prometheusCmd() *cobra.Command {
	// 定义二级命令: prometheus
	var cmd = &cobra.Command{
		Use:   "prometheus",
		Short: "prometheus相关操作",
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}
	cmd.AddCommand(
		//prometheusPrepareCmd(),
		prometheusInstallCmd(),
		grafanaInstallCmd(),
		registerExporterCmd(),
		deregisterExporterCmd(),
		installNodeExporterCmd(),
		installPostgresExporterCmd(),
		installRedisExporterCmd(),
		installMongodbExporterCmd(),
		installMariadbExporterCmd(),
	)
	return cmd
}

// dbup prometheus prepare
//func prometheusPrepareCmd() *cobra.Command {
//	var cfgFile string
//	var pre config.Prepare
//	cmd := &cobra.Command{
//		Use:   "prepare",
//		Short: "prometheus 安装之前, 生成安装前的部署配置信息",
//		RunE: func(cmd *cobra.Command, args []string) error {
//			if err := pre.MakeConfigFile(cfgFile); err != nil {
//				return err
//			}
//			return nil
//		},
//	}
//	cmd.Flags().StringVarP(&pre.Dir, "dir", "d", "", "数据目录")
//	cmd.Flags().StringVarP(&cfgFile, "config", "c", "", fmt.Sprintf("安装配置文件,默认为:$HOME/%s", config.DefaultPrometheusCfgFile))
//	return cmd
//}

// dbup prometheus install
func prometheusInstallCmd() *cobra.Command {
	var cfgFile string
	var pre config.Prepare
	cmd := &cobra.Command{
		Use:   "install",
		Short: "prometheus 单机版安装",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := prometheus.NewPrometheus()
			return p.Install(pre, cfgFile)
		},
	}
	cmd.Flags().StringVarP(&pre.Dir, "dir", "d", config.DefaultPrometheusDir, "数据目录")
	cmd.Flags().IntVarP(&pre.PrometheusPort, "prometheus-port", "P", 0, "prometheus 监听端口")
	//cmd.Flags().IntVarP(&pre.GrafanaPort, "grafana_port", "G", 0, "pgsql 数据库监听端口")
	//cmd.Flags().IntVarP(&pre.ConsulPort, "consul_port", "C", 0, "pgsql 数据库监听端口")
	cmd.Flags().IntVarP(&pre.NodeExporterPort, "node-port", "N", 0, "node_exporter 端口")
	cmd.Flags().StringVarP(&cfgFile, "config", "c", "", fmt.Sprintf("安装配置文件,默认为:$HOME/%s", config.DefaultPrometheusCfgFile))
	cmd.Flags().StringVar(&pre.GrafanaPassword, "grafana-password", "", "grafana 登录密码，默认随机生成")
	cmd.Flags().BoolVar(&pre.WithoutGrafana, "without-grafana", false, "不安装grafana，默认安装")
	return cmd
}

func grafanaInstallCmd() *cobra.Command {
	var cfgFile string
	var pre config.Prepare
	cmd := &cobra.Command{
		Use:   "install-grafana",
		Short: "单独安装grafana",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := prometheus.NewPrometheus()
			return p.InstallGrafana(pre, cfgFile)
		},
	}
	cmd.Flags().StringVarP(&pre.Dir, "dir", "d", config.DefaultPrometheusDir, "数据目录")
	cmd.Flags().StringVar(&pre.GrafanaPassword, "grafana-password", "", "grafana 登录密码，默认随机生成")
	return cmd
}

// dbup register exporter to prometheus
func registerExporterCmd() *cobra.Command {
	var cfg config.RegisterExporterConf
	cmd := &cobra.Command{
		Use:   "register-exporter",
		Short: "注册 exporter",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := prometheus.NewPrometheus()
			return p.RegisterExporter(cfg)
		},
	}

	cmd.Flags().StringVarP(&cfg.JobName, "job", "j", "", fmt.Sprintf("必填项, export名称: %s", strings.Join(config.JobNameList, ",")))
	//cmd.Flags().StringVarP(&cfg.Interval, "interval", "i", "60s", "prometheus 拉取间隔")
	cmd.Flags().StringVarP(&cfg.MetricUrl, "metric-url", "m", "", "必填项, prometheus 拉取地址,eg. http://127.0.0.1:9090 或者 http://127.0.0.1:9090/metrics")
	//cmd.Flags().StringVarP(&cfg.ConsulAddr, "consul_addr", "c", "http://127.0.0.1:8500", "consul 注册地址")
	cmd.Flags().StringVarP(&cfg.PrometheusConfDir, "prometheus-conf-dir", "d", config.DefaultPrometheusConfDir, "prometheus配置安装地址")
	cmd.Flags().StringToStringVarP(&cfg.Tags, "tags", "t", nil, "tag标签,eg. key1=value1,key2=value2")
	cmd.Flags().IntVarP(&cfg.PrometheusPort, "prometheus-port", "p", 9090, "prometheus端口")

	_ = cmd.MarkFlagRequired("job")
	_ = cmd.MarkFlagRequired("metric-url")

	return cmd
}

func deregisterExporterCmd() *cobra.Command {
	var cfg config.RegisterExporterConf
	cmd := &cobra.Command{
		Use:   "deregister-exporter",
		Short: "撤销 exporter 的注册",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := prometheus.NewPrometheus()
			return p.DeRegisterExporter(cfg)
		},
	}

	cmd.Flags().StringVarP(&cfg.JobName, "job", "j", "", fmt.Sprintf("必填项, export名称: %s", strings.Join(config.JobNameList, ",")))
	cmd.Flags().StringVarP(&cfg.MetricUrl, "metric-url", "m", "", "必填项, prometheus 拉取地址,eg. http://127.0.0.1:9090 或者 http://127.0.0.1:9090/metrics")
	cmd.Flags().StringVarP(&cfg.PrometheusConfDir, "prometheus-conf-dir", "d", config.DefaultPrometheusConfDir, "prometheus配置安装地址")
	cmd.Flags().IntVarP(&cfg.PrometheusPort, "prometheus-port", "p", 9090, "prometheus端口")

	_ = cmd.MarkFlagRequired("job")
	_ = cmd.MarkFlagRequired("metric-url")

	return cmd
}

// dbup register exporter to prometheus
func installNodeExporterCmd() *cobra.Command {
	var cfg config.NodeExporterConf
	cmd := &cobra.Command{
		Use:   "install-node-exporter",
		Short: "安装 node-exporter",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := prometheus.NewPrometheus()
			return p.InstallNodeExporter(cfg)
		},
	}

	cmd.Flags().StringVarP(&cfg.Dir, "dir", "d", config.DefaultExportersDir, "安装目录")
	cmd.Flags().IntVarP(&cfg.Port, "port", "p", 0, "node_exporter 端口")

	return cmd
}

func installPostgresExporterCmd() *cobra.Command {
	var cfg config.PostgresExporterConf
	cmd := &cobra.Command{
		Use:   "install-postgres-exporter",
		Short: "安装 postgres-exporter",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := prometheus.NewPrometheus()
			return p.InstallPostgresExporter(cfg)
		},
	}

	cmd.Flags().StringVarP(&cfg.Dir, "dir", "d", config.DefaultExportersDir, "安装目录")
	cmd.Flags().IntVarP(&cfg.Port, "port", "P", 0, "postgres_exporter 端口，默认 9187")
	cmd.Flags().StringVarP(&cfg.PgAddr, "pg-addr", "a", "", "pgsql 实例连接信息ip:port，eg. 127.0.0.1:5432")
	cmd.Flags().StringVarP(&cfg.Pass, "pass", "p", "", "pgsql 实例 postgres 账号的密码")

	_ = cmd.MarkFlagRequired("pg-addr")
	_ = cmd.MarkFlagRequired("pass")

	return cmd
}

func installRedisExporterCmd() *cobra.Command {
	var cfg config.RedisExporterConf
	cmd := &cobra.Command{
		Use:   "install-redis-exporter",
		Short: "安装 redis-exporter",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := prometheus.NewPrometheus()
			return p.InstallRedisExporter(cfg)
		},
	}

	cmd.Flags().StringVarP(&cfg.Dir, "dir", "d", config.DefaultExportersDir, "安装目录")
	cmd.Flags().IntVarP(&cfg.Port, "port", "P", 0, "redis_exporter 端口，默认 9121")
	cmd.Flags().StringVarP(&cfg.RedisAddr, "redis-addr", "a", "", "redis 实例连接信息ip:port，eg. 127.0.0.1:6379")
	cmd.Flags().StringVarP(&cfg.RedisPassword, "redis-password", "p", "", "redis实例的密码")

	_ = cmd.MarkFlagRequired("redis-addr")
	_ = cmd.MarkFlagRequired("redis-password")

	return cmd
}

func installMongodbExporterCmd() *cobra.Command {
	var cfg config.MongodbExporterConf
	cmd := &cobra.Command{
		Use:   "install-mongodb-exporter",
		Short: "安装 mongodb-exporter",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := prometheus.NewPrometheus()
			return p.InstallMongodbExporter(cfg)
		},
	}

	cmd.Flags().StringVarP(&cfg.Dir, "dir", "d", config.DefaultExportersDir, "安装目录")
	cmd.Flags().IntVarP(&cfg.Port, "port", "P", 0, "redis_exporter 端口，默认 9216")
	cmd.Flags().StringVarP(&cfg.MongodbAddr, "mongodb-addr", "a", "", "mongodb 实例连接信息ip:port，eg. 127.0.0.1:27017")
	cmd.Flags().StringVarP(&cfg.MongodbPassword, "mongodb-password", "p", "", "mongodb 实例的密码")
	cmd.Flags().StringVarP(&cfg.MongodbUser, "mongodb-user", "u", "", "mongodb 实例的用户")

	_ = cmd.MarkFlagRequired("mongodb-addr")
	_ = cmd.MarkFlagRequired("mongodb-password")
	_ = cmd.MarkFlagRequired("mongodb-user")

	return cmd
}

func installMariadbExporterCmd() *cobra.Command {
	var cfg config.MariadbExporterConf
	cmd := &cobra.Command{
		Use:   "install-mariadb-exporter",
		Short: "安装 mariadb-exporter",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := prometheus.NewPrometheus()
			return p.InstallMariadbExporter(cfg)
		},
	}

	cmd.Flags().StringVarP(&cfg.Dir, "dir", "d", config.DefaultExportersDir, "安装目录")
	cmd.Flags().IntVarP(&cfg.Port, "port", "P", 0, "mariadb_exporter 端口，默认 9104")
	cmd.Flags().StringVarP(&cfg.MariadbAddr, "mariadb-addr", "H", "", "mariadb 实例ip, 默认 127.0.0.1")
	cmd.Flags().StringVarP(&cfg.MariadbPassword, "mariadb-password", "p", "", "mariadb 实例的密码")
	cmd.Flags().StringVarP(&cfg.MariadbUser, "mariadb-user", "u", "", "mariadb 实例的用户")
	cmd.Flags().IntVarP(&cfg.MariadbPort, "mariadb-port", "o", 0, "mariadb 实例的端口")

	_ = cmd.MarkFlagRequired("mariadb-addr")
	_ = cmd.MarkFlagRequired("mariadb-password")
	_ = cmd.MarkFlagRequired("mariadb-user")
	_ = cmd.MarkFlagRequired("mariadb-port")

	return cmd
}
