package service

import (
	"dbup/internal/environment"
	"dbup/internal/global"
	"dbup/internal/mongodb/config"
	"dbup/internal/utils"
	"dbup/internal/utils/arrlib"
	"dbup/internal/utils/command"
	"dbup/internal/utils/logger"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/ini.v1"
)

type MongoSInstall struct {
	Option          *config.MongosOptions
	Config          *config.MongoSConfig
	Service         *config.MongoDBService
	KeyFileContent  string
	Owner           string
	SysUser         string
	SysGroup        string
	PackageFullName string
}

func NewMongoSInstall(option *config.MongosOptions) *MongoSInstall {
	return &MongoSInstall{
		Option:          option,
		PackageFullName: filepath.Join(environment.GlobalEnv().ProgramPath, global.PackagePath, config.Kinds, fmt.Sprintf(config.PackageFile, config.DefaultMongoDBVersion, environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH)),
	}
}

// 检查service文件是否存在, 如果存在退出安装, 检查mongodb安装包是否存在且md5正确
func (i *MongoSInstall) CheckEnv() error {
	if err := i.Option.CheckEnv(); err != nil {
		return err
	}
	serviceFileName := fmt.Sprintf(config.ServiceFileName, i.Option.Port)
	serviceFileFullName := filepath.Join(global.ServicePath, serviceFileName)
	if utils.IsExists(serviceFileFullName) {
		return fmt.Errorf("启动文件(%s)已经存在, 停止安装", serviceFileFullName)
	}

	if err := i.GetOwner(); err != nil {
		return err
	}

	return global.CheckPackage(environment.GlobalEnv().ProgramPath, i.PackageFullName, config.Kinds)
}

// 确定加入集群用哪个IP
func (i *MongoSInstall) GetOwner() error {
	h, e := os.Hostname()
	if e != nil {
		return fmt.Errorf("获取主机名失败")
	}
	if i.Option.Ipv6 {
		if i.Option.Owner != h {
			return fmt.Errorf("开启IPV6部署功能需要指定本地主机名进行mongodb通信")
		}

		if err := i.Option.CheckIPV6(); err != nil {
			return err
		}
	}

	ips, err := utils.LocalIP()
	if err != nil {
		return err
	}

	if i.Option.Owner == "" {
		if len(ips) == 1 {
			i.Owner = ips[0]
		} else {
			return fmt.Errorf("本机配置了多个IP地址, 请通过参数 --owner 手动指定使用哪个IP地址进行mongodb通信")
		}
	} else {
		if err := utils.IsIP(i.Option.Owner); err != nil {

			if i.Option.Owner == h {
				i.Owner = i.Option.Owner
				return nil
			} else {
				return fmt.Errorf("参数 --owner 不是正确的IP地址格式, 也不是本机主机名")
			}
		}

		if arrlib.InArray(i.Option.Owner, ips) {
			i.Owner = i.Option.Owner
		} else {
			return fmt.Errorf("参数 --owner 手动指定的IP地址, 不是本机配置的IP地址, 请指定正确的本机地址")
		}
	}
	return nil
}

// 检查service文件是否存在, 如果存在退出安装, 检查mongodb安装包是否存在且md5正确
func (i *MongoSInstall) GetKeyFileContent(filename string) (string, error) {
	cfg, err := ini.LoadSources(ini.LoadOptions{
		SpaceBeforeInlineComment: true,
	}, filename)
	if err != nil {
		return "", fmt.Errorf("获取keyfile信息失败: %v", err)
	}

	s := cfg.Section(config.Kinds).Key("key_file").MustString("")
	return s, nil
}

func (i *MongoSInstall) Run() error {
	if err := i.HandleArgs(); err != nil {
		return err
	}
	if !i.Option.Yes {
		var yes string
		logger.Successf("MongoS 端口: %d\n", i.Option.Port)
		logger.Successf("MongoS 安装路径: %s\n", i.Option.Dir)
		logger.Successf("是否确认安装[y|n]:")
		if _, err := fmt.Scanln(&yes); err != nil {
			return err
		}
		if strings.ToUpper(yes) != "Y" && strings.ToUpper(yes) != "YES" {
			os.Exit(0)
		}
	}

	if err := i.InstallAndInitDB(); err != nil {
		if !i.Option.NoRollback {
			logger.Warningf("安装失败, 开始回滚\n")
			uninstall := MongoDBUNInstall{Port: i.Option.Port, BasePath: i.Option.Dir}
			uninstall.Uninstall()
		}
		return err
	}

	// 整个过程结束，生成连接信息文件, 并返回MongoDB用户名、密码、授权IP
	i.Info()
	return nil
}

func (i *MongoSInstall) InstallAndInitDB() error {
	service := fmt.Sprintf(config.ServiceFileName, i.Option.Port)
	if err := i.Install(service); err != nil {
		return err
	}
	logger.Infof("启动实例\n")
	if err := command.SystemCtl(service, "start"); err != nil {
		return err
	}
	logger.Infof("初始化\n")

	return nil
}

func (i *MongoSInstall) HandleArgs() error {
	var err error
	i.SysUser = i.Option.SystemUser
	i.SysGroup = i.Option.SystemGroup

	if i.KeyFileContent, err = i.GetKeyFileContent(filepath.Join(environment.GlobalEnv().ProgramPath, global.PackagePath, global.Md5FileName)); err != nil {
		return err
	}

	i.Config = config.NewMongoSConfig(i.Option)
	if i.Service, err = config.NewMongoDBService(filepath.Join(environment.GlobalEnv().ProgramPath, global.ServiceTemplatePath, config.MongoDBServiceTemplateFile)); err != nil {
		return err
	}
	return i.Service.FormatMongosBody(i.Option, i.SysUser, i.SysGroup)
}

func (i *MongoSInstall) Mkdir() error {
	logger.Infof("创建数据目录和程序目录\n")
	if err := os.MkdirAll(environment.GlobalEnv().DbupInfoPath, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(i.Option.Dir, config.DefaultMongoDBConfigDir), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(i.Option.Dir, config.DefaultMongoDBLogDir), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(i.Option.Dir, config.DefaultMongoDBDataDir), 0755); err != nil {
		return err
	}

	return nil
}

func (i *MongoSInstall) ChownDir(path string) error {
	cmd := fmt.Sprintf("chown -R %s:%s %s", i.SysUser, i.SysGroup, path)
	l := command.Local{}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("修改数据目录所属用户失败: %v, 标准错误输出: %s", err, stderr)
	}
	return nil
}

func (i *MongoSInstall) Install(service string) error {
	logger.Infof("开始安装\n")
	serviceFile := filepath.Join(global.ServicePath, service)

	// 创建子目录
	if err := i.Mkdir(); err != nil {
		return err
	}

	// 解压安装包
	logger.Infof("解压安装包: %s 到 %s \n", i.PackageFullName, i.Option.Dir)
	if err := utils.UntarGz(i.PackageFullName, i.Option.Dir); err != nil {
		return err
	}

	// 检查依赖
	if missLibs, err := global.Checkldd(filepath.Join(i.Option.Dir, config.DefaultMongoDBBinDir, config.DefaultMongoSBinFile)); err != nil {
		return err
	} else {
		if len(missLibs) != 0 {
			errInfo := ""
			for _, missLib := range missLibs {
				errInfo = errInfo + fmt.Sprintf("%s, 缺少: %s, 需要: %s\n", missLib.Info, missLib.Name, missLib.Repair)
			}
			return errors.New(errInfo)
		}
	}

	if err := global.YAMLSaveToFile(filepath.Join(i.Option.Dir, config.DefaultMongoDBConfigDir, config.DefaultMongoSConfigFile), i.Config); err != nil {
		return err
	}

	if err := utils.WriteToFile(filepath.Join(i.Option.Dir, "data", "keyfile"), i.KeyFileContent); err != nil {
		return err
	}

	if err := os.Chmod(filepath.Join(i.Option.Dir, "data", "keyfile"), 0400); err != nil {
		return err
	}

	if err := i.ChownDir(i.Option.Dir); err != nil {
		return err
	}

	// 生成 service 启动文件
	//if err := global.INISaveToFile(serviceFile, i.Service); err != nil {
	//	return err
	//}
	if err := i.Service.SaveTo(serviceFile); err != nil {
		return err
	}

	// service reload 并 设置开机自启动
	if err := command.SystemdReload(); err != nil {
		return err
	}

	logger.Infof("设置开机自启动\n")
	if err := command.SystemCtl(service, "enable"); err != nil {
		return err
	}

	if i.Option.ResourceLimit != "" {
		logger.Infof("设置资源限制启动\n")
		if err := command.SystemResourceLimit(service, i.Option.ResourceLimit); err != nil {
			return err
		}
	}

	return nil
}

func (i *MongoSInstall) Info() {
	var ip string

	if i.Option.BindIP == "0.0.0.0" || i.Option.BindIP == "0.0.0.0,::" {
		ip = "127.0.0.1"
	} else {
		ip = i.Option.BindIP
	}
	logger.Successf("\n")
	logger.Successf("MongoS 初始化[完成]\n")
	logger.Successf("MongoS 端 口:%d\n", i.Option.Port)
	logger.Successf("数据目录:%s\n", i.Option.Dir)
	logger.Successf("启动用户:%s\n", i.SysUser)
	logger.Successf("启动方式:systemctl start %s\n", fmt.Sprintf(config.ServiceFileName, i.Option.Port))
	logger.Successf("关闭方式:systemctl stop %s\n", fmt.Sprintf(config.ServiceFileName, i.Option.Port))
	logger.Successf("重启方式:systemctl restart %s\n", fmt.Sprintf(config.ServiceFileName, i.Option.Port))
	logger.Successf("登录命令: %s --authenticationDatabase admin -u %s -p '%s' --host %s --port %d\n", filepath.Join(i.Option.Dir, "bin", "mongo"), i.Option.Username, i.Option.Password, ip, i.Option.Port)
}
