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

// 安装 mongodb exporter
type InstallMongodbExporter struct {
	prepare  *config.MongodbExporterConf
	Port     int
	basePath string
}

func NewInstallMongodbExporter() *InstallMongodbExporter {
	return &InstallMongodbExporter{
		prepare: &config.MongodbExporterConf{},
	}
}

func (i *InstallMongodbExporter) Run(pre config.MongodbExporterConf, cfgFile string) error {
	// 初始化参数和配置环节
	if err := i.HandlePrepareArgs(pre, cfgFile); err != nil {
		return err
	}
	i.HandleArgs()

	if err := i.Install(); err != nil {
		return err
	}

	i.Info(config.MongodbExporter, i.Port)

	// @todo info 记录到文件
	return nil
}

// 检查命令行配置, 处理安装前准备好的相关参数
func (i *InstallMongodbExporter) HandlePrepareArgs(pre config.MongodbExporterConf, cfgFile string) error {
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

func (i *InstallMongodbExporter) MergePrepareArgs(pre config.MongodbExporterConf) {
	logger.Infof("根据命令行参数调整安装配置\n")

	if pre.Port != 0 {
		i.prepare.Port = pre.Port
	}

	if pre.Dir != "" {
		i.prepare.Dir = pre.Dir
	}

	if pre.MongodbAddr != "" {
		i.prepare.MongodbAddr = pre.MongodbAddr
	}

	if pre.MongodbPassword != "" {
		i.prepare.MongodbPassword = pre.MongodbPassword
	}

	if pre.MongodbUser != "" {
		i.prepare.MongodbUser = pre.MongodbUser
	}
}

func (i *InstallMongodbExporter) HandleArgs() {
	//i.packageFullName = packageName
	//if i.packageFullName == "" {
	//	i.packageFullName = filepath.Join(environment.GlobalEnv().ProgramPath, global.PackagePath, config.Kinds, fmt.Sprintf(config.PackageFile, config.DefaultPrometheusVersion, environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH))
	//}
	//i.ConsulPort = i.prepare.ConsulPort
	i.Port = i.prepare.Port
	i.basePath = i.prepare.Dir
}

func (i *InstallMongodbExporter) getPackageFullName(app string) string {
	packageFullName := filepath.Join(environment.GlobalEnv().ProgramPath, global.PackagePath, config.Kinds, fmt.Sprintf(config.PackageFile, app, environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH))
	return packageFullName
}

func (i *InstallMongodbExporter) getServerPath(app string, port int) string {
	return filepath.Join(i.basePath, fmt.Sprintf("%s%d", app, port))
}

func (i *InstallMongodbExporter) getDataPath(app string) string {
	return filepath.Join(i.basePath, app, "data")
}

func (i *InstallMongodbExporter) getServiceFileName(app string, port int) string {
	return fmt.Sprintf("%s%d.service", app, port)
}

func (i *InstallMongodbExporter) getServiceFileFullName(app string, port int) string {
	return filepath.Join(i.getServerPath(app, port), i.getServiceFileName(app, port))
}
func (i *InstallMongodbExporter) Install() error {
	if !utils.IsExists(i.basePath) {
		if err := os.MkdirAll(i.basePath, 0755); err != nil {
			return err
		}
	}

	port := i.Port
	logger.Infof("开始安装 Mongodb exporter, port:%d \n", port)
	// 创建子目录
	logger.Infof("创建目录\n")
	if err := os.MkdirAll(i.getServerPath(config.MongodbExporter, port), 0755); err != nil {
		return err
	}

	// 解压安装包
	packageFullName := i.getPackageFullName(config.MongodbExporter)
	logger.Infof("解压安装包: %s 到 %s \n", packageFullName, i.getServerPath(config.MongodbExporter, port))
	if err := utils.UntarGz(packageFullName, i.getServerPath(config.MongodbExporter, port)); err != nil {
		return err
	}

	serviceName := i.getServiceFileName(config.MongodbExporter, port)
	logger.Infof("添加 service 文件 %s\n", serviceName)
	body := fmt.Sprintf(config.MongodbExporterService, i.prepare.MongodbUser, i.prepare.MongodbPassword, i.prepare.MongodbAddr, i.basePath, port, port)
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

func (i *InstallMongodbExporter) Info(app string, port int) {
	serviceName := i.getServiceFileName(app, port)
	logger.Successf("\n")
	logger.Successf("生成连接信息文件: %s\n", fmt.Sprintf("/usr/lib/systemd/system/%s", serviceName))
	logger.Successf("%s 初始化[完成]\n", app)
	logger.Successf("%s 端 口:%d\n", app, port)
	logger.Successf("启动方式:systemctl start %s\n", serviceName)
	logger.Successf("关闭方式:systemctl stop %s\n", serviceName)
	logger.Successf("重启方式:systemctl restart %s\n", serviceName)
}
