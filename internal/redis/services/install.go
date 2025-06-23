/*
@Author : WuWeiJian
@Date : 2021-01-06 11:28
*/

package services

import (
	"dbup/internal/environment"
	"dbup/internal/global"
	"dbup/internal/redis/config"
	"dbup/internal/redis/dao"
	"dbup/internal/utils"
	"dbup/internal/utils/arrlib"
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
)

// 安装redis的总控制逻辑
type Install struct {
	parameters          *config.Parameters
	config              *config.RedisConfig
	service             *config.RedisService
	port                int
	SysUser             string
	SysGroup            string
	basePath            string
	packageFullName     string
	serverPath          string
	serverBinPath       string
	serverFileName      string
	serverFileFullName  string
	configFileName      string
	configFileFullName  string
	dataPath            string
	logsPath            string
	servicePath         string
	serviceFileName     string
	serviceFileFullName string
	modules             []string
	version             string
}

func NewInstall() *Install {
	return &Install{
		parameters:     &config.Parameters{},
		config:         config.NewRedisConfig(),
		serverFileName: config.ServerFileName,
		configFileName: config.ConfFileName,
		servicePath:    global.ServicePath,
		version:        config.DefaultRedisVersion,
	}
}

func (i *Install) Run(param config.Parameters, cfgFile string, onlyCheck bool) error {
	// 初始化参数和配置环节
	if err := i.InitAndCheck(param, cfgFile); err != nil {
		return err
	}

	if onlyCheck {
		return nil
	}

	if !i.parameters.Yes {
		var yes string
		if !i.parameters.Cluster && i.parameters.Master != "" {
			logger.Successf("\n")
			logger.Successf("本次安装实例为从节点\n")
			logger.Successf("要加入的集群为: %s\n", i.parameters.Master)
			logger.Successf("\n")
		}

		logger.Successf("端口: %d\n", i.port)
		logger.Successf("安装路径: %s\n", i.basePath)
		logger.Successf("是否确认安装[y|n]:")
		if _, err := fmt.Scanln(&yes); err != nil {
			return err
		}
		if strings.ToUpper(yes) != "Y" && strings.ToUpper(yes) != "YES" {
			os.Exit(0)
		}
	}

	if err := i.Install(); err != nil {
		if !i.parameters.NoRollback {
			logger.Warningf("安装失败, 开始回滚\n")
			i.Uninstall()
		}
		return err
	}

	// 整个过程结束，生成连接信息文件
	return i.Info()
}

func (i *Install) InitAndCheck(param config.Parameters, cfgFile string) error {
	if err := i.HandleParam(param, cfgFile); err != nil {
		return err
	}
	if err := i.parameters.CheckEnv(); err != nil {
		return err
	}
	i.Init()
	if err := i.HandleConfig(); err != nil {
		return err
	}
	if err := i.HandleSystemd(); err != nil {
		return err
	}

	if utils.IsExists(i.serviceFileFullName) {
		return fmt.Errorf("启动文件(%s)已经存在, 停止安装", i.serviceFileFullName)
	}

	return global.CheckPackage(environment.GlobalEnv().ProgramPath, i.packageFullName, config.Kinds)
}

// 检查命令行配置, 处理安装前准备好的相关参数
func (i *Install) HandleParam(param config.Parameters, cfgFile string) error {
	if cfgFile != "" {
		if utils.IsExists(cfgFile) {
			logger.Infof("从配置文件中获取安装配置: %s\n", cfgFile)
			if err := i.parameters.Load(cfgFile); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("指定的配置文件不存在: %s", cfgFile)
		}
	}

	i.MergePrepareArgs(param)
	i.parameters.InitArgs()
	if err := i.parameters.Validator(); err != nil {
		return err
	}
	return nil
}

func (i *Install) CreateUser() error {
	logger.Infof("创建启动用户: %s\n", i.SysUser)
	u, err := user.Lookup(i.SysUser)
	if err == nil { // 如果用户已经存在,则i.adminGroup设置为真正的所属组名
		g, _ := user.LookupGroupId(u.Gid)
		i.SysGroup = g.Name
		return nil
	}
	// groupadd -f <group-name>
	groupAdd := fmt.Sprintf("%s -f %s", command.GroupAddCmd, i.SysGroup)

	// useradd -g <group-name> <user-name>
	userAdd := fmt.Sprintf("%s -g %s %s", command.UserAddCmd, i.SysGroup, i.SysUser)

	l := command.Local{}
	if _, stderr, err := l.Run(groupAdd); err != nil {
		return fmt.Errorf("创建用户组(%s)失败: %v, 标准错误输出: %s", i.SysGroup, err, stderr)
	}
	if _, stderr, err := l.Run(userAdd); err != nil {
		return fmt.Errorf("创建用户(%s)失败: %v, 标准错误输出: %s", i.SysUser, err, stderr)
	}
	return nil
}

// 合并命令行配置
func (i *Install) MergePrepareArgs(param config.Parameters) {
	logger.Infof("根据命令行参数调整安装配置\n")

	i.parameters.Master = param.Master
	i.parameters.ResourceLimit = param.ResourceLimit
	i.parameters.Appendonly = param.Appendonly
	i.parameters.MaxmemoryPolicy = param.MaxmemoryPolicy

	if param.SystemUser != "" {
		i.parameters.SystemUser = param.SystemUser
	}

	if param.SystemUser != "" {
		i.parameters.SystemGroup = param.SystemGroup
	}

	if param.Password != "" {
		i.parameters.Password = param.Password
	}
	if param.Port != 0 {
		i.parameters.Port = param.Port
		i.parameters.Dir = fmt.Sprintf("%s%d", config.DefaultRedisDir, i.parameters.Port)
	}
	if param.Dir != "" {
		i.parameters.Dir = param.Dir
	}
	if param.MemorySize != "" {
		i.parameters.MemorySize = param.MemorySize
	}

	if param.Module != "" {
		i.parameters.Module = param.Module
	}

	if param.Cluster {
		i.parameters.Cluster = true
	}

	if param.Yes {
		i.parameters.Yes = true
	}
	if param.NoRollback {
		i.parameters.NoRollback = true
	}
}

func (i *Install) Init() {
	i.SysUser = i.parameters.SystemUser
	i.SysGroup = i.parameters.SystemGroup
	i.packageFullName = filepath.Join(environment.GlobalEnv().ProgramPath, global.PackagePath, config.Kinds, fmt.Sprintf(config.PackageFile, config.DefaultRedisVersion, environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH))
	i.port = i.parameters.Port
	i.basePath = i.parameters.Dir
	i.serverPath = filepath.Join(i.basePath, config.ServerDir)
	i.serverBinPath = filepath.Join(i.serverPath, "bin")
	i.dataPath = filepath.Join(i.basePath, config.DataDir)
	i.logsPath = filepath.Join(i.basePath, config.LogsDir)
	i.serverFileFullName = filepath.Join(i.serverBinPath, i.serverFileName)
	i.configFileFullName = filepath.Join(i.dataPath, i.configFileName)
	i.serviceFileName = fmt.Sprintf(config.ServiceFileName, i.port)
	i.serviceFileFullName = filepath.Join(i.servicePath, i.serviceFileName)
	if i.parameters.Module != "" {
		i.modules = strings.Split(i.parameters.Module, ",")
	}
}

func (i *Install) HandleConfig() error {
	i.config.PidFile = filepath.Join(i.dataPath, "redis.pid")
	i.config.Port = i.port
	i.config.Socket = fmt.Sprintf("/tmp/.redis%d.sock", i.port)
	i.config.Logfile = filepath.Join(i.logsPath, "redis.log")
	i.config.DbFilename = fmt.Sprintf("redis%d.rdb", i.port)
	i.config.Dir = i.dataPath
	i.config.Appendonly = i.parameters.Appendonly
	i.config.MaxmemoryPolicy = i.parameters.MaxmemoryPolicy

	if i.parameters.Cluster {
		i.config.Cluster = "yes"
	}

	r, _ := regexp.Compile(config.RegexpMemorySuffix)
	index := r.FindStringIndex(i.parameters.MemorySize)
	if index == nil {
		return fmt.Errorf("内存参数必须包含单位后缀(MB 或 GB)")
	}
	memory := i.parameters.MemorySize[:index[0]]
	suffix := strings.ToUpper(i.parameters.MemorySize[index[0]:])
	m, err := strconv.Atoi(memory)
	if err != nil {
		return err
	}

	maxmemoryMB := 0
	switch suffix {
	case "M", "MB":
		maxmemoryMB = m
	case "G", "GB":
		maxmemoryMB = m * 1024
	default:
		return fmt.Errorf("不支持的内存后缀单位")
	}

	i.config.MaxMemory = maxmemoryMB + config.ReplBacklogSizeMB
	i.config.RequirePass = i.parameters.Password
	i.config.MasterAuth = i.parameters.Password
	for _, module := range i.modules {
		i.config.Modules = append(i.config.Modules, filepath.Join(i.serverPath, "so", module+".so"))
	}
	if arrlib.InArray("redisgraph", i.modules) {
		i.config.Appendonly = "no"
		i.config.Save = "save 900 1\nsave 300 10\nsave 60 10000"
	}
	i.config.FormatBody()
	return nil
}

func (i *Install) HandleSystemd() error {
	var err error
	if i.service, err = config.NewRedisService(filepath.Join(environment.GlobalEnv().ProgramPath, global.ServiceTemplatePath, config.RedisServiceTemplateFile)); err != nil {
		return err
	}
	i.service.User = i.SysUser
	i.service.PidFile = filepath.Join(i.dataPath, "redis.pid")
	i.service.ServiceProcessName = i.serverFileFullName
	i.service.ConfigFile = filepath.Join(i.dataPath, "redis.conf")
	i.service.RedisCli = filepath.Join(i.serverBinPath, config.ClientFileName)
	i.service.Port = i.port
	i.service.Password = i.parameters.Password
	i.service.FormatBody()
	return nil
}

// 安装环节(开始在操作系统上生成文件)
func (i *Install) Install() error {
	logger.Infof("开始安装\n")

	//检查并创建 redis 账号
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

	// 检查依赖
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

	// 生成 redis.conf 数据库的配置文件
	if err := i.MakeConfigFile(i.configFileFullName); err != nil {
		return err
	}

	if err := i.ChownDir(i.parameters.Dir); err != nil {
		return err
	}

	// 生成 service 启动文件
	if err := i.MakeSystemdFile(i.serviceFileFullName); err != nil {
		return err
	}

	// service reload 并 设置开机自启动, 然后启动PG进程
	if err := command.SystemdReload(); err != nil {
		return err
	}

	logger.Infof("设置开机自启动, 并启动实例\n")
	if err := command.SystemCtl(i.serviceFileName, "enable"); err != nil {
		return err
	}

	if i.parameters.ResourceLimit != "" {
		logger.Infof("设置资源限制\n")
		if err := command.SystemResourceLimit(i.serviceFileName, i.parameters.ResourceLimit); err != nil {
			return err
		}
	}

	if err := command.SystemCtl(i.serviceFileName, "start"); err != nil {
		return err
	}

	if !i.parameters.Cluster && i.parameters.Master != "" {
		if err := i.Replication(); err != nil {
			return err
		}
	}

	return nil
}

func (i *Install) Replication() error {
	var master string
	var port int

	ipPort := strings.Split(i.parameters.Master, ":")
	master = ipPort[0]
	if len(ipPort) > 1 {
		port, _ = strconv.Atoi(ipPort[1])
	} else {
		port = i.port
	}

	logger.Infof("5秒后建立主从关系\n")
	time.Sleep(5 * time.Second)
	conn, err := dao.NewRedisConn("127.0.0.1", i.port, i.parameters.Password)
	if err != nil {
		return err
	}
	defer conn.Conn.Close()

	if err := conn.SlaveOf(master, port); err != nil {
		return err
	}

	logger.Infof("5秒后检查主从状态\n")
	time.Sleep(5 * time.Second)
	localIPs, err := utils.LocalIP()
	if err != nil {
		return err
	}

	conn1, err := dao.NewRedisConn(master, port, i.parameters.Password)
	if err != nil {
		return err
	}
	defer conn.Conn.Close()

	ips := conn1.SlaveIPs()

	for _, ip := range ips {
		for _, localIP := range localIPs {
			if localIP+":"+strconv.Itoa(i.port) == ip {
				logger.Infof("主从同步正常, 请自行观察数据同步是否完成\n")
				return nil
			}
		}

	}
	return fmt.Errorf("主从状态异常")
}

func (i *Install) Uninstall() {
	logger.Warningf("停止进程, 并删除启动文件: %s\n", i.serviceFileFullName)
	if i.serviceFileFullName != "" && utils.IsExists(i.serviceFileFullName) {
		if err := command.SystemCtl(i.serviceFileName, "stop"); err != nil {
			logger.Warningf("停止redis失败: %s\n", err)
		} else {
			logger.Warningf("停止redis成功\n")
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

	logger.Warningf("删除安装目录: %s\n", i.basePath)
	if i.basePath != "" && utils.IsDir(i.basePath) {
		if err := os.RemoveAll(i.basePath); err != nil {
			logger.Warningf("删除安装目录失败: %s\n", err)
		} else {
			logger.Warningf("删除安装目录成功\n")
		}
	}
}

func (i *Install) ChownDir(path string) error {
	cmd := fmt.Sprintf("chown -R %s:%s %s", i.SysUser, i.SysGroup, path)
	l := command.Local{}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("修改数据目录所属用户失败: %v, 标准错误输出: %s", err, stderr)
	}
	return nil
}

func (i *Install) Info() error {
	filename := filepath.Join(environment.GlobalEnv().DbupInfoPath, fmt.Sprintf("%s%d", config.Kinds, i.port))
	info := config.PgsqlInfo{
		Port:      i.port,
		Host:      "127.0.0.1",
		Socket:    i.config.Socket,
		Password:  i.parameters.Password,
		DeployDir: i.serverPath,
		DataDir:   i.dataPath,
	}
	if err := info.SlaveTo(filename); err != nil {
		return err
	}

	//logger.Successf("\n")
	logger.Successf("Redis 初始化[完成]\n")
	logger.Successf("连接信息保存到: %s\n", filename)
	logger.Successf("Redis 端 口:%d\n", i.port)
	logger.Successf("Redis 密 码:%s\n", i.parameters.Password)
	logger.Successf("数据目录:%s\n", i.dataPath)
	logger.Successf("启动方式:systemctl start %s\n", i.serviceFileName)
	logger.Successf("关闭方式:systemctl stop %s\n", i.serviceFileName)
	logger.Successf("重启方式:systemctl restart %s\n", i.serviceFileName)
	logger.Successf("登录命令: %s -h %s -p %d -a %s\n", filepath.Join(i.serverBinPath, config.ClientFileName), "127.0.0.1", i.port, "<password>")
	if i.parameters.Master != "" {
		logger.Successf("\n")
		logger.Successf("请自行检查主从数据同步进度\n")
	}
	return nil
}

func (i *Install) Mkdir() error {
	logger.Infof("创建数据目录和程序目录\n")
	if err := os.MkdirAll(environment.GlobalEnv().DbupInfoPath, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(i.dataPath, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(i.logsPath, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(i.serverPath, 0755); err != nil {
		return err
	}
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

func (i *Install) MakeSystemdFile(filename string) error {
	logger.Infof("创建启动文件: %s\n", filename)
	return i.service.SaveTo(filename)
}
