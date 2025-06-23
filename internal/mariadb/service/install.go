// Created by LiuSainan on 2022-06-09 11:53:03

package service

import (
	"bytes"
	"dbup/internal/environment"
	"dbup/internal/global"
	"dbup/internal/mariadb/config"
	"dbup/internal/mariadb/dao"
	"dbup/internal/utils"
	"dbup/internal/utils/command"
	"dbup/internal/utils/logger"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// 安装pgsql的总控制逻辑
type MariaDBInstall struct {
	Option *config.MariaDBOptions
	// Goption         *config.MariaDBGaleraOptions
	Config          *config.MariaDBConfig
	Gconfig         *config.MariaDBGaleraConfig
	Service         *config.MariaDBService
	PackageFullName string
}

func NewMariaDBInstall(option *config.MariaDBOptions) *MariaDBInstall {
	return &MariaDBInstall{
		Option:          option,
		Config:          config.NewMariaDBConfig(),
		Gconfig:         config.NewMariaDBGaleraConfig(),
		PackageFullName: filepath.Join(environment.GlobalEnv().ProgramPath, global.PackagePath, config.Kinds, fmt.Sprintf(config.PackageFile, config.DefaultMariaDBVersion, environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH)),
	}
}

func (i *MariaDBInstall) HandleArgs() error {
	var err error
	if err = global.CheckPackage(environment.GlobalEnv().ProgramPath, i.PackageFullName, config.Kinds); err != nil {
		return err
	}

	if i.Service, err = config.NewMariaDBService(filepath.Join(environment.GlobalEnv().ProgramPath, global.ServiceTemplatePath, config.MariaDBServiceTemplateFile)); err != nil {
		return err
	}

	if i.Option.Join != "" {
		i.Option.Role = config.MariaDBSlaveRole
	}

	if i.Option.Galera {
		if i.Option.Onenode {
			if err = i.Service.GaleraFormatBody(i.Option); err != nil {
				return err
			}
		} else {
			if err = i.Service.FormatBody(i.Option); err != nil {
				return err
			}
		}
		i.Gconfig.HandleGaleraConfig(i.Option)
	} else {
		if err = i.Service.FormatBody(i.Option); err != nil {
			return err
		}
		i.Config.HandleConfig(i.Option)
	}

	return nil
}

func (i *MariaDBInstall) Run() error {
	if err := i.HandleArgs(); err != nil {
		return err
	}

	if !i.Option.Yes {
		var yes string
		// if i.Option.Role == config.MariaDBSlaveRole {
		// 	logger.Successf("\n")
		// 	logger.Successf("本次安装实例为(%s)从节点\n", i.Option.Role)
		// 	logger.Successf("要同步的主节点为: %s\n", i.Option.Join)
		// 	logger.Successf("\n")
		// }
		logger.Successf("端口: %d\n", i.Option.Port)
		logger.Successf("root用户密码: %s\n", i.Option.Password)
		logger.Successf("安装路径: %s\n", i.Option.Dir)
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
			uninstall := MariaDBUNInstall{Port: i.Option.Port, BasePath: i.Option.Dir}
			uninstall.Uninstall()
		}
		return err
	}

	// 整个过程结束，生成连接信息文件, 并返回Mariadb用户名、密码、授权IP
	i.Info()
	return nil
}

func (i *MariaDBInstall) InstallAndInitDB() error {
	service := fmt.Sprintf(config.ServiceFileName, i.Option.Port)
	servicePath := global.ServicePath
	serviceFileFullName := filepath.Join(servicePath, service)
	if err := i.Install(service); err != nil {
		return err
	}
	logger.Infof("启动实例\n")
	if i.Option.Galera {
		if i.Option.Onenode {
			logger.Infof("启动 Galera 第一个节点\n")
			if err := command.SystemCtl(service, "start"); err != nil {
				return err
			}

			time.Sleep(6 * time.Second)
			logger.Infof("初始化 Galera 第一个节点\n")
			if err := i.Changelocalpassword(); err != nil {
				return err
			}
			if err := i.GaleraRemoveSystem(serviceFileFullName); err != nil {
				return err
			}
			if err := command.SystemdReload(); err != nil {
				return err
			}

		} else {
			if err := command.SystemCtl(service, "start"); err != nil {
				return err
			}
			logger.Successf("Galera 节点初始化并完成同步\n")
		}
	} else {
		if err := command.SystemCtl(service, "start"); err != nil {
			return err
		}
		logger.Infof("开始初始化\n")
		switch i.Option.Role {
		case config.MariaDBMasterRole:
			if err := i.InitPrimary(); err != nil {
				return err
			}
		case config.MariaDBSlaveRole:
			if err := i.InitSecondary(); err != nil {
				return err
			}
		}
	}

	return nil
}

// func (i *MariaDBInstall) Galeraclusteraddress() {
// 	Galerabase := []string{}
// 	if i.Option.Galerabaseport != 4567 {
// 		ips := strings.Split(i.Option.Wsrepclusteraddress, ",")
// 		for _, ip := range ips {
// 			galerabaseport := fmt.Sprintf("%s:%d", ip, i.Option.Galerabaseport)
// 			Galerabase = append(Galerabase, galerabaseport)
// 		}
// 	}
// 	i.Option.Wsrepclusteraddress = strings.Join(Galerabase, ",")
// }

func (i *MariaDBInstall) GaleraRemoveSystem(ServiceFileName string) error {

	input, err := ioutil.ReadFile(ServiceFileName)
	if err != nil {
		return err
	}

	output := bytes.Replace(input, []byte("--wsrep-new-cluster"), []byte(" "), -1)

	if err = ioutil.WriteFile(ServiceFileName, output, 0666); err != nil {
		return fmt.Errorf("%s 文件写入异常 %s", ServiceFileName, err)
	}

	return nil
}

func (i *MariaDBInstall) CreateUser() error {
	logger.Infof("创建启动用户: %s\n", i.Option.SystemUser)
	u, err := user.Lookup(i.Option.SystemUser)
	if err == nil { // 如果用户已经存在,则i.adminGroup设置为真正的所属组名
		g, _ := user.LookupGroupId(u.Gid)
		i.Option.SystemGroup = g.Name
		return nil
	}
	// groupadd -f <group-name>
	groupAdd := fmt.Sprintf("%s -f %s", command.GroupAddCmd, i.Option.SystemGroup)

	// useradd -g <group-name> <user-name>
	userAdd := fmt.Sprintf("%s -g %s %s", command.UserAddCmd, i.Option.SystemGroup, i.Option.SystemUser)

	l := command.Local{}
	if _, stderr, err := l.Run(groupAdd); err != nil {
		return fmt.Errorf("创建用户组(%s)失败: %v, 标准错误输出: %s", i.Option.SystemGroup, err, stderr)
	}
	if _, stderr, err := l.Run(userAdd); err != nil {
		return fmt.Errorf("创建用户(%s)失败: %v, 标准错误输出: %s", i.Option.SystemUser, err, stderr)
	}
	return nil
}

func (i *MariaDBInstall) ChownDir(path string) error {
	cmd := fmt.Sprintf("chown -R %s:%s %s", i.Option.SystemUser, i.Option.SystemGroup, path)
	l := command.Local{}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("修改数据目录所属用户失败: %v, 标准错误输出: %s", err, stderr)
	}
	return nil
}

func (i *MariaDBInstall) Mkdir() error {
	logger.Infof("创建数据目录和程序目录\n")
	if err := os.MkdirAll(environment.GlobalEnv().DbupInfoPath, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(i.Option.Dir, 0755); err != nil {
		return err
	}

	return nil
}

func (i *MariaDBInstall) Install(service string) error {
	logger.Infof("开始安装\n")
	// if i.Option.Join != "" {
	// 	i.Option.Role = config.MariaDBSlaveRole
	// }
	serviceFile := filepath.Join(global.ServicePath, service)
	//检查并创建 MariaDB 账号
	if err := i.CreateUser(); err != nil {
		return err
	}

	// 创建目录授权
	if err := i.Mkdir(); err != nil {
		return err
	}

	// 解压安装包
	logger.Infof("解压安装包: %s 到 %s \n", i.PackageFullName, i.Option.Dir)
	if err := utils.UntarGz(i.PackageFullName, i.Option.Dir); err != nil {
		return err
	}

	// 检查主要指定文件依赖
	ExecLib := []string{"mariadb", "mariadbd", "mariadb-dump"}
	for _, cmd := range ExecLib {
		if missLibs, err := global.Checkldd(filepath.Join(i.Option.Dir, "bin", cmd)); err != nil {
			return err
		} else {
			if len(missLibs) != 0 {
				if err := i.LibComplement(missLibs, ExecLib); err != nil {
					return err
				}
				// 替换完依赖等待 6 秒
				time.Sleep(6 * time.Second)
			}
		}
	}

	// 检查 galera 库文件依赖
	// if err := i.Galeralibcheck(); err != nil {
	// 	return err
	// }

	if i.Option.Galera {
		if err := i.Gconfig.GaleraSaveTo(filepath.Join(i.Option.Dir, config.DefaultMariaDBConfigDir, config.DefaultMariaDBConfigFile)); err != nil {
			return err
		}
	} else {
		if err := i.Config.SaveTo(filepath.Join(i.Option.Dir, config.DefaultMariaDBConfigDir, config.DefaultMariaDBConfigFile)); err != nil {
			return err
		}
	}

	if err := i.ChownDir(i.Option.Dir); err != nil {
		return err
	}

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

func (i *MariaDBInstall) LibComplement(NoLiblist []global.MissSoLibrariesfile, ExecLib []string) error {
	Lib8List := []string{"libtinfo.so.5", "libncurses.so.5"}
	SySLibs := []string{"/lib64"}

	// 开始补齐
	for _, missLib := range NoLiblist {
		re := regexp.MustCompile(`\s+`)
		result := re.ReplaceAllString(missLib.Info, "")
		Libname := strings.Split(result, "=")[0]
		for _, s := range Lib8List {
			if strings.Contains(s, Libname) {
				// 检查补齐的lib是否可用
				if missLibs, err := global.Checkldd(filepath.Join(i.Option.Dir, "lib/newlib", s)); err != nil {
					return err
				} else {
					if len(missLibs) != 0 {
						errInfo := ""
						for _, missLib := range missLibs {
							errInfo = errInfo + fmt.Sprintf("NewLib 文件验证: %s, 缺少: %s, 需要: %s\n", missLib.Info, missLib.Name, missLib.Repair)
						}
						return errors.New(errInfo)
					}
				}
				logger.Warningf("安装出现缺失的Lib文件 %s , 尝试开始进行自动补齐\n", Libname)
				Libfullname := filepath.Join(i.Option.Dir, "lib/newlib", Libname)
				for _, syslibpath := range SySLibs {
					syslibfullname := filepath.Join(syslibpath, Libname)
					if !utils.IsExists(syslibfullname) {
						if err := command.CopyFileDir(Libfullname, syslibpath); err != nil {
							return err
						}
						if err := os.Chmod(syslibfullname, 0755); err != nil {
							return err
						}
					}
				}
			}
		}
	}

	// 再次验证执行文件依赖可用性
	for _, Execmd := range ExecLib {
		if missLibs, err := global.Checkldd(filepath.Join(i.Option.Dir, "bin", Execmd)); err != nil {
			return err
		} else {
			if len(missLibs) != 0 {
				errInfo := ""
				for _, missLib := range missLibs {
					errInfo = errInfo + fmt.Sprintf("执行文件验证: %s, 缺少: %s, 需要: %s\n", missLib.Info, missLib.Name, missLib.Repair)
				}
				return errors.New(errInfo)
			}
		}
	}

	return nil
}

func (i *MariaDBInstall) Galeralibcheck() error {

	if i.Option.Galera {
		if galeralibs, err := global.Checkldd(filepath.Join(i.Option.Dir, "lib/galera", "libgalera_smm.so")); err != nil {
			return err
		} else {
			if len(galeralibs) != 0 {
				libsslfile := "libssl.so.1.0.0"
				libcryptofile := "libcrypto.so.1.0.0"
				libsslcmd := fmt.Sprintf("cp %s %s", filepath.Join(i.Option.Dir, "lib/galera", libsslfile), filepath.Join("/lib64", libsslfile))
				libcryptocmd := fmt.Sprintf("cp %s %s", filepath.Join(i.Option.Dir, "lib/galera", libcryptofile), filepath.Join("/lib64", libcryptofile))
				l := command.Local{}
				if _, stderr, err := l.Run(libsslcmd); err != nil {
					return fmt.Errorf("cp命令操作失败: %s, 标准错误输出: %s", err, stderr)
				}
				if _, stderr, err := l.Run(libcryptocmd); err != nil {
					return fmt.Errorf("cp命令操作失败: %s, 标准错误输出: %s", err, stderr)
				}
				if galeralib, err := global.Checkldd(filepath.Join(i.Option.Dir, "lib/galera", "libgalera_smm.so")); err != nil {
					return err
				} else {
					if len(galeralib) != 0 {
						errInfo := ""
						for _, missLib := range galeralib {
							errInfo = errInfo + fmt.Sprintf("%s, 缺少: %s, 需要: %s\n", missLib.Info, missLib.Name, missLib.Repair)
						}
						return errors.New(errInfo)
					}
				}

			}
		}
	}
	return nil
}

func (i *MariaDBInstall) CreateRepluser() error {
	host := config.DefaultMariaDBlocalhost
	conn, err := dao.NewMariaDBConn(host, i.Option.Port, "root", i.Option.Password, "")
	if err != nil {
		return err
	}
	defer conn.DB.Close()

	if err := conn.CreateUser(i.Option.Repluser, "%", i.Option.ReplPassword); err != nil {
		return fmt.Errorf("创建复制账号 %s 失败: %v", i.Option.Repluser, err)
	}

	if err := conn.Grant(i.Option.Repluser, "%", config.MariaDBReplicationPrivileges); err != nil {
		return fmt.Errorf("授权复制账号 %s 失败: %v", i.Option.Repluser, err)
	}

	return nil
}

func (i *MariaDBInstall) CreateBackupuser() error {
	host := config.DefaultMariaDBlocalhost
	conn, err := dao.NewMariaDBConn(host, i.Option.Port, "root", i.Option.Password, "")
	if err != nil {
		return err
	}
	defer conn.DB.Close()

	if err := conn.CreateUser(i.Option.Backupuser, "%", i.Option.BackupPassword); err != nil {
		return fmt.Errorf("创建备份账号 %s 失败: %v", i.Option.Repluser, err)
	}

	if err := conn.Grant(i.Option.Backupuser, "%", config.MariaDBBackupPrivileges); err != nil {
		return fmt.Errorf("授权备份账号 %s 失败: %v", i.Option.Repluser, err)
	}

	return nil
}

func (i *MariaDBInstall) Changelocalpassword() error {
	host := config.DefaultMariaDBlocalhost
	logger.Infof("等待实例 initialization buffer 状态\n")

	conn, err := dao.NewMariaDBConn(host, i.Option.Port, "root", config.DefaultMariaDBRootPassword, "")
	if err != nil {
		return err
	}
	defer conn.DB.Close()

	if err := conn.ChangePassword("root", "localhost", i.Option.Password); err != nil {
		if err := conn.ChangePassword("root", "localhost", i.Option.Password); err != nil {
			return fmt.Errorf("修改 root@localhost 密码报错: %v", err)
		}
	}

	if err := conn.ChangePassword("root", "127.0.0.1", i.Option.Password); err != nil {
		return fmt.Errorf("修改 root@127.0.0.1 密码报错: %v", err)
	}

	return nil
}

func (i *MariaDBInstall) ChangeMaster() error {
	host := config.DefaultMariaDBlocalhost
	conn, err := dao.NewMariaDBConn(host, i.Option.Port, "root", i.Option.Password, "")
	if err != nil {
		return err
	}
	defer conn.DB.Close()

	if i.Option.AddSlave && i.Option.BackupData {
		if err := conn.FlushUser(); err != nil {
			return fmt.Errorf("刷新权限表失败")
		}
	}

	if err := conn.ChangeMasterTo(i.Option.Join, i.Option.Port, i.Option.Repluser, i.Option.ReplPassword); err != nil {
		return fmt.Errorf("创建主从关系 %s 失败: %v", i.Option.Repluser, err)
	}

	if err := conn.StartSlave(); err != nil {
		return fmt.Errorf("启动从同步 %s 失败: %v", i.Option.Repluser, err)
	}

	return nil
}

func (i *MariaDBInstall) InitPrimary() error {
	if err := i.Changelocalpassword(); err != nil {
		return err
	}

	if err := i.CreateRepluser(); err != nil {
		return err
	}

	if err := i.CreateBackupuser(); err != nil {
		return err
	}

	return nil
}

func (i *MariaDBInstall) DataSync() error {
	ipPort := strings.Split(i.Option.Join, ":")
	i.Option.Join = ipPort[0]
	masterport, _ := strconv.Atoi(ipPort[1])
	i.Option.Port = masterport

	logger.Infof("开始同步主库 %s:%d 数据到新增的从节点 %s:%d\n", ipPort[0], masterport, i.Option.OwnerIP, i.Option.Port)

	dumpfile := filepath.Join(i.Option.Dir, "/bin/mariadb-dump")
	clientfile := filepath.Join(i.Option.Dir, "/bin/mariadb")

	cmd := fmt.Sprintf("%s -h%s -P%d -u%s -p'%s' --ignore-database=mysql --hex-blob --single-transaction --master-data=2 --gtid  --max_allowed_packet=1G --routines --triggers  -A "+
		" | %s -P%d  -h127.0.0.1   -uroot  -p'%s'  --max_allowed_packet=1G ", dumpfile, ipPort[0], masterport, i.Option.Backupuser, i.Option.BackupPassword, clientfile, i.Option.Port, i.Option.Password)

	l := command.Local{Timeout: 259200}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("执行 mariadb 数据同步失败: %v, 标准错误输出: %s", err, stderr)
	}

	// ipPort := strings.Split(i.Option.Join, ":")
	// masterip := ipPort[0]
	// masterport := ipPort[1]

	return nil
}

func (i *MariaDBInstall) InitSecondary() error {
	if err := i.Changelocalpassword(); err != nil {
		return err
	}

	if i.Option.AddSlave && i.Option.BackupData {
		if err := i.DataSync(); err != nil {
			return err
		}
	}

	if err := i.ChangeMaster(); err != nil {
		return err
	}
	return nil
}

func (i *MariaDBInstall) Info() {
	if !i.Option.Galera {
		logger.Successf("\n")
		logger.Successf("MariaDB初始化[完成]\n")
		logger.Successf("MariaDB端 口:%d\n", i.Option.Port)
		logger.Successf("MariaDB管理用户:root\n")
		logger.Successf("MariaDB管理密码:%s\n", i.Option.Password)
		logger.Successf("MariaDB复制用户:%s\n", i.Option.Repluser)
		logger.Successf("MariaDB复制密码:%s\n", i.Option.ReplPassword)
		logger.Successf("MariaDB备份用户:%s\n", i.Option.Backupuser)
		logger.Successf("MariaDB备份密码:%s\n", i.Option.BackupPassword)
		logger.Successf("启动方式:systemctl start %s\n", fmt.Sprintf(config.ServiceFileName, i.Option.Port))
		logger.Successf("关闭方式:systemctl stop %s\n", fmt.Sprintf(config.ServiceFileName, i.Option.Port))
		logger.Successf("重启方式:systemctl restart %s\n", fmt.Sprintf(config.ServiceFileName, i.Option.Port))
		logger.Successf("登录命令: %s  -uroot -p'%s' --host 127.0.0.1 --port %d\n", filepath.Join(i.Option.Dir, "bin", "mariadb"), i.Option.Password, i.Option.Port)
		if i.Option.Join != "" {
			logger.Successf("\n")
			logger.Successf("请自行检查主从数据同步进度\n")
		}
	} else if i.Option.Galera && i.Option.Onenode {
		logger.Successf("\n")
		logger.Successf("MariaDB Galera 主节点初始化[完成]\n")
		logger.Successf("MariaDB Galera 主节点端口:%d\n", i.Option.Port)
		logger.Successf("MariaDB管理用户:root\n")
		logger.Successf("MariaDB管理密码:%s\n", i.Option.Password)
		logger.Successf("启动方式:systemctl start %s\n", fmt.Sprintf(config.ServiceFileName, i.Option.Port))
		logger.Successf("关闭方式:systemctl stop %s\n", fmt.Sprintf(config.ServiceFileName, i.Option.Port))
		logger.Successf("重启方式:systemctl restart %s\n", fmt.Sprintf(config.ServiceFileName, i.Option.Port))
		logger.Successf("登录命令: %s  -uroot -p'%s' --host 127.0.0.1 --port %d\n", filepath.Join(i.Option.Dir, "bin", "mariadb"), i.Option.Password, i.Option.Port)
	}
}
