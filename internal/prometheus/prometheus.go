/*
@Author : WuWeiJian
@Date : 2020-12-15 19:52
*/

package prometheus

import (
	"dbup/internal/prometheus/config"
	"dbup/internal/prometheus/services"
)

// Prometheus 结构体
type Prometheus struct {
}

// 所有的Prometheus逻辑都在这里开始
func NewPrometheus() *Prometheus {
	return &Prometheus{}
}

func (p *Prometheus) Install(pre config.Prepare, cfgFile string) error {
	inst := services.NewInstall()
	return inst.Run(pre, cfgFile)
}

func (p *Prometheus) InstallGrafana(pre config.Prepare, cfgFile string) error {
	inst := services.NewInstall()
	pre.OnlyGrafana = true

	return inst.Run(pre, cfgFile)
}

func (p *Prometheus) InstallNodeExporter(pre config.NodeExporterConf) error {
	inst := services.NewInstallNodeExporter()
	return inst.Run(pre, "")
}

func (p *Prometheus) InstallPostgresExporter(pre config.PostgresExporterConf) error {
	inst := services.NewInstallPostgresExporter()
	return inst.Run(pre, "")
}

func (p *Prometheus) InstallRedisExporter(pre config.RedisExporterConf) error {
	inst := services.NewInstallRedisExporter()
	return inst.Run(pre, "")
}

func (p *Prometheus) InstallMongodbExporter(pre config.MongodbExporterConf) error {
	inst := services.NewInstallMongodbExporter()
	return inst.Run(pre, "")
}

func (p *Prometheus) InstallMariadbExporter(pre config.MariadbExporterConf) error {
	inst := services.NewInstallMariadbExporter()
	return inst.Run(pre, "")
}

func (p *Prometheus) RegisterExporter(cfg config.RegisterExporterConf) error {
	inst := services.NewRegisterExporter(&cfg)
	return inst.Run()
}

func (p *Prometheus) DeRegisterExporter(cfg config.RegisterExporterConf) error {
	inst := services.NewRegisterExporter(&cfg)
	return inst.RunDeregister()
}
