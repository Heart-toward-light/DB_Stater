/*
@Author : WuWeiJian
@Date : 2021-04-19 14:40
*/

package services

import (
	"dbup/internal/environment"
	"dbup/internal/global"
	"dbup/internal/pgsql/config"
	"dbup/internal/utils"
	"dbup/internal/utils/command"
	"dbup/internal/utils/logger"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

// 安装pgpool的总控制逻辑
type PgPoolInstall struct {
	parameter              *config.PgPoolParameter
	config                 *config.PgPoolConfig
	pgHba                  *config.PgHba
	service                *config.PgPoolService
	adminUser              string
	adminGroup             string
	adminPassword          string
	port                   int
	basePath               string
	packageFullName        string
	serverBinPath          string
	configFileName         string
	configFileFullName     string
	serviceProcessName     string
	serviceProcessFullName string
	servicePath            string
	serviceFileName        string
	serviceFileFullName    string
	poolHbaFileFullName    string
	pcpFileFullName        string
	version                string
}

func NewPgPoolInstall() *PgPoolInstall {
	hba := config.NewPgHba()
	hba.Init(config.DefaultPGAdminUser)
	return &PgPoolInstall{
		parameter:          &config.PgPoolParameter{},
		config:             config.NewPgPoolConfig(),
		pgHba:              hba,
		serviceProcessName: config.PGPOOLServerProcessName,
		configFileName:     config.PGPOOLConfFileName,
		servicePath:        global.ServicePath,
		version:            config.DefaultPGPOOLVersion,
	}
}

func (i *PgPoolInstall) Run(pre config.PgPoolParameter, cfgFile, packageName string, onlyCheck bool) error {
	if err := i.InitAndCheck(pre, cfgFile, packageName); err != nil {
		return err
	}

	if onlyCheck {
		return nil
	}

	if !i.parameter.Yes {
		var yes string
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
		if !i.parameter.NoRollback {
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

func (i *PgPoolInstall) InitAndCheck(param config.PgPoolParameter, cfgFile, packageName string) error {
	// 初始化参数和配置环节
	if err := i.HandlePrepareArgs(param, cfgFile); err != nil {
		return err
	}
	if err := i.parameter.Validator(); err != nil {
		return err
	}
	if err := i.parameter.CheckEnv(); err != nil {
		return err
	}
	i.HandleArgs(packageName)

	if err := i.config.HandleConfig(i.parameter, i.basePath); err != nil {
		return err
	}
	//i.HandlePgHba()
	if err := i.HandleSystemd(); err != nil {
		return err
	}

	if utils.IsExists(i.serviceFileFullName) {
		return fmt.Errorf("启动文件(%s)已经存在, 停止安装", i.serviceFileFullName)
	}

	return global.CheckPackage(environment.GlobalEnv().ProgramPath, i.packageFullName, config.PGPOOLKinds)
}

// 合并命令行配置
func (i *PgPoolInstall) MergePrepareArgs(param config.PgPoolParameter) {
	logger.Infof("根据命令行参数调整安装配置\n")

	i.parameter.ResourceLimit = param.ResourceLimit
	if param.Port != 0 {
		i.parameter.Port = param.Port
		i.parameter.Dir = fmt.Sprintf("%s%d", config.DefaultPGPoolDir, i.parameter.Port)
	}

	if param.Dir != "" {
		i.parameter.Dir = param.Dir
	}

	if param.PcpPort != 0 {
		i.parameter.PcpPort = param.PcpPort
	}

	if param.WDPort != 0 {
		i.parameter.WDPort = param.WDPort
	}

	if param.HeartPort != 0 {
		i.parameter.HeartPort = param.HeartPort
	}

	if param.BindIP != "" {
		i.parameter.BindIP = param.BindIP
	}

	if param.PcpBindIP != "" {
		i.parameter.PcpBindIP = param.PcpBindIP
	}

	if param.PGPoolIP != "" {
		i.parameter.PGPoolIP = param.PGPoolIP
	}

	if param.Username != "" {
		i.parameter.Username = param.Username
	}

	if param.Password != "" {
		i.parameter.Password = param.Password
	}

	if param.Address != "" {
		i.parameter.Address = param.Address
	}

	i.MergePrepareArgs2(param)
}

// 避免提交时坏味道, 合并参数部分拆分为两个函数
func (i *PgPoolInstall) MergePrepareArgs2(param config.PgPoolParameter) {
	if param.PGMaster != "" {
		i.parameter.PGMaster = param.PGMaster
	}

	if param.PGSlave != "" {
		i.parameter.PGSlave = param.PGSlave
	}

	if param.PGPort != 0 {
		i.parameter.PGPort = param.PGPort
	}

	if param.PGDir != "" {
		i.parameter.PGDir = param.PGDir
	}

	if param.Yes {
		i.parameter.Yes = true
	}

	if param.NoRollback {
		i.parameter.NoRollback = true
	}

	if param.NodeID != 0 {
		i.parameter.NodeID = param.NodeID
	}
}

// 检查命令行配置, 处理安装前准备好的相关参数
func (i *PgPoolInstall) HandlePrepareArgs(param config.PgPoolParameter, cfgFile string) error {
	if cfgFile != "" {
		if utils.IsExists(cfgFile) {
			logger.Infof("从配置文件中获取安装配置: %s\n", cfgFile)
			if err := i.parameter.Load(cfgFile); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("指定的配置文件不存在: %s", cfgFile)
		}
	}

	i.MergePrepareArgs(param)
	i.parameter.InitArgs()

	return nil
}

// 检查命令行配置
func (i *PgPoolInstall) HandleArgs(packageName string) {
	i.adminUser = config.DefaultPGAdminUser
	i.adminGroup = i.adminUser
	//i.adminPassword = i.parameter.AdminPassword

	i.packageFullName = packageName
	if i.packageFullName == "" {
		i.packageFullName = filepath.Join(environment.GlobalEnv().ProgramPath, global.PackagePath, config.PGPOOLKinds, fmt.Sprintf(config.PGPOOLPackageFile, config.DefaultPGPOOLVersion, environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH))
	}

	i.port = i.parameter.Port
	i.basePath = i.parameter.Dir
	//i.serverPath = filepath.Join(i.basePath, config.PGPOOLServerDir)
	i.serverBinPath = filepath.Join(i.basePath, "bin")
	//i.dataPath = filepath.Join(i.basePath, config.PGPOOLDataDir)
	//i.serverFileFullName = filepath.Join(i.serverBinPath, i.serverFileName)
	i.serviceProcessFullName = filepath.Join(i.serverBinPath, i.serviceProcessName)
	i.configFileFullName = filepath.Join(i.basePath, "etc", i.configFileName)
	i.serviceFileName = fmt.Sprintf(config.PGPOOLServiceFileName, i.port)
	i.serviceFileFullName = filepath.Join(i.servicePath, i.serviceFileName)
	i.poolHbaFileFullName = filepath.Join(i.basePath, "etc", config.PGPOOLHbaFileName)
	i.pcpFileFullName = filepath.Join(i.basePath, "etc", config.PGPCPFileName)
}

func (i *PgPoolInstall) HandleSystemd() error {
	var err error
	if i.service, err = config.NewPgPoolService(filepath.Join(environment.GlobalEnv().ProgramPath, global.ServiceTemplatePath, config.PGPoolServiceTemplateFile)); err != nil {
		return err
	}

	env := []string{fmt.Sprintf("LD_LIBRARY_PATH=%s", filepath.Join(i.basePath, "lib"))}
	start := fmt.Sprintf("%s -f %s -a %s -F %s  -C -D", i.serviceProcessFullName, i.configFileFullName, i.poolHbaFileFullName, i.pcpFileFullName)
	stop := fmt.Sprintf("%s -f %s -m fast stop", i.serviceProcessFullName, i.configFileFullName)

	return i.service.HandleConfig(env, i.adminUser, start, stop)
}

// 安装环节(开始在操作系统上生成文件)
func (i *PgPoolInstall) Install() error {
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

	// 检查依赖
	if missLibs, err := global.Checkldd(i.serviceProcessFullName); err != nil {
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

	//if err := i.ChownDir(i.basePath); err != nil {
	//	return err
	//}

	// 生成 pgpool.service, pool_hba.conf, pgpool.conf, pcp.conf 文件
	if err := i.pgHba.SaveTo(i.poolHbaFileFullName); err != nil {
		return err
	}
	if err := i.config.SaveTo(i.configFileFullName); err != nil {
		return err
	}

	//if err := utils.WriteToFile(i.pcpFileFullName, "pgpool:5b8b2a1e5da6f6d1a6de021df03f306b"); err != nil {
	//	return err
	//}
	//

	// 之前的默认密码明文给忘记了, 所以修改如下, 明文件密码是: yyjpcphyqlsnytzjlzzh
	if err := utils.WriteToFile(i.pcpFileFullName, "pgpool:a93e7837ccebecbeecdcc92bee05c763"); err != nil {
		return err
	}

	if err := utils.WriteToFile(filepath.Join(i.basePath, "etc", "pgpool_node_id"), strconv.Itoa(i.parameter.NodeID)); err != nil {
		return err
	}

	if err := i.service.SaveTo(i.serviceFileFullName); err != nil {
		return err
	}

	// 创建业务用户
	if err := i.CreateDBUser(); err != nil {
		return err
	}

	if err := i.ChownDir(i.basePath); err != nil {
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

	if i.parameter.ResourceLimit != "" {
		logger.Infof("设置资源限制\n")
		if err := command.SystemResourceLimit(i.serviceFileName, i.parameter.ResourceLimit); err != nil {
			return err
		}
	}

	if err := command.SystemCtl(i.serviceFileName, "start"); err != nil {
		return err
	}

	return nil
}

func (i *PgPoolInstall) CreateUser() error {
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

func (i *PgPoolInstall) Mkdir() error {
	logger.Infof("创建数据目录和程序目录\n")
	if err := os.MkdirAll(environment.GlobalEnv().DbupInfoPath, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(i.basePath, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(i.servicePath, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(config.DefaultPGPOOLSocketPath, 0755); err != nil {
		return err
	}
	//if err := i.ChownDir(config.DefaultPGPOOLSocketPath); err != nil {  // /tmp 目录不不用改权限
	//	return err
	//}
	//if err := utils.CreateRunDir("postgresql-12.conf", "postgresql", i.adminUser, i.adminGroup); err != nil {
	//	return err
	//}
	return nil
}

func (i *PgPoolInstall) ChownDir(path string) error {
	cmd := fmt.Sprintf("chown -R %s:%s %s", i.adminUser, i.adminGroup, path)
	l := command.Local{}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("修改数据目录所属用户失败: %v, 标准错误输出: %s", err, stderr)
	}
	return nil
}

func (i *PgPoolInstall) Uninstall() {
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

	logger.Warningf("删除安装目录: %s\n", i.basePath)
	if i.basePath != "" && utils.IsDir(i.basePath) {
		if err := os.RemoveAll(i.basePath); err != nil {
			logger.Warningf("删除安装目录失败: %s\n", err)
		} else {
			logger.Warningf("删除安装目录成功\n")
		}
	}
}

func (i *PgPoolInstall) CreatePoolPasswd(username, password string) error {
	if username == "" || password == "" {
		return fmt.Errorf("用户(%s)或密码(%s)不能为空", username, password)
	}

	cmd := fmt.Sprintf("%s -m -u '%s' '%s'  -f %s", filepath.Join(i.serverBinPath, "pg_md5"), username, password, i.configFileFullName)
	l := command.Local{}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("创建访问用户(%s)失败: %v, 标准错误输出: %s", cmd, err, stderr)
	}
	return nil
}

func (i *PgPoolInstall) CreateDBUser() error {
	if err := i.CreatePoolPasswd(i.parameter.Username, i.parameter.Password); err != nil {
		return err
	}
	if err := i.CreatePoolPasswd(config.DefaultPGHideUser, config.DefaultPGHidePass); err != nil {
		return err
	}

	hba := config.NewPgHba()
	if err := hba.Load(i.poolHbaFileFullName); err != nil {
		return err
	}

	address := "localhost,127.0.0.1/32," + i.parameter.Address
	addrs := strings.Split(address, ",")
	for _, addr := range addrs {
		if addr == "localhost" || addr == "local" {
			find := hba.FindRecordByTypeAndUserAndDBAndAddr("local", i.parameter.Username, i.parameter.Username, "")
			if len(find) == 0 {
				hba.AddR("local", i.parameter.Username, i.parameter.Username, "")
			}
		} else {
			var ipm string
			if err := utils.CheckAddressFormat(addr); err == nil {
				ipm = utils.IpAddMaskIfNot(addr)
			} else {
				ipm = addr
			}
			find := hba.FindRecordByTypeAndUserAndDBAndAddr("host", i.parameter.Username, i.parameter.Username, ipm)
			if len(find) == 0 {
				hba.AddRecord(i.parameter.Username, i.parameter.Username, ipm)
			}
		}
	}
	return hba.SaveTo(i.poolHbaFileFullName)
}

func (i *PgPoolInstall) Info() error {
	//TODO: 完成info函数
	logger.Successf("完成pgpool单机版本安装\n")
	return nil
}
