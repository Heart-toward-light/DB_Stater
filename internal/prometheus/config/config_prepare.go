/*
@Author : WuWeiJian
@Date : 2020-12-16 15:42
*/

package config

import (
	"dbup/internal/environment"
	"dbup/internal/utils"
	"dbup/internal/utils/logger"
	"fmt"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
	"gopkg.in/ini.v1"
)

// 安装时读取的配置文件
type Prepare struct {
	PrometheusPort   int    `ini:"prometheus_port" comment:"监听端口，如果没有特殊要求请勿修改"`
	GrafanaPort      int    `ini:"grafana_port" comment:"监听端口，如果没有特殊要求请勿修改"`
	ConsulPort       int    `ini:"consul_port" comment:"监听端口，如果没有特殊要求请勿修改"`
	NodeExporterPort int    `ini:"node_exporter_port" comment:"监听端口，如果没有特殊要求请勿修改"`
	Dir              string `ini:"dir" comment:"数据部署目录，请确认该目录存在，默认为/opt/prometheus，如无特殊要求请勿修改"`
	GrafanaPassword  string `ini:"grafana_password" comment:"grafana 登录密码"`
	OnlyGrafana      bool   `comment:"是否只安装grafana"`
	WithoutGrafana   bool   `ini:"without-grafana" comment:"是否只安装grafana"`
}

// 初始化生成配置文件
func (p *Prepare) MakeConfigFile(cfgFile string) error {
	var err error
	if cfgFile, err = p.CfgPath(cfgFile); err != nil {
		return err
	}

	if utils.IsExists(cfgFile) {
		return fmt.Errorf("配置文件 ( %s ) 已存在, 请根据需要调整配置\n执行: [ dbup prometheus install --config=%s ] 命令安装监控程序", cfgFile, cfgFile)
	}

	p.InitArgs()
	if err := p.SaveTo(cfgFile); err != nil {
		return err
	}
	fmt.Printf("准备完成, 请根据需要调整配置文件: %s \n", cfgFile)
	fmt.Printf("调整完成后,请执行: [ dbup pgsql install --config=%s ] 进行安装\n", cfgFile)
	return nil
}

// 确定配置文件位置
func (p *Prepare) CfgPath(cfgFile string) (string, error) {
	if cfgFile != "" {
		if !utils.IsExists(cfgFile) {
			return cfgFile, fmt.Errorf("指定的配置文件不存在: %s", cfgFile)
		}
		return cfgFile, nil
	}
	home, err := homedir.Dir()
	if err != nil {
		return "", fmt.Errorf("使用默认配置文件, 获取当前用户家目录失败: %v", err)
	}
	cfgFile = filepath.Join(home, DefaultPrometheusCfgFile)
	return cfgFile, nil
}

// Load 从配置文件加载配置到Prepare实例
func (p *Prepare) Load(filename string) error {
	cfg, err := ini.LoadSources(ini.LoadOptions{
		SpaceBeforeInlineComment: true,
	}, filename)
	if err != nil {
		return fmt.Errorf("加载配置文件失败: %v", err)
	}

	if err = cfg.MapTo(p); err != nil {
		return fmt.Errorf("配置文件映射到结构体失败: %v", err)
	}
	return nil
}

// SaveTo 将Prepare实例数据写入配置文件
func (p *Prepare) SaveTo(filename string) error {
	cfg := ini.Empty(ini.LoadOptions{IgnoreInlineComment: true})
	if err := ini.ReflectFrom(cfg, p); err != nil {
		return fmt.Errorf("部署配置映射到(%s)文件错误: %v", filename, err)
	}
	if err := cfg.SaveTo(filename); err != nil {
		return fmt.Errorf("部署配置保存到(%s)文件错误: %v", filename, err)
	}
	return nil
}

func (p *Prepare) InitArgs() {
	logger.Infof("初始化安装参数\n")
	if p.PrometheusPort == 0 {
		if environment.GlobalEnv().GOARCH == "arm64" {
			p.PrometheusPort = DefaultPrometheusPortArm
		} else {
			p.PrometheusPort = DefaultPrometheusPort
		}
	}

	if p.ConsulPort == 0 {
		p.ConsulPort = DefaultConsulPort
	}

	if p.GrafanaPort == 0 {
		p.GrafanaPort = DefaultGrafanaPort
	}

	if p.NodeExporterPort == 0 {
		p.NodeExporterPort = utils.RandomPort(DefaultNodeExporterPort)
	}

	if p.Dir == "" {
		p.Dir = DefaultPrometheusDir
	}

	if p.GrafanaPassword == "" {
		p.GrafanaPassword = utils.GeneratePasswd(16)
	}
}

func (p *Prepare) Validator() error {
	logger.Infof("验证参数\n")

	if p.OnlyGrafana {
		if err := p.validatePort(p.GrafanaPort); err != nil {
			return err
		}
		return nil
	}

	// 端口
	if err := p.validatePort(p.PrometheusPort); err != nil {
		return err
	}

	//if err := p.validatePort(p.ConsulPort); err != nil {
	//	return err
	//}

	if !p.WithoutGrafana {
		if err := p.validatePort(p.GrafanaPort); err != nil {
			return err
		}
	}

	if err := p.validatePort(p.NodeExporterPort); err != nil {
		return err
	}

	// 数据目录
	if err := utils.ValidatorDir(p.Dir); err != nil {
		return err
	}
	return nil
}

func (p *Prepare) validatePort(port int) error {
	// 端口
	if port < 1025 || port > 65535 {
		return fmt.Errorf("端口号(%d), 不是一个正确的端口号. 端口号必须在 1025 ~ 65535 之间", port)
	}
	if utils.PortInUse(port) {
		return fmt.Errorf("端口号被占用: %d", port)
	}

	return nil
}
