/*
@Author : WuWeiJian
@Date : 2020-12-16 17:05
*/

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

// 安装节点 exporter
type InstallNodeExporter struct {
	prepare *config.NodeExporterConf
	//consulConfig    *config.ConsulConfig
	Port     int
	basePath string
}

func NewInstallNodeExporter() *InstallNodeExporter {
	return &InstallNodeExporter{
		prepare: &config.NodeExporterConf{},
	}
}

func (i *InstallNodeExporter) Run(pre config.NodeExporterConf, cfgFile string) error {
	// 初始化参数和配置环节
	if err := i.HandlePrepareArgs(pre, cfgFile); err != nil {
		return err
	}
	i.HandleArgs()

	if err := i.Install(); err != nil {
		return err
	}

	// 整个过程结束，生成连接信息文件, 并返回监控相关信息

	i.Info(config.NodeExporter, i.Port)
	return nil
}

// 检查命令行配置, 处理安装前准备好的相关参数
func (i *InstallNodeExporter) HandlePrepareArgs(pre config.NodeExporterConf, cfgFile string) error {
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

func (i *InstallNodeExporter) MergePrepareArgs(pre config.NodeExporterConf) {
	logger.Infof("根据命令行参数调整安装配置\n")

	if pre.Port != 0 {
		i.prepare.Port = pre.Port
	}

	if pre.Dir != "" {
		i.prepare.Dir = pre.Dir
	}
}

func (i *InstallNodeExporter) HandleArgs() {
	//i.packageFullName = packageName
	//if i.packageFullName == "" {
	//	i.packageFullName = filepath.Join(environment.GlobalEnv().ProgramPath, global.PackagePath, config.Kinds, fmt.Sprintf(config.PackageFile, config.DefaultPrometheusVersion, environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH))
	//}
	//i.ConsulPort = i.prepare.ConsulPort
	i.Port = i.prepare.Port
	i.basePath = i.prepare.Dir
}

func (i *InstallNodeExporter) getPackageFullName(app string) string {
	packageFullName := filepath.Join(environment.GlobalEnv().ProgramPath, global.PackagePath, config.Kinds, fmt.Sprintf(config.PackageFile, app, environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH))
	return packageFullName
}

func (i *InstallNodeExporter) getServerPath(app string) string {
	return filepath.Join(i.basePath, fmt.Sprintf("%s_%s", app, "dbup"))
}

func (i *InstallNodeExporter) getDataPath(app string) string {
	return filepath.Join(i.basePath, app, "data")
}

func (i *InstallNodeExporter) getServiceFileName(app string) string {
	return fmt.Sprintf("%s_dbup.service", app)
}

func (i *InstallNodeExporter) getServiceFileFullName(app string, port int) string {
	return filepath.Join(i.getServerPath(app), i.getServiceFileName(app))
}
func (i *InstallNodeExporter) Install() error {
	if !utils.IsExists(i.basePath) {
		if err := os.MkdirAll(i.basePath, 0755); err != nil {
			return err
		}
	}

	port := i.Port
	logger.Infof("开始安装 Node exporter, port:%d \n", port)

	serverPath := i.getServerPath(config.NodeExporter)
	if utils.IsExists(serverPath) {
		return fmt.Errorf("当前主机已安装 node_exporter，再次安装需要先执行 systemctl stop %s && rm -rf %s", i.getServiceFileName(config.NodeExporter), serverPath)
	}
	// 创建子目录
	logger.Infof("创建目录\n")
	if err := os.MkdirAll(serverPath, 0755); err != nil {
		return err
	}
	//if err := os.MkdirAll(i.dataPath, 0755); err != nil {
	//	return err
	//}
	//if err := os.MkdirAll(i.configPath, 0755); err != nil {
	//	return err
	//}

	// 解压安装包
	packageFullName := i.getPackageFullName(config.NodeExporter)
	logger.Infof("解压安装包: %s 到 %s \n", packageFullName, serverPath)
	if err := utils.UntarGz(packageFullName, serverPath); err != nil {
		return err
	}

	logger.Infof("添加 service 文件\n")
	serviceName := i.getServiceFileName(config.NodeExporter)
	body := fmt.Sprintf(config.NodeExporterService, i.basePath, port)
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

//func (i *InstallNodeExporter) MakeConfigFile(filename string) error {
//	logger.Infof("创建配置文件: %s\n", filename)
//
//	if utils.IsExists(filename) {
//		if err := command.MoveFile(filename); err != nil {
//			return err
//		}
//	}
//	return i.config.SaveTo(filename)
//}

func (i *InstallNodeExporter) Info(app string, port int) {
	serviceName := i.getServiceFileName(app)
	logger.Successf("\n")
	logger.Successf("生成连接信息文件: %s\n", fmt.Sprintf("/usr/lib/systemd/system/%s", serviceName))
	logger.Successf("%s 初始化[完成]\n", app)
	logger.Successf("%s 端 口:%d\n", app, port)
	logger.Successf("启动方式:systemctl start %s\n", serviceName)
	logger.Successf("关闭方式:systemctl stop %s\n", serviceName)
	logger.Successf("重启方式:systemctl restart %s\n", serviceName)
}
