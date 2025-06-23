/*
@Author : WuWeiJian
@Date : 2020-12-03 14:16
*/

package services

import (
	"bufio"
	"dbup/internal/environment"
	"dbup/internal/global"
	"dbup/internal/pgsql/config"
	"dbup/internal/pgsql/dao"
	"dbup/internal/utils"
	"dbup/internal/utils/command"
	"dbup/internal/utils/logger"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gopkg.in/ini.v1"
)

// 安装pgsql的总控制逻辑
type Install struct {
	prepare                *config.Prepare
	config                 *config.PgsqlConfig
	pgHba                  *config.PgHba
	service                *config.PostgresService
	repmgr                 *config.RepmgrConfig
	adminUser              string
	adminGroup             string
	adminPassword          string
	port                   int
	basePath               string
	packageFullName        string
	serverPath             string
	serverLibPath          string
	serverBinPath          string
	serverFileName         string
	serverFileFullName     string
	configFileName         string
	configFileFullName     string
	dataPath               string
	serviceProcessName     string
	serviceProcessFullName string
	servicePath            string
	serviceFileName        string
	serviceFileFullName    string
	version                string
}

func NewInstall() *Install {
	//env, err := environment.NewEnvironment()
	hba := config.NewPgHba()
	return &Install{
		prepare:            &config.Prepare{},
		config:             config.NewPgsqlConfig(),
		pgHba:              hba,
		serverFileName:     config.ServerFileName,
		serviceProcessName: config.ServerProcessName,
		configFileName:     config.ConfFileName,
		servicePath:        global.ServicePath,
		version:            config.DefaultPGVersion,
	}
}

func (i *Install) Run(pre config.Prepare, cfgFile, packageName string, onlyCheck, onlyInstall bool) error {
	if err := i.InitAndCheck(pre, cfgFile, packageName); err != nil {
		return err
	}

	if onlyCheck {
		return nil
	}

	if !i.prepare.Yes {
		var yes string
		logger.Successf("端口: %d\n", i.port)
		logger.Successf("用户: %s\n", i.prepare.Username)
		logger.Successf("数据库名: %s\n", i.prepare.Username)
		logger.Successf("安装路径: %s\n", i.basePath)
		logger.Successf("是否确认安装[y|n]:")
		if _, err := fmt.Scanln(&yes); err != nil {
			return err
		}
		if strings.ToUpper(yes) != "Y" && strings.ToUpper(yes) != "YES" {
			os.Exit(0)
		}
	}

	if err := i.InstallAndInitDB(onlyInstall); err != nil {
		if !i.prepare.NoRollback {
			logger.Warningf("安装失败, 开始回滚\n")
			i.Uninstall()
		}
		return err
	}

	// 整个过程结束，生成连接信息文件, 并返回pguser用户名、密码、授权IP
	if err := i.Info(); err != nil {
		return err
	}

	return nil
}

func (i *Install) RunSlave(pre config.Prepare, masterNode string) error {
	var master string
	var port int
	var err error
	i.prepare = &pre
	i.prepare.InitSlaveArgs()
	if err := i.prepare.ValidatorSlave(); err != nil {
		return err
	}
	if master, port, err = i.ValidatorMasterIP(masterNode); err != nil {
		return err
	}
	if err := i.InitAndCheckSlave(); err != nil {
		return err
	}

	if !i.prepare.Yes {
		var yes string
		logger.Successf("\n")
		logger.Successf("本次安装实例为从节点:\n")
		logger.Successf("要同步数据的主库节点为: %s\n", masterNode)
		logger.Successf("\n")
		logger.Successf("端口: %d\n", i.port)
		logger.Successf("用户: %s\n", i.prepare.Username)
		logger.Successf("数据库名: %s\n", i.prepare.Username)
		logger.Successf("安装路径: %s\n", i.basePath)
		logger.Successf("是否确认安装[y|n]:")
		if _, err := fmt.Scanln(&yes); err != nil {
			return err
		}
		if strings.ToUpper(yes) != "Y" && strings.ToUpper(yes) != "YES" {
			os.Exit(0)
		}
	}

	if err := i.InstallSlave(master, port); err != nil {
		if !i.prepare.NoRollback {
			logger.Warningf("安装失败, 开始回滚\n")
			i.Uninstall()
		}
		return err
	}

	// // 整个过程结束，生成连接信息文件, 并返回pguser用户名、密码、授权IP
	// if err := i.Info(); err != nil {
	// 	return err
	// }

	return nil
}

func (i *Install) InstallSlave(master string, port int) error {
	if err := i.InstallAndInitDB(true); err != nil {
		return err
	}

	if err := i.ReplicaSlave(master, port); err != nil {
		return err
	}

	var cfg config.PgsqlConfig
	if err := global.INILoadFromFile(i.configFileFullName, &cfg, ini.LoadOptions{
		PreserveSurroundedQuote:  true,
		IgnoreInlineComment:      true,
		SpaceBeforeInlineComment: true,
	}); err != nil {
		return err
	}
	cfg.Port = i.port
	cfg.LogDirectory = fmt.Sprintf("'%s/log'", i.dataPath)
	if err := cfg.SaveTo(i.configFileFullName); err != nil {
		return err
	}

	// 设置数据目录和程序目录的所属用户和权限
	if err := i.ChownDir(i.basePath); err != nil {
		return err
	}

	if err := command.SystemCtl(i.serviceFileName, "start"); err != nil {
		return err
	}

	// 检查集群状态; 好像只授权replication库的权限没办法登录,只能不做检测了
	// logger.Infof("等待集群状态:\n")
	// stat := false
	// for n := 1; n <= 20; n++ {
	// 	time.Sleep(3 * time.Second)
	// 	if ok := i.CheckSlaveStatus(master, port); ok {
	// 		stat = true
	// 		break
	// 	}
	// }

	// if !stat {
	// 	return fmt.Errorf("主从状态异常\n")
	// }

	logger.Successf("从库正常\n")
	logger.Successf("集群搭建成功\n")

	return nil
}

func (i *Install) CheckSlaveStatus(master string, port int) bool {
	localIPs, err := utils.LocalIP()
	if err != nil {
		logger.Warningf("获取本机IP地址失败: %s", err)
		return false
	}

	time.Sleep(5 * time.Second)
	var conn *dao.PgConn
	if conn, err = dao.NewPgConn(master, port, i.prepare.Username, i.prepare.Password, "replication"); err != nil {
		logger.Warningf("连接主库失败: %s", err)
		return false
	}
	defer conn.DB.Close()

	repls, err := conn.ReplicationIp()
	if err != nil {
		logger.Warningf("查看从库状态失败: %s", err)
		return false
	}

	isok := false
	for _, repl := range repls {
		for _, localIP := range localIPs {
			if repl == localIP {
				isok = true
				break
			}
		}
		if isok {
			break
		}
	}

	return isok
}

func (i *Install) ReplicaSlave(master string, port int) error {
	cmd := fmt.Sprintf("PGPASSWORD='%s' %s  -D %s -R -Fp -Xs -v  -p %d -h %s -U %s  -P",
		i.prepare.Password,
		filepath.ToSlash(filepath.Join(i.serverBinPath, "pg_basebackup")),
		i.dataPath,
		port,
		master,
		i.prepare.Username)
	l := command.Local{Timeout: 259200}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("同步主库数据失败: %v, 标准错误输出: %s", err, stderr)
	}
	return nil
}

func (i *Install) ValidatorMasterIP(masterIP string) (masterHost string, port int, err error) {
	// 判断
	if masterIP == "" {
		return "", 0, fmt.Errorf("请指定主库IP:PORT")
	}

	ipPort := strings.Split(masterIP, ":")
	// var conn *dao.PgConn

	if err := utils.IsIPv4(ipPort[0]); err != nil {
		if !utils.IsHostName(ipPort[0]) {
			return "", 0, fmt.Errorf("不是可用的IP地址或主机名不可访问")
		}
	}
	if len(ipPort) > 1 {
		port, err = strconv.Atoi(ipPort[1])
		if err != nil {
			return "", 0, fmt.Errorf("masterIP, 不是可用的端口")
		}
		if port <= 0 || port >= 65536 {
			return "", 0, fmt.Errorf("masterIP, 不是可用的端口")
		}
	}

	// 主库连接探测, 查看权限;  好像只授权replication库的权限没办法登录,只能不做检测了
	// if conn, err = dao.NewPgConn(ipPort[0], port, i.prepare.Username, i.prepare.Password, "replication"); err != nil {
	// 	return "", 0, fmt.Errorf("连接主库失败: %v", err)
	// }
	// defer conn.DB.Close()

	// if !conn.IsReplicationGrant(i.prepare.Username) {
	// 	return "", 0, fmt.Errorf("用户: %s 在主库: %s 没有复制权限", i.prepare.Username, ipPort[0])
	// }

	return ipPort[0], port, nil
}

func (i *Install) InstallAndInitDB(onlyInstall bool) error {
	if err := i.Install(); err != nil {
		return err
	}

	if onlyInstall {
		return nil
	}

	if err := i.InitDB(); err != nil {
		return err
	}
	return nil
}

func (i *Install) InitAndCheckSlave() error {
	if err := utils.ValidatorDir(i.prepare.Dir); err != nil {
		return err
	}

	if utils.PortInUse(i.prepare.Port) {
		return fmt.Errorf("端口号被占用: %d", i.prepare.Port)
	}

	i.HandleArgs("")

	// if err := i.config.HandleConfig(i.prepare, filepath.Join(i.dataPath, "log")); err != nil {
	// 	return err
	// }
	//i.HandlePgHba()
	if err := i.HandleSystemd(); err != nil {
		return err
	}

	if utils.IsExists(i.serviceFileFullName) {
		return fmt.Errorf("启动文件(%s)已经存在, 停止安装", i.serviceFileFullName)
	}

	//if err := global.CheckPackage(environment.GlobalEnv().ProgramPath, i.packageFullName, config.Kinds); err != nil {
	//	return err
	//}

	//return global.InstallDbuplib()
	return global.CheckPackage(environment.GlobalEnv().ProgramPath, i.packageFullName, config.Kinds)
}

func (i *Install) InitAndCheck(pre config.Prepare, cfgFile, packageName string) error {
	// 初始化参数和配置环节
	if err := i.HandlePrepareArgs(pre, cfgFile); err != nil {
		return err
	}

	// 如果安装repmgr集群, 需要指定本机IP
	if strings.Contains(i.prepare.Libraries, "repmgr") {
		if err := i.prepare.GetOwner(); err != nil {
			return err
		}
	}

	if err := i.prepare.Validator(); err != nil {
		return err
	}
	if err := i.prepare.CheckEnv(); err != nil {
		return err
	}
	i.HandleArgs(packageName)

	if err := i.config.HandleConfig(i.prepare, filepath.Join(i.dataPath, "log")); err != nil {
		return err
	}
	//i.HandlePgHba()
	if err := i.HandleSystemd(); err != nil {
		return err
	}

	if utils.IsExists(i.serviceFileFullName) {
		return fmt.Errorf("启动文件(%s)已经存在, 停止安装", i.serviceFileFullName)
	}

	if strings.Contains(pre.Libraries, "repmgr") {
		i.HandleRepmgrConfig()
	}

	//if err := global.CheckPackage(environment.GlobalEnv().ProgramPath, i.packageFullName, config.Kinds); err != nil {
	//	return err
	//}

	//return global.InstallDbuplib()
	return global.CheckPackage(environment.GlobalEnv().ProgramPath, i.packageFullName, config.Kinds)
}

// 合并命令行配置
func (i *Install) MergePrepareArgs(pre config.Prepare) {
	logger.Infof("根据命令行参数调整安装配置\n")

	i.prepare.AdminPassword = pre.AdminPassword
	i.prepare.AdminPasswordExpireAt = pre.AdminPasswordExpireAt
	i.prepare.RepmgrOwnerIP = pre.RepmgrOwnerIP
	i.prepare.RepmgrNodeID = pre.RepmgrNodeID
	i.prepare.RepmgrUser = pre.RepmgrUser
	i.prepare.RepmgrPassword = pre.RepmgrPassword
	i.prepare.RepmgrDBName = pre.RepmgrDBName
	i.prepare.ResourceLimit = pre.ResourceLimit

	if pre.SystemUser != "" {
		i.prepare.SystemUser = pre.SystemUser
	}

	if pre.SystemGroup != "" {
		i.prepare.SystemGroup = pre.SystemGroup
	}

	if pre.AdminAddress != "" {
		i.prepare.AdminAddress = pre.AdminAddress
	}

	if pre.Username != "" {
		i.prepare.Username = pre.Username
	}
	if pre.Password != "" {
		i.prepare.Password = pre.Password
	}
	if pre.Port != 0 {
		i.prepare.Port = pre.Port
		i.prepare.Dir = fmt.Sprintf("%s%d", config.DefaultPGDir, i.prepare.Port)
	}
	if pre.Dir != "" {
		i.prepare.Dir = pre.Dir
	}
	if pre.MemorySize != "" {
		i.prepare.MemorySize = pre.MemorySize
	}
	if pre.BindIP != "" {
		i.prepare.BindIP = pre.BindIP
	}
	if pre.Address != "" {
		i.prepare.Address = pre.Address
	}

	if pre.Libraries != "" {
		i.prepare.Libraries = pre.Libraries
	}

	if pre.Yes {
		i.prepare.Yes = true
	}
	if pre.NoRollback {
		i.prepare.NoRollback = true
	}
}

// 检查命令行配置, 处理安装前准备好的相关参数
func (i *Install) HandlePrepareArgs(pre config.Prepare, cfgFile string) error {
	//var err error
	//if cfgFile, err = i.prepare.CfgPath(cfgFile); err != nil {
	//	return err
	//}

	//if (pre.Username == "" || pre.Password == "") && !utils.IsExists(cfgFile) {
	//	return fmt.Errorf("安装配置文件不存在, 并且命令行参数用户和密码为空, 取消安装. 请在命令行指定 -u 用户名 -p 密码 两个参数; 或安装前,先执行 dbup pgsql prepare 生成安装配置文件")
	//}

	//if !utils.IsExists(cfgFile) {
	//	return fmt.Errorf("安装配置文件( %s )不存在,请指定 -c 参数指定配置文件路径, 或先执行 dbup pgsql prepare 生成安装配置文件", cfgFile)
	//}

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

	i.MergePrepareArgs(pre)
	i.prepare.InitArgs()
	return nil
}

// 检查命令行配置
func (i *Install) HandleArgs(packageName string) {
	i.adminUser = i.prepare.SystemUser
	i.adminGroup = i.prepare.SystemGroup
	i.adminPassword = i.prepare.AdminPassword

	i.packageFullName = packageName
	if i.packageFullName == "" {
		i.packageFullName = filepath.Join(environment.GlobalEnv().ProgramPath, global.PackagePath, config.Kinds, fmt.Sprintf(config.PackageFile, config.DefaultPGVersion, environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH))
	}

	i.port = i.prepare.Port
	i.basePath = i.prepare.Dir
	i.serverPath = filepath.Join(i.basePath, config.ServerDir)
	i.serverBinPath = filepath.Join(i.serverPath, "bin")
	i.serverLibPath = filepath.Join(i.serverPath, "lib")
	i.dataPath = filepath.Join(i.basePath, config.DataDir)
	i.serverFileFullName = filepath.Join(i.serverBinPath, i.serverFileName)
	i.serviceProcessFullName = filepath.Join(i.serverBinPath, i.serviceProcessName)
	i.configFileFullName = filepath.Join(i.dataPath, i.configFileName)
	i.serviceFileName = fmt.Sprintf(config.ServiceFileName, i.port)
	i.serviceFileFullName = filepath.Join(i.servicePath, i.serviceFileName)
}

//
//func (i *Install) HandlePgHba() {
//	i.pgHba = append(i.pgHba,
//		&config.PgHbaConfig{
//			Type:     "local",
//			DBName: "all",
//			User:     config.DefaultPGAdminUser,
//			Address:  "",
//			Method:   "md5",
//		},
//		&config.PgHbaConfig{
//			Type:     "local",
//			DBName: "replication",
//			User:     config.DefaultPGAdminUser,
//			Address:  "",
//			Method:   "md5",
//		},
//		&config.PgHbaConfig{
//			Type:     "local",
//			DBName: "all",
//			User:     i.prepare.Username,
//			Address:  "",
//			Method:   "md5",
//		},
//		&config.PgHbaConfig{
//			Type:     "local",
//			DBName: "replication",
//			User:     i.prepare.Username,
//			Address:  "",
//			Method:   "md5",
//		},
//		&config.PgHbaConfig{
//			Type:     "host",
//			DBName: "all",
//			User:     i.prepare.Username,
//			Address:  "127.0.0.1/32",
//			Method:   "md5",
//		},
//		&config.PgHbaConfig{
//			Type:     "host",
//			DBName: "replication",
//			User:     i.prepare.Username,
//			Address:  "127.0.0.1/32",
//			Method:   "md5",
//		})
//
//	addrs := strings.Split(i.prepare.Address, ",")
//	for _, addr := range addrs {
//		if addr == "localhost" {
//			continue
//		}
//
//		var ipmsk string
//		ipMask := strings.Split(addr, "/")
//		switch len(ipMask) {
//		case 1:
//			ipmsk = utils.Ipv4AddMask(addr)
//		case 2:
//			ipmsk = addr
//		default:
//			continue
//		}
//
//		if ipmsk == "127.0.0.1/32" {
//			continue
//		}
//
//		i.pgHba = append(i.pgHba,
//			&config.PgHbaConfig{
//				Type:     "host",
//				DBName: "all",
//				User:     i.prepare.Username,
//				Address:  ipmsk,
//				Method:   "md5",
//			},
//			&config.PgHbaConfig{
//				Type:     "host",
//				DBName: "replication",
//				User:     i.prepare.Username,
//				Address:  ipmsk,
//				Method:   "md5",
//			})
//	}
//}

func (i *Install) HandleSystemd() error {
	var err error
	if i.service, err = config.NewPostgresService(filepath.Join(environment.GlobalEnv().ProgramPath, global.ServiceTemplatePath, config.PostgresServiceTemplateFile)); err != nil {
		return err
	}

	i.service.Description = fmt.Sprintf("PostgreSQL %s database server", i.version)
	i.service.User = i.adminUser
	i.service.Group = i.adminGroup
	i.service.ServiceProcessName = i.serviceProcessFullName
	i.service.DataPath = i.dataPath
	i.service.LibPath = filepath.Join(i.serverPath, "lib")
	i.service.Version = i.version

	return i.service.FormatBody()
}

func (i *Install) HandleRepmgrConfig() {
	i.repmgr = config.NewRepmgrConfig()
	i.repmgr.NodeId = i.prepare.RepmgrNodeID
	i.repmgr.NodeName = fmt.Sprintf("'%s'", i.prepare.RepmgrOwnerIP)
	i.repmgr.Conninfo = fmt.Sprintf("'host=%s port=%d user=%s password=%s dbname=%s connect_timeout=%d'", i.prepare.RepmgrOwnerIP, i.prepare.Port, i.prepare.RepmgrUser, i.prepare.RepmgrPassword, i.prepare.RepmgrDBName, 5)
	i.repmgr.PgBindir = fmt.Sprintf("'%s'", i.serverBinPath)
	i.repmgr.DataDirectory = fmt.Sprintf("'%s'", i.dataPath)
	i.repmgr.LogFile = fmt.Sprintf("'%s'", filepath.Join(i.basePath, "logs", "repmgr.log"))
	i.repmgr.PromoteCommand = fmt.Sprintf("'PGPASSWORD=%s %s standby promote -f %s --log-level NOTICE --verbose --log-to-file'", i.prepare.RepmgrPassword, filepath.Join(i.serverBinPath, "repmgr"), filepath.Join(i.basePath, "repmgr", "repmgr.conf"))
	i.repmgr.FollowCommand = fmt.Sprintf("'PGPASSWORD=%s %s standby follow -f %s -W --log-level DEBUG --verbose --log-to-file --upstream-node-id=%%n'", i.prepare.RepmgrPassword, filepath.Join(i.serverBinPath, "repmgr"), filepath.Join(i.basePath, "repmgr", "repmgr.conf"))
}

func (i *Install) MakeRepmgrConfig(filename string) error {
	logger.Infof("创建repmgr配置文件: %s\n", filename)
	return i.repmgr.SaveTo(filename)
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

//func (i *Install) MakeHbaFile(filename string) error {
//	logger.Infof("创建配置文件: %s\n", filename)
//
//	if utils.IsExists(filename) {
//		if err := command.MoveFile(filename); err != nil {
//			return err
//		}
//	}
//	f, err := os.Create(filename)
//	if err != nil {
//		return err
//	}
//	defer f.Close()
//
//	w := bufio.NewWriter(f)
//	title := fmt.Sprintf(config.HbaFormat, "# TYPE", "DATABASE", "USER", "ADDRESS", "METHOD")
//	if _, err := fmt.Fprintln(w, title); err != nil {
//		return err
//	}
//	for _, v := range i.pgHba {
//		line := fmt.Sprintf(config.HbaFormat, v.Type, v.DBName, v.User, v.Address, v.Method)
//		if _, err := fmt.Fprintln(w, line); err != nil {
//			return err
//		}
//	}
//	return w.Flush()
//}

func (i *Install) MakeSystemdFile(filename string) error {
	logger.Infof("创建启动文件: %s\n", filename)
	return i.service.SaveTo(filename)
}

func (i *Install) CreateUser() error {
	logger.Infof("创建启动用户: %s\n", i.adminUser)
	u, err := user.Lookup(i.adminUser)
	if err == nil { // 如果用户已经存在,则i.adminGroup设置为真正的所属组名
		g, _ := user.LookupGroupId(u.Gid)
		i.adminGroup = g.Name
		return nil
	}
	// groupadd -f <group-name>
	groupAdd := fmt.Sprintf("%s -f %s", command.GroupAddCmd, i.adminGroup)

	// useradd -g <group-name> <user-name>
	userAdd := fmt.Sprintf("%s -g %s %s", command.UserAddCmd, i.adminGroup, i.adminUser)

	l := command.Local{}
	if _, stderr, err := l.Run(groupAdd); err != nil {
		return fmt.Errorf("创建用户组(%s)失败: %v, 标准错误输出: %s", i.adminGroup, err, stderr)
	}
	if _, stderr, err := l.Run(userAdd); err != nil {
		return fmt.Errorf("创建用户(%s)失败: %v, 标准错误输出: %s", i.adminUser, err, stderr)
	}
	return nil
}

func (i *Install) Mkdir() error {
	logger.Infof("创建数据目录和程序目录\n")
	if err := os.MkdirAll(environment.GlobalEnv().DbupInfoPath, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(i.serverPath, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(i.dataPath, 0700); err != nil {
		return err
	}
	//if err := os.MkdirAll(filepath.Join(i.dataPath, "log"), 0755); err != nil {
	//	return err
	//}
	if strings.Contains(i.prepare.Libraries, "repmgr") {
		if err := os.MkdirAll(filepath.Join(i.basePath, "repmgr"), 0700); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Join(i.basePath, "logs"), 0700); err != nil {
			return err
		}
	}

	if err := os.MkdirAll(i.servicePath, 0755); err != nil {
		return err
	}
	// DefaultPGSocketPath = "/var/run/postgresql/" 改为了 DefaultPGSocketPath = "/tmp/" 不需要手动创建了
	// if err := os.MkdirAll(config.DefaultPGSocketPath, 0755); err != nil {
	// 	return err
	// }
	// if err := i.ChownDir(config.DefaultPGSocketPath); err != nil {
	// 	return err
	// }
	// if err := utils.CreateRunDir("postgresql-12.conf", "postgresql", i.adminUser, i.adminGroup); err != nil {
	// 	return err
	// }
	return nil
}

func (i *Install) ChownDir(path string) error {
	cmd := fmt.Sprintf("chown -R %s:%s %s", i.adminUser, i.adminGroup, path)
	l := command.Local{}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("修改数据目录所属用户失败: %v, 标准错误输出: %s", err, stderr)
	}
	return nil
}

func (i *Install) InitDatabase() error {
	logger.Infof("初始化数据库\n")

	pwfile := filepath.Join(i.basePath, config.PasswordFile)
	f, err := os.Create(pwfile)
	if err != nil {
		return fmt.Errorf("创建密码文件失败: %v", err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	if _, err = w.WriteString(i.adminPassword); err != nil {
		return fmt.Errorf("创建密码文件失败: %v", err)
	}
	if err = w.Flush(); err != nil {
		return err
	}

	defer os.Remove(pwfile)

	if err = os.Chmod(pwfile, 0644); err != nil {
		return err
	}

	cmd := fmt.Sprintf("%s -D %s -E UTF8 --locale=en_US.utf8 --pwfile=%s", filepath.Join(i.serverBinPath, config.InitDBCmd), i.dataPath, pwfile)

	l := command.Local{User: i.adminUser}

	if _, stderr, err := l.Sudo(cmd); err != nil {
		return fmt.Errorf("启动pgsql失败: %v, 标准错误输出: %s", err, stderr)
	}
	return nil
}

func (i *Install) Uninstall() {
	if strings.Contains(i.prepare.Libraries, "repmgr") {
		l := command.Local{User: i.adminUser}
		// cmd1 := fmt.Sprintf("%s daemon stop", filepath.Join(i.serverBinPath, "repmgr"))
		cmd1 := fmt.Sprintf("kill $(cat %s)", filepath.Join(i.basePath, "repmgr", "repmgrd.pid"))
		if _, stderr, err := l.Sudo(cmd1); err != nil {
			logger.Warningf("停止repmgr daemon 失败: %s,标准错误输出: %s\n", err, stderr)
		}

		cmd2 := fmt.Sprintf("%s stop -D %s -s -m immediate", i.serverFileFullName, i.dataPath)
		if _, stderr, err := l.Sudo(cmd2); err != nil {
			logger.Warningf("停止pgsql 失败: %s,标准错误输出: %s\n", err, stderr)
		}
	} else {
		logger.Warningf("停止进程, 并删除启动文件: %s\n", i.serviceFileFullName)
		if i.serviceFileFullName != "" && utils.IsExists(i.serviceFileFullName) {
			if err := command.SystemCtl(i.serviceFileName, "stop"); err != nil {
				logger.Warningf("停止pgsql失败: %s\n", err)
			} else {
				logger.Warningf("停止pgsql成功\n")
			}
			if err := os.Remove(i.serviceFileFullName); err != nil {
				logger.Warningf("删除启动文件失败: %s\n", err)
			} else {
				logger.Warningf("删除启动文件成功\n")
			}
		}

		if err := command.SystemdReload(); err != nil {
			logger.Warningf("systemctl daemon-reload 失败\n")
		}
	}

	logger.Warningf("删除安装目录: %s\n", i.basePath)
	if i.basePath != "" && utils.IsDir(i.basePath) {
		if err := os.RemoveAll(i.basePath); err != nil {
			logger.Warningf("删除安装目录失败: %s\n", err)
		} else {
			logger.Warningf("删除安装目录成功\n")
		}
	}
}

//func (i *Run) CreateDB() error {
//	cmd := fmt.Sprintf("PGPASSWORD=%s %s -h /tmp -U %s  -d %s -p %d -c \"CREATE DATABASE %s;\"", i.adminPassword, filepath.Join(i.serverBinPath, config.PsqlCmd), i.adminUser, i.adminUser, i.port, i.prepare.Username)
//	l := command.Local{}
//	if _, stderr, err := l.Run(cmd); err != nil {
//		return fmt.Errorf("创建PG用户失败: %v, 标准错误输出: %s", err, stderr)
//	}
//	return nil
//}
//
//func (i *Run) CreateDBUser() error {
//	cmd := fmt.Sprintf("PGPASSWORD=%s %s -h /tmp -U %s  -d %s -p %d -c \"CREATE USER %s WITH LOGIN CREATEDB password '%s';\"", i.adminPassword, filepath.Join(i.serverBinPath, config.PsqlCmd), i.adminUser, i.adminUser, i.port, i.prepare.Username, i.prepare.Password)
//	l := command.Local{}
//	if _, stderr, err := l.Run(cmd); err != nil {
//		return fmt.Errorf("创建PG用户失败: %v, 标准错误输出: %s", err, stderr)
//	}
//	return nil
//}

// 安装环节(开始在操作系统上生成文件)
func (i *Install) Install() error {
	logger.Infof("开始安装\n")

	//检查并创建 postgres 账号, 设置pgsql的所属用户为postgres
	if err := i.CreateUser(); err != nil {
		return err
	}

	// 创建子目录
	if err := i.Mkdir(); err != nil {
		return err
	}

	// 解压安装包
	logger.Infof("解压安装包: %s 到 %s \n", i.packageFullName, i.basePath)
	if err := utils.UntarGz(i.packageFullName, i.basePath); err != nil {
		return err
	}

	// if strings.Contains(i.prepare.Libraries, "repmgr") {
	// 	if err := i.MakeRepmgrConfig(filepath.Join(i.basePath, "repmgr", "repmgr.conf")); err != nil {
	// 		return err
	// 	}
	// }

	// 检查依赖
	if missLibs, err := global.Checkldd(i.serverFileFullName); err != nil {
		return err
	} else {
		if len(missLibs) != 0 {
			if err := i.LibComplement(missLibs); err != nil {
				return err
			}

			// for _, missLib := range missLibs {
			// 	errInfo = errInfo + fmt.Sprintf("%s, 缺少: %s, 需要: %s\n", missLib.Info, missLib.Name, missLib.Repair)
			// }
			// return errors.New(errInfo)
		}
	}

	if err := i.ChownDir(i.basePath); err != nil {
		return err
	}

	return i.SystemdInit()
}

func (i *Install) LibComplement(NoLiblist []global.MissSoLibrariesfile) error {
	LibList := []string{"libssl.so.10", "libcrypto.so.10", "libtinfo.so.5", "libncurses.so.5"}
	SySLibs := []string{"/lib64", "/lib"}
	for _, missLib := range NoLiblist {
		re := regexp.MustCompile(`\s+`)
		result := re.ReplaceAllString(missLib.Info, "")
		Libname := strings.Split(result, "=")[0]
		for _, s := range LibList {
			if strings.Contains(s, Libname) {
				logger.Warningf("安装出现缺失的Lib文件 %s , 开始进行自动补齐\n", Libname)
				Libfullname := filepath.Join(i.serverLibPath, "newlib", Libname)
				for _, syslibpath := range SySLibs {
					syslibfullname := filepath.Join(syslibpath, Libname)
					if !utils.IsExists(syslibfullname) {
						// if utils.IsExists(Libfullname) {
						if err := command.CopyFileDir(Libfullname, syslibpath); err != nil {
							return err
						}
						// }
						if err := os.Chmod(syslibfullname, 0755); err != nil {
							return err
						}
					}
				}
			}
		}
	}

	if missLibs, err := global.Checkldd(i.serverFileFullName); err != nil {
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

	return nil
}

func (i *Install) SystemdInit() error {
	// 生成 service 启动文件
	// if !strings.Contains(i.prepare.Libraries, "repmgr") {
	if err := i.MakeSystemdFile(i.serviceFileFullName); err != nil {
		return err
	}

	// service reload 并 设置开机自启动, 然后启动PG进程
	if err := command.SystemdReload(); err != nil {
		return err
	}

	logger.Infof("设置开机自启动\n")
	if err := command.SystemCtl(i.serviceFileName, "enable"); err != nil {
		return err
	}

	if i.prepare.ResourceLimit != "" {
		logger.Infof("设置资源限制\n")
		if err := command.SystemResourceLimit(i.serviceFileName, i.prepare.ResourceLimit); err != nil {
			return err
		}
	}
	// }
	return nil
}

func (i *Install) SystemdLaunch() error {
	if !strings.Contains(i.prepare.Libraries, "repmgr") {
		return command.SystemCtl(i.serviceFileName, "start")
	}
	cmd := fmt.Sprintf("%s start -D %s -l %s", i.serviceProcessFullName, i.dataPath, filepath.Join(i.basePath, "logs", "postgres.log"))
	l := command.Local{User: i.adminUser}
	if _, stderr, err := l.Sudo(cmd); err != nil {
		return fmt.Errorf("启动pgsql失败: %v, 标准错误输出: %s", err, stderr)
	}
	return nil
}

func (i *Install) InitDB() error {
	// 初始化数据库
	if err := i.InitDatabase(); err != nil {
		return err
	}

	// 生成 postgresql.conf 数据库的配置文件
	if err := i.MakeConfigFile(i.configFileFullName); err != nil {
		return err
	}

	// 生成 pg_hba.conf 配置文件
	//if err := i.MakeHbaFile(filepath.Join(i.dataPath, config.PgHbaFileName)); err != nil {
	//	return err
	//}
	i.pgHba.Init(i.adminUser)
	if err := i.pgHba.SaveTo(filepath.Join(i.dataPath, config.PgHbaFileName)); err != nil {
		return err
	}

	// 设置数据目录和程序目录的所属用户和权限
	if err := i.ChownDir(i.basePath); err != nil {
		return err
	}

	if err := i.SystemdLaunch(); err != nil {
		return err
	}

	// 在pg中创建数据库 和 用户
	logger.Infof("创建数据库账号\n")
	if err := i.CreateDBUser(); err != nil {
		return err
	}

	return nil
}

//func (i *Install) CreateDBUser() error {
//	conn, err := dao.NewPgConn(config.DefaultPGSocketPath, i.port, i.adminUser, i.adminPassword, i.adminUser)
//	if err != nil {
//		return err
//	}
//	defer conn.DB.Close()
//
//	if err := conn.CreateDB(i.prepare.Username); err != nil {
//		return fmt.Errorf("创建PG库 %s 失败: %v", i.prepare.Username, err)
//	}
//
//	if err := conn.CreateUser(i.prepare.Username, i.prepare.Password, config.DefaultPGUserPriv); err != nil {
//		return fmt.Errorf("创建PG用户 %s 失败: %v", i.prepare.Username, err)
//	}
//
//	if err := conn.CreateUser(config.DefaultPGReplUser, config.DefaultPGReplPass, config.DefaultPGReplPriv); err != nil {
//		return fmt.Errorf("创建PG用户 %s 失败: %v", config.DefaultPGReplUser, err)
//	}
//	return nil
//}

func (i *Install) CreateDBUser() error {
	m := &PGManager{
		Host:      config.DefaultPGSocketPath,
		Port:      i.port,
		AdminUser: i.adminUser,
		// AdminUser:     config.DefaultPGAdminUser,
		AdminPassword: i.adminPassword,
		// AdminDatabase: i.adminUser,
		AdminDatabase: config.DefaultPGAdminUser,
		User:          config.DefaultPGAdminUser,
		Password:      i.prepare.AdminPassword,
		DBName:        "all",
		Role:          "admin",
		Address:       i.prepare.AdminAddress,
	}

	if err := m.InitConn(); err != nil {
		return err
	}
	defer m.Conn.DB.Close()

	if m.Address != "" {
		if err := m.UserGrant(); err != nil {
			return err
		}
	}

	m.User = config.DefaultPGHideUser
	m.Password = config.DefaultPGHidePass
	m.Address = "localhost"

	if err := m.UserCreate(); err != nil {
		return err
	}
	if err := m.UserGrant(); err != nil {
		return err
	}

	// 创建完隐藏用户之后, 给超级管理员用户设置过期时间
	if i.prepare.AdminPasswordExpireAt != "" {
		if err := m.AlterUserExpireAt(config.DefaultPGAdminUser, i.prepare.AdminPasswordExpireAt); err != nil {
			return err
		}
	}

	// 创建普通用户
	m.User = i.prepare.Username
	m.Password = i.prepare.Password
	m.DBName = i.prepare.Username
	// m.Role = "normal"
	m.Role = "dbuser"
	m.Address = "localhost,127.0.0.1/32," + i.prepare.Address

	if err := m.DatabaseCreate(); err != nil {
		return err
	}
	if err := m.UserCreate(); err != nil {
		return err
	}
	return m.UserGrant()
}

func (i *Install) Info() error {
	filename := filepath.Join(environment.GlobalEnv().DbupInfoPath, fmt.Sprintf("%s%d", config.Kinds, i.port))
	info := config.PgsqlInfo{
		Port:      i.port,
		Host:      "127.0.0.1",
		Socket:    config.DefaultPGSocketPath,
		Username:  i.prepare.Username,
		Password:  i.prepare.Password,
		Database:  i.prepare.Username,
		DeployDir: i.serverPath,
		DataDir:   i.dataPath,
	}
	if err := info.SlaveTo(filename); err != nil {
		return err
	}

	//logger.Successf("\n")
	logger.Successf("PG初始化[完成]\n")
	logger.Successf("连接信息保存到: %s\n", filename)
	logger.Successf("PG端 口:%d\n", i.port)
	logger.Successf("PG用 户:%s\n", i.prepare.Username)
	logger.Successf("PG密 码:%s\n", i.prepare.Password)
	logger.Successf("数据库名:%s\n", i.prepare.Username)
	logger.Successf("数据目录:%s\n", i.dataPath)
	logger.Successf("启动用户:%s\n", i.adminUser)
	logger.Successf("启动方式:systemctl start %s\n", i.serviceFileName)
	logger.Successf("关闭方式:systemctl stop %s\n", i.serviceFileName)
	logger.Successf("重启方式:systemctl restart %s\n", i.serviceFileName)
	logger.Successf("登录命令: %s -U %s -p %d\n", filepath.Join(i.serverBinPath, config.PsqlCmd), i.prepare.Username, i.port)
	return nil
}
