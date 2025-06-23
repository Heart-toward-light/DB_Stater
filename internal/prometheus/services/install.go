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
	"time"

	"github.com/levigross/grequests"
)

// 安装pgsql的总控制逻辑
type Install struct {
	prepare *config.Prepare
	//consulConfig    *config.ConsulConfig
	config *config.PrometheusConfig
	//service         *config.PrometheusService
	PrometheusPort   int
	GrafanaPort      int
	GrafanaPassword  string
	ConsulPort       int
	NodeExporterPort int
	basePath         string
	//prometheusServerPath string
	//grafanaServerPath    string
	//serverPath           string
	//serverFileName       string
	//serverFileFullName   string
	//configPath           string
	//configFileName       string
	//configFileFullName   string
	//dataPath             string
	//servicePath          string
	//serviceFileName      string
	//serviceFileFullName  string
	//version              string
}

func NewInstall() *Install {
	//env, err := environment.NewEnvironment()
	return &Install{
		prepare: &config.Prepare{},
		config:  config.NewPrometheusConfig(),
		//consulConfig: config.NewConsulConfig(),
		//service:        config.NewPrometheusService(),
		//serverFileName: config.ServerFileName,
		//configFileName: config.ConfFileName,
		//servicePath:    global.ServicePath,
		//version:        config.DefaultPrometheusVersion,
	}
}

// func (i *Install) checkIsCentOS7() bool {
// 	if environment.GlobalEnv().HostInfo.Platform != "centos" {
// 		return false
// 	}

// 	ver := environment.GlobalEnv().HostInfo.PlatformVersion
// 	if version.Compare(ver, "7", ">=") {
// 		return true
// 	}

// 	return false
// }

func (i *Install) Run(pre config.Prepare, cfgFile string) error {
	// 验证 centos7
	// if !i.checkIsCentOS7() {
	// 	return fmt.Errorf("操作系统必须在 CentOS7 及以上")
	// }

	// 初始化参数和配置环节
	if err := i.HandlePrepareArgs(pre, cfgFile); err != nil {
		return err
	}
	i.HandleArgs()

	if err := i.Install(); err != nil {
		return err
	}

	// 整个过程结束，生成连接信息文件, 并返回监控相关信息
	//i.Info()

	return nil
}

// 检查命令行配置, 处理安装前准备好的相关参数
func (i *Install) HandlePrepareArgs(pre config.Prepare, cfgFile string) error {
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

func (i *Install) MergePrepareArgs(pre config.Prepare) {
	logger.Infof("根据命令行参数调整安装配置\n")

	if pre.PrometheusPort != 0 {
		i.prepare.PrometheusPort = pre.PrometheusPort
	}

	if pre.GrafanaPort != 0 {
		i.prepare.GrafanaPort = pre.GrafanaPort
	}

	if pre.ConsulPort != 0 {
		i.prepare.ConsulPort = pre.ConsulPort
	}

	if pre.NodeExporterPort != 0 {
		i.prepare.NodeExporterPort = pre.NodeExporterPort
	}

	if pre.Dir != "" {
		i.prepare.Dir = pre.Dir
	}

	if pre.GrafanaPassword != "" {
		i.prepare.GrafanaPassword = pre.GrafanaPassword
	}

	i.prepare.OnlyGrafana = pre.OnlyGrafana
}

func (i *Install) HandleArgs() {
	//i.packageFullName = packageName
	//if i.packageFullName == "" {
	//	i.packageFullName = filepath.Join(environment.GlobalEnv().ProgramPath, global.PackagePath, config.Kinds, fmt.Sprintf(config.PackageFile, config.DefaultPrometheusVersion, environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH))
	//}
	i.PrometheusPort = i.prepare.PrometheusPort
	//i.ConsulPort = i.prepare.ConsulPort
	i.GrafanaPort = i.prepare.GrafanaPort
	i.NodeExporterPort = i.prepare.NodeExporterPort

	i.basePath = i.prepare.Dir
	i.GrafanaPassword = i.prepare.GrafanaPassword
}

func (i *Install) getPackageFullName(app string) string {
	packageFullName := filepath.Join(environment.GlobalEnv().ProgramPath, global.PackagePath, config.Kinds, fmt.Sprintf(config.PackageFile, app, environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH))
	return packageFullName
}

func (i *Install) getServerPath(app string) string {
	return filepath.Join(i.basePath, app)
}

func (i *Install) getDataPath(app string) string {
	return filepath.Join(i.basePath, app, "data")
}

func (i *Install) getServiceFileName(app string, port int) string {
	if app == config.Grafana {
		return fmt.Sprintf("%s.service", app)
	}
	return fmt.Sprintf("%s%d.service", app, port)
}

func (i *Install) getServiceFileFullName(app string, port int) string {
	return filepath.Join(i.getServerPath(app), i.getServiceFileName(app, port))
}
func (i *Install) Install() error {
	if err := os.MkdirAll(i.basePath, 0755); err != nil {
		return err
	}

	if err := os.MkdirAll(environment.GlobalEnv().DbupInfoPath, 0755); err != nil {
		return err
	}

	if i.prepare.OnlyGrafana {
		if err := i.installGrafana(); err != nil {
			return err
		}

		return nil
	}

	//if err := i.installConsul(); err != nil {
	//	return err
	//}

	if err := i.installPrometheus(); err != nil {
		return err
	}

	if !i.prepare.WithoutGrafana {
		if err := i.installGrafana(); err != nil {
			return err
		}
	}

	//if err := i.installNodeExporter(); err != nil {
	//	return err
	//}

	return nil
}

func (i *Install) installConsul() error {
	port := i.ConsulPort
	logger.Infof("开始安装 consul, port: %d\n", port)

	// 解压安装包
	packageFullName := i.getPackageFullName(config.Consul)
	logger.Infof("解压安装包: %s 到 %s \n", packageFullName, i.basePath)
	if err := utils.UntarGz(packageFullName, i.basePath); err != nil {
		return err
	}

	//// 生成 prometheus.yml 数据库的配置文件
	//if err := i.MakeConfigFile(i.configFileFullName); err != nil {
	//	return err
	//}
	//
	// TODO
	logger.Infof("添加 service 文件\n")
	serviceName := i.getServiceFileName(config.Consul, port)
	body := fmt.Sprintf(config.ConsulService, i.basePath, i.basePath, i.basePath)
	filename := fmt.Sprintf("/usr/lib/systemd/system/%s", serviceName)
	if err := ioutil.WriteFile(filename, []byte(body), 0644); err != nil {
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

	i.Info(config.Consul, port)
	return nil
}

func (i *Install) MakeConfigFile(filename string) error {
	logger.Infof("创建配置文件: %s\n", filename)

	if utils.IsExists(filename) {
		if err := command.MoveFile(filename); err != nil {
			return err
		}
	}
	return i.config.SaveTo(filename)
}

func (i *Install) Info(app string, port int) {
	filename := filepath.Join(environment.GlobalEnv().DbupInfoPath, fmt.Sprintf("%s%d", app, port))

	if app == config.Grafana {
		info := config.GrafanaInfo{
			Port:        port,
			User:        "admin",
			Password:    i.GrafanaPassword,
			InstallPath: i.getServerPath(app),
		}
		if err := info.SaveTo(filename); err != nil {
			logger.Errorf("写入配置文件 %s 失败:%s", filename, err.Error())
		}
	} else if app == config.Prometheus {
		info := config.PrometheusInfo{
			Port:        port,
			InstallPath: i.getServerPath(app),
		}
		if err := info.SaveTo(filename); err != nil {
			logger.Errorf("写入配置文件 %s 失败:%s", filename, err.Error())
		}
	}

	logger.Successf("\n")
	logger.Successf("生成连接信息文件: %s\n", filename)
	logger.Successf("%s 初始化[完成]\n", app)
	logger.Successf("%s 端 口:%d\n", app, port)
	if app == config.Grafana {
		logger.Successf("%s 账号:admin,密码:%s\n", app, i.GrafanaPassword)
	}
	logger.Successf("启动方式:systemctl start %s\n", i.getServiceFileName(app, port))
	logger.Successf("关闭方式:systemctl stop %s\n", i.getServiceFileName(app, port))
	logger.Successf("重启方式:systemctl restart %s\n", i.getServiceFileName(app, port))
}

func (i *Install) installPrometheus() (err error) {
	port := i.PrometheusPort
	logger.Infof("开始安装 prometheus,port:%d \n", port)

	// 添加 consul 的 host
	if err := i.addConsulHost(); err != nil {
		return err
	}

	// 解压安装包
	packageFullName := i.getPackageFullName(config.Prometheus)
	logger.Infof("解压安装包: %s 到 %s \n", packageFullName, i.basePath)
	if err := utils.UntarGz(packageFullName, i.basePath); err != nil {
		return err
	}

	logger.Infof("添加 service 文件\n")
	serviceName := i.getServiceFileName(config.Prometheus, port)
	body := fmt.Sprintf(config.PrometheusService, i.basePath, i.basePath, i.basePath, i.basePath, i.basePath, port)
	filename := fmt.Sprintf("/usr/lib/systemd/system/%s", serviceName)
	if err := ioutil.WriteFile(filename, []byte(body), 0644); err != nil {
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

	i.Info(config.Prometheus, port)
	return nil
}

func (i *Install) installGrafana() (err error) {
	port := i.GrafanaPort
	logger.Infof("开始安装 grafana, port:%d \n", port)

	// 解压安装包
	packageFullName := i.getPackageFullName(config.Grafana)
	logger.Infof("解压安装包: %s 到 %s \n", packageFullName, i.basePath)
	if err := utils.UntarGz(packageFullName, i.basePath); err != nil {
		return err
	}

	logger.Infof("添加 service 文件\n")
	serviceName := i.getServiceFileName(config.Grafana, port)
	body := fmt.Sprintf(config.GrafanaService, i.basePath, i.basePath, i.basePath)
	filename := fmt.Sprintf("/usr/lib/systemd/system/%s", serviceName)
	if err := ioutil.WriteFile(filename, []byte(body), 0644); err != nil {
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

	times := 1

	for {
		time.Sleep(time.Millisecond * 100)

		if utils.PortInUse(i.GrafanaPort) {
			break
		}

		times++
		if times >= 1000 {
			return fmt.Errorf("grafana 启动超时")
		}
	}

	// 更新密码
	chUrl := fmt.Sprintf("http://127.0.0.1:%d/api/user/password", i.GrafanaPort)

	_, err = grequests.Put(chUrl, &grequests.RequestOptions{
		Auth:    []string{"admin", "qianxin"},
		Headers: map[string]string{"Content-Type": "application/json"},
		JSON: map[string]string{
			"oldPassword": "qianxin",
			"newPassword": i.GrafanaPassword,
			"confirmNew":  i.GrafanaPassword,
		},
	})

	if err != nil {
		return fmt.Errorf("初始化 grafana 密码失败,err:%s", err.Error())
	}

	i.Info(config.Grafana, port)
	return nil

}

func (i *Install) installNodeExporter() (err error) {
	port := i.NodeExporterPort
	logger.Infof("开始安装 Node exporter, port:%d \n", port)
	// 创建子目录
	//logger.Infof("创建目录\n")
	//if err := os.MkdirAll(i.getServerPath(config.Consul), 0755); err != nil {
	//	return err
	//}
	//if err := os.MkdirAll(i.dataPath, 0755); err != nil {
	//	return err
	//}
	//if err := os.MkdirAll(i.configPath, 0755); err != nil {
	//	return err
	//}

	// 解压安装包
	packageFullName := i.getPackageFullName(config.NodeExporter)
	logger.Infof("解压安装包: %s 到 %s \n", packageFullName, i.basePath)
	if err := utils.UntarGz(packageFullName, i.basePath); err != nil {
		return err
	}

	//// 生成 prometheus.yml 数据库的配置文件
	//if err := i.MakeConfigFile(i.configFileFullName); err != nil {
	//	return err
	//}
	//
	logger.Infof("添加 service 文件\n")
	serviceName := i.getServiceFileName(config.NodeExporter, port)
	body := fmt.Sprintf(config.NodeExporterService, i.basePath, port)
	filename := fmt.Sprintf("/usr/lib/systemd/system/%s", serviceName)
	if err := ioutil.WriteFile(filename, []byte(body), 0644); err != nil {
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

	i.Info(config.NodeExporter, port)
	return nil
}

func (i *Install) addConsulHost() (err error) {
	cmd := fmt.Sprintf("echo '127.0.0.1 consul01' >>  /etc/hosts")
	l := command.Local{}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("执行(%s)失败: %v, 标准错误输出: %s", cmd, err, stderr)
	}
	return nil
}
