package services

import (
	"dbup/internal/environment"
	"dbup/internal/global"
	"dbup/internal/prometheus/config"
	"dbup/internal/utils"
	"dbup/internal/utils/command"
	"dbup/internal/utils/logger"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// 安装 Mariadb exporter
type InstallMariadbExporter struct {
	prepare  *config.MariadbExporterConf
	Port     int
	basePath string
}

func NewInstallMariadbExporter() *InstallMariadbExporter {
	return &InstallMariadbExporter{
		prepare: &config.MariadbExporterConf{},
	}
}

func (i *InstallMariadbExporter) Run(pre config.MariadbExporterConf, cfgFile string) error {
	// 初始化参数和配置环节
	if err := i.HandlePrepareArgs(pre, cfgFile); err != nil {
		return err
	}
	i.HandleArgs()

	if err := i.Install(); err != nil {
		return err
	}

	i.Info(config.MariadbExporter, i.Port)

	// @todo info 记录到文件
	return nil
}

// 检查命令行配置, 处理安装前准备好的相关参数
func (i *InstallMariadbExporter) HandlePrepareArgs(pre config.MariadbExporterConf, cfgFile string) error {
	if cfgFile != "" {
		if utils.IsExists(cfgFile) {
			logger.Infof("从配置文件中获取安装配置: %s\n", cfgFile)
			if err := i.prepare.Load(cfgFile); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("指定的配置文件不存在: %s", cfgFile)
		}
	}

	i.prepare.InitArgs()
	i.MergePrepareArgs(pre)
	if err := i.prepare.Validator(); err != nil {
		return err
	}
	return nil
}

func (i *InstallMariadbExporter) MergePrepareArgs(pre config.MariadbExporterConf) {
	logger.Infof("根据命令行参数调整安装配置\n")

	if pre.Port != 0 {
		i.prepare.Port = pre.Port
	}

	if pre.Dir != "" {
		i.prepare.Dir = pre.Dir
	}

	if pre.MariadbAddr != "" {
		i.prepare.MariadbAddr = pre.MariadbAddr
	}

	if pre.MariadbPort != 0 {
		i.prepare.MariadbPort = pre.MariadbPort
	}

	if pre.MariadbPassword != "" {
		i.prepare.MariadbPassword = pre.MariadbPassword
	}

	if pre.MariadbUser != "" {
		i.prepare.MariadbUser = pre.MariadbUser
	}
}

func (i *InstallMariadbExporter) HandleArgs() {
	//i.packageFullName = packageName
	//if i.packageFullName == "" {
	//	i.packageFullName = filepath.Join(environment.GlobalEnv().ProgramPath, global.PackagePath, config.Kinds, fmt.Sprintf(config.PackageFile, config.DefaultPrometheusVersion, environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH))
	//}
	//i.ConsulPort = i.prepare.ConsulPort
	i.Port = i.prepare.Port
	i.basePath = i.prepare.Dir
}

func (i *InstallMariadbExporter) getPackageFullName(app string) string {
	packageFullName := filepath.Join(environment.GlobalEnv().ProgramPath, global.PackagePath, config.Kinds, fmt.Sprintf(config.PackageFile, app, environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH))
	return packageFullName
}

func (i *InstallMariadbExporter) getServerPath(app string, port int) string {
	return filepath.Join(i.basePath, fmt.Sprintf("%s%d", app, port))
}

func (i *InstallMariadbExporter) getServiceFileName(app string, port int) string {
	return fmt.Sprintf("%s%d.service", app, port)
}

func (i *InstallMariadbExporter) getConfigFileName(port int) string {
	return fmt.Sprintf("my%d.cnf", port)
}

func (i *InstallMariadbExporter) Install() error {
	if !utils.IsExists(i.basePath) {
		if err := os.MkdirAll(i.basePath, 0755); err != nil {
			return err
		}
	}

	port := i.Port
	logger.Infof("开始安装 Mariadb exporter, port:%d \n", port)
	// 创建子目录
	logger.Infof("创建目录\n")
	if err := os.MkdirAll(i.getServerPath(config.MariadbExporter, port), 0755); err != nil {
		return err
	}

	// 解压安装包
	packageFullName := i.getPackageFullName(config.MariadbExporter)
	logger.Infof("解压安装包: %s 到 %s \n", packageFullName, i.getServerPath(config.MariadbExporter, port))

	if err := utils.UntarGz(packageFullName, i.getServerPath(config.MariadbExporter, port)); err != nil {
		return err
	}

	configname := i.getConfigFileName(i.prepare.MariadbPort)
	logger.Infof("添加配置文件 %s\n", configname)
	configbody := fmt.Sprintf(config.MariadbExporterConfig, i.prepare.MariadbUser, i.prepare.MariadbPassword, i.prepare.MariadbPort, i.prepare.MariadbAddr)
	configfile := fmt.Sprintf("%s/%s%d/%s", i.basePath, config.MariadbExporter, port, configname)
	if err := ioutil.WriteFile(configfile, []byte(configbody), 0755); err != nil {
		return err
	}

	serviceName := i.getServiceFileName(config.MariadbExporter, port)
	logger.Infof("添加 service 文件 %s\n", serviceName)
	body := fmt.Sprintf(config.MariadbExporterService, i.basePath, port, port, configfile)
	filename := fmt.Sprintf("/usr/lib/systemd/system/%s", serviceName)
	if err := ioutil.WriteFile(filename, []byte(body), 0755); err != nil {
		return err
	}

	// service reload 并 设置开机自启动
	if err := command.SystemdReload(); err != nil {
		return err
	}

	logger.Infof("设置开机自启动, 并启动实例\n")
	if err := command.SystemCtl(serviceName, "enable"); err != nil {
		return err
	}

	// 启动
	if err := command.SystemCtl(serviceName, "start"); err != nil {
		return err
	}

	return nil
}

func (i *InstallMariadbExporter) Info(app string, port int) {
	serviceName := i.getServiceFileName(app, port)
	logger.Successf("\n")
	logger.Successf("生成连接信息文件: %s\n", fmt.Sprintf("/usr/lib/systemd/system/%s", serviceName))
	logger.Successf("%s 初始化[完成]\n", app)
	logger.Successf("%s 端 口:%d\n", app, port)
	logger.Successf("启动方式:systemctl start %s\n", serviceName)
	logger.Successf("关闭方式:systemctl stop %s\n", serviceName)
	logger.Successf("重启方式:systemctl restart %s\n", serviceName)
}
