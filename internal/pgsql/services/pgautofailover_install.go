package services

import (
	"bufio"
	"dbup/internal/environment"
	"dbup/internal/global"
	"dbup/internal/pgsql/config"
	"dbup/internal/utils"
	"dbup/internal/utils/command"
	"dbup/internal/utils/logger"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type PghaInstall struct {
	monitor                *config.PGAutoFailoverMonitor
	pgnode                 *config.PGAutoFailoverPGNode
	service                *config.AutoPGService
	config                 *config.PgsqlConfig
	pgHba                  *config.PgHba
	host                   string
	port                   int
	adminUser              string
	adminGroup             string
	basePath               string
	dataPath               string
	adminPassword          string
	serverPath             string
	serverBinPath          string
	packageFullName        string
	ResourceLimit          string
	libDir                 string
	libFile                string
	serverFileName         string
	serverAutoFile         string
	serviceProcessName     string
	serviceProcessFullName string
	configFileName         string
	configFileFullName     string
	servicePath            string
	serviceFileName        string
	serviceFileFullName    string
	version                string
}

func NewPghaInstall() *PghaInstall {
	hba := config.NewPgHba()
	return &PghaInstall{
		monitor:            &config.PGAutoFailoverMonitor{},
		pgnode:             &config.PGAutoFailoverPGNode{},
		config:             config.NewPgsqlConfig(),
		pgHba:              hba,
		serverFileName:     config.ServerFileName,
		serverAutoFile:     config.PGAutoFailoverCmd,
		serviceProcessName: config.ServerProcessName,
		configFileName:     config.ConfFileName,
		servicePath:        global.ServicePath,
		version:            config.DefaultPGVersion,
	}
}

func (i *PghaInstall) MonitorRun(mon config.PGAutoFailoverMonitor, onlyCheck bool) error {
	if err := i.MonitorCheck(mon); err != nil {
		return err
	}

	if onlyCheck {
		return nil
	}

	if !i.monitor.Yes {
		var yes string
		logger.Successf("开始安装 PgAutoFailover Monitor\n")
		logger.Successf("端口: %d\n", i.port)
		logger.Successf("安装路径: %s\n", i.monitor.Dir)
		logger.Successf("是否确认安装[y|n]:")
		if _, err := fmt.Scanln(&yes); err != nil {
			return err
		}
		if strings.ToUpper(yes) != "Y" && strings.ToUpper(yes) != "YES" {
			os.Exit(0)
		}
	}

	if err := i.MonitorInstall(); err != nil {
		if !i.monitor.NoRollback {
			logger.Warningf("安装失败, 开始回滚\n")
			if err := i.Uninstall(config.PGMonitor); err != nil {
				return err
			}
		}
		return err
	}

	return nil
}

func (i *PghaInstall) PGdataRun(data config.PGAutoFailoverPGNode, onlyCheck, onlyflushpass bool, cfgFile string) error {
	if onlyflushpass {
		// 配置免密认证
		if err := i.FlushPGAuth(onlyflushpass); err != nil {
			return err
		}
	} else {
		if err := i.PGdataCheck(data, cfgFile); err != nil {
			return err
		}

		if onlyCheck {
			return nil
		}

		if !i.pgnode.Yes {
			var yes string
			logger.Successf("开始安装 PgAutoFailover PGdata\n")
			logger.Successf("端口: %d\n", i.port)
			logger.Successf("安装路径: %s\n", i.pgnode.Dir)
			logger.Successf("是否确认安装[y|n]:")
			if _, err := fmt.Scanln(&yes); err != nil {
				return err
			}
			if strings.ToUpper(yes) != "Y" && strings.ToUpper(yes) != "YES" {
				os.Exit(0)
			}
		}

		if err := i.PGdataInstall(); err != nil {
			if !i.pgnode.NoRollback {
				logger.Warningf("安装失败, 开始回滚\n")
				if err := i.Uninstall(config.PGNode); err != nil {
					return err
				}
			}
			return err
		}

		if err := i.Info(); err != nil {
			return err
		}
	}
	return nil
}

func (i *PghaInstall) PGdataCheck(data config.PGAutoFailoverPGNode, cfgFile string) error {
	// i.pgnode = &data

	// 初始化参数和配置环节
	if err := i.HandlePrepareArgs(data, cfgFile); err != nil {
		return err
	}

	if err := i.pgnode.Validator(); err != nil {
		return err
	}

	// 验证服务
	if err := i.pgnode.Checkservice(); err != nil {
		return err
	}

	if err := i.HandleArgs(config.PGNode); err != nil {
		return err
	}

	if err := utils.ValidatorDir(i.pgnode.Dir); err != nil {
		return err
	}

	if err := i.config.PGdataHandleConfig(i.pgnode, filepath.Join(i.dataPath, "log")); err != nil {
		return err
	}

	return nil
}

func (i *PghaInstall) MonitorCheck(mon config.PGAutoFailoverMonitor) error {
	i.monitor = &mon

	// 验证参数
	if err := i.monitor.Validator(); err != nil {
		return err
	}

	i.HandleArgs(config.PGMonitor)

	if err := utils.ValidatorDir(i.monitor.Dir); err != nil {
		return err
	}

	return nil
}

func (i *PghaInstall) PGdataInstall() error {
	logger.Infof("开始安装 PGdata 节点\n")

	// 检查监控节点服务
	if err := i.pgnode.Checkmonitor(); err != nil {
		return err
	}

	if err := i.SystemInit(); err != nil {
		return err
	}

	// 配置免密认证
	if err := i.FlushPGAuth(false); err != nil {
		return err
	}

	// 初始化数据库
	if err := i.InitDatabase(config.PGNode); err != nil {
		return err
	}

	if i.pgnode.Onenode {
		if err := i.MakeConfigFile(i.configFileFullName); err != nil {
			return err
		}
		if err := i.ChownDir(i.basePath); err != nil {
			return err
		}
		logger.Infof("设置开机启动文件并启动 PG_Auto_failover_PGdata \n")

		if err := i.SystemdInit(); err != nil {
			return err
		}

		time.Sleep(6 * time.Second)
		logger.Infof("创建数据库账号\n")
		if err := i.CreateDBUser(); err != nil {
			return err
		}
	} else {
		// 设置启动文件并启动
		logger.Infof("设置开机启动文件并启动 PG_Auto_failover_PGdata \n")

		if err := i.SystemdInit(); err != nil {
			return err
		}
	}

	if i.pgnode.ResourceLimit != "" {
		logger.Infof("设置资源限制\n")
		if err := command.SystemResourceLimit(i.serviceFileName, i.pgnode.ResourceLimit); err != nil {
			return err
		}
	}

	return nil
}

func (i *PghaInstall) SystemInit() error {
	//检查并创建系统 postgres 账号, 设置pgsql的所属用户为postgres
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

	// 检查系统版本
	if err := i.Checksys(); err != nil {
		return err
	}

	// 检查依赖
	if err := i.Checkldd(); err != nil {
		return err
	}

	if err := i.ChownDir(i.basePath); err != nil {
		return err
	}

	if err := i.ChownTmpDir(i.basePath); err != nil {
		return err
	}

	return nil
}

func (i *PghaInstall) MonitorInstall() error {
	logger.Infof("开始安装 Monitor 节点\n")

	if err := i.SystemInit(); err != nil {
		return err
	}

	// 初始化数据库
	if err := i.InitDatabase(config.PGMonitor); err != nil {
		return err
	}

	// 设置启动文件并启动
	logger.Infof("设置开机启动文件并启动 PG_Auto_failover_Monitor\n")

	if err := i.SystemdInit(); err != nil {
		return err
	}

	time.Sleep(6 * time.Second)
	logger.Infof("通信用户加密\n")
	if err := i.MonitorChangePass(); err != nil {
		return err
	}
	logger.Successf("Monitor 部署完成\n")

	return nil
}

// 检查系统版本
func (i *PghaInstall) Checksys() error {
	v, err := global.OsLoad()
	if err != nil {
		return err
	}
	if v.OSName == "rhel" || v.OSName == "anolis" && global.Osamd() {
		if strings.Split(v.Version, ".")[0] == "8" {
			if err := command.MoveFile(i.serverAutoFile); err != nil {
				return fmt.Errorf("修改 %s %s 系统环境执行文件 %s 失败: %s", v.OSName, v.Version, i.serverAutoFile, err)
			}
			filname := filepath.Join(i.libDir, "redhat8/pg_autoctl")
			logger.Warningf("检测到系统是 %s %s 环境, 开始替换单独编译文件 pg_autoctl \n", v.OSName, v.Version)
			if err := command.CopyFileDir(filname, i.serverBinPath); err != nil {
				return err
			}
		}
	} else if v.OSName == "NFS" && global.Osamd() {
		if err := command.MoveFile(i.serverAutoFile); err != nil {
			return fmt.Errorf("修改 NFS 系统环境执行文件 %s 失败: %s", i.serverAutoFile, err)
		}
		filname := filepath.Join(i.libDir, "NFS/pg_autoctl")
		logger.Warningf("检测到系统是 NFS 环境, 开始替换单独编译文件 pg_autoctl \n")
		if err := command.CopyFileDir(filname, i.serverBinPath); err != nil {
			return err
		}
	}
	return nil
}

// 检查依赖
func (i *PghaInstall) Checkldd() error {

	if missLibs, err := global.Checkldd(i.serverAutoFile); err != nil {
		return err
	} else {
		if len(missLibs) != 0 {
			if err := i.LibComplement(missLibs); err != nil {
				return err
			}
			// 替换完依赖等待 6 秒
			time.Sleep(6 * time.Second)
		}
	}
	return nil
}

func (i *PghaInstall) LibComplement(NoLiblist []global.MissSoLibrariesfile) error {
	LibList := []string{"libssl.so.10", "libcrypto.so.10", "libtinfo.so.5", "libncurses.so.5"}
	SySLibs := []string{"/lib64", "/lib"}
	for _, missLib := range NoLiblist {
		re := regexp.MustCompile(`\s+`)
		result := re.ReplaceAllString(missLib.Info, "")
		Libname := strings.Split(result, "=")[0]
		for _, s := range LibList {
			if strings.Contains(s, Libname) {
				logger.Warningf("安装出现缺失的Lib文件 %s , 开始进行自动补齐\n", Libname)
				Libfullname := filepath.Join(i.libDir, "newlib", Libname)
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

	if missLibs, err := global.Checkldd(i.serverAutoFile); err != nil {
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

func (i *PghaInstall) Mkdir() error {
	logger.Infof("创建数据目录和程序目录\n")

	pg_tmp_dir := fmt.Sprintf("/tmp/pg_autoctl%s", i.basePath)

	if err := os.MkdirAll(environment.GlobalEnv().DbupInfoPath, 0755); err != nil {
		return err
	}

	if err := os.MkdirAll(i.serverPath, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(i.dataPath, 0700); err != nil {
		return err
	}

	if err := os.MkdirAll(i.servicePath, 0755); err != nil {
		return err
	}

	if err := os.MkdirAll(pg_tmp_dir, 0755); err != nil {
		return err
	}

	if err := i.Validationsoftlink(i.basePath); err != nil {
		return err
	}

	return nil
}

// 检查配置根路径是否为软链路径并返回实际路径
func (i *PghaInstall) Validationsoftlink(path string) error {
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return fmt.Errorf("路径 %s 验证异常: %v", path, err)
	}

	if path != realPath {
		pg_tmp_dir := fmt.Sprintf("/tmp/pg_autoctl%s", realPath)
		if err := os.MkdirAll(pg_tmp_dir, 0755); err != nil {
			return fmt.Errorf("物理路径 %s 创建失败: %v", realPath, err)
		}
		if err := i.ChownTmpDir(realPath); err != nil {
			return fmt.Errorf("物理路径 %s 授权失败: %v", realPath, err)
		}
	}

	return nil
}

// 检查命令行配置
func (i *PghaInstall) HandleArgs(role string) error {

	switch role {
	case config.PGMonitor:
		i.port = i.monitor.Port
		i.host = i.monitor.Host
		i.basePath = i.monitor.Dir
		i.adminUser = i.monitor.SystemUser
		i.adminGroup = i.monitor.SystemGroup
		i.adminPassword = i.monitor.AdminPassword
		i.serviceFileName = fmt.Sprintf(config.ServiceMonitorName, i.port)
		// i.serviceFileFullName = filepath.Join(i.servicePath, i.serviceFileName)
	case config.PGNode:
		i.port = i.pgnode.Port
		i.host = i.pgnode.Host
		i.basePath = i.pgnode.Dir
		i.adminUser = i.pgnode.SystemUser
		i.adminGroup = i.pgnode.SystemGroup
		i.adminPassword = i.pgnode.AdminPassword
		i.serviceFileName = fmt.Sprintf(config.ServiceNodeName, i.port)

		// i.serviceFileFullName = filepath.Join(i.servicePath, i.serviceFileName)
	}

	i.serverPath = filepath.Join(i.basePath, config.ServerDir)
	i.serverBinPath = filepath.Join(i.serverPath, "bin")
	i.dataPath = filepath.Join(i.basePath, config.DataDir)
	i.serverAutoFile = filepath.Join(i.serverBinPath, i.serverAutoFile)
	i.serviceProcessFullName = filepath.Join(i.serverBinPath, i.serviceProcessName)
	i.libDir = filepath.Join(i.serverPath, "lib")
	i.libFile = filepath.Join(i.libDir, "libpq.so.5.14")
	i.serviceFileFullName = filepath.Join(i.servicePath, i.serviceFileName)
	i.configFileFullName = filepath.Join(i.dataPath, i.configFileName)

	i.packageFullName = filepath.Join(environment.GlobalEnv().ProgramPath, global.PackagePath, config.Kinds, fmt.Sprintf(config.PackageFile, config.DefaultPGVersion, environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH))

	return global.CheckPackage(environment.GlobalEnv().ProgramPath, i.packageFullName, config.Kinds)
}

func (i *PghaInstall) HandleSystemd() error {
	var err error
	if i.service, err = config.NewAutoPGService(filepath.Join(environment.GlobalEnv().ProgramPath, global.ServiceTemplatePath, config.PGHAServiceTemplateFile)); err != nil {
		return err
	}

	i.service.User = i.adminUser
	i.service.WorkingDirectory = filepath.Join("/home/", i.adminUser)
	i.service.DataPath = i.dataPath
	i.service.ServiceProcessName = i.serverAutoFile

	return i.service.FormatBody()
}

func (i *PghaInstall) CreateUser() error {
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

func (i *PghaInstall) ChownDir(path string) error {
	cmd := fmt.Sprintf("chown -R %s:%s %s", i.adminUser, i.adminGroup, path)
	l := command.Local{}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("修改数据目录所属用户失败: %v, 标准错误输出: %s", err, stderr)
	}
	return nil
}

func (i *PghaInstall) ChownTmpDir(path string) error {
	cmd := fmt.Sprintf("chown -R %s:%s  /tmp/pg_autoctl%s", i.adminUser, i.adminGroup, path)
	l := command.Local{}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("修改数据目录所属用户失败: %v, 标准错误输出: %s", err, stderr)
	}
	return nil
}

func (i *PghaInstall) SystemdInit() error {
	// cmd := fmt.Sprintf("%s show systemd --pgdata %s > %s", i.serverAutoFile, i.dataPath, i.serviceFileFullName)

	// l := command.Local{}
	// if _, stderr, err := l.Sudo(cmd); err != nil {
	// 	return fmt.Errorf("设置启动文件 %s: %v, 标准错误输出: %s", i.serviceFileName, err, stderr)
	// }

	if err := i.MakeSystemdFile(i.serviceFileFullName); err != nil {
		return err
	}

	// service reload 并 设置开机自启动, 然后启动PG进程
	if err := command.SystemdReload(); err != nil {
		return err
	}

	if err := command.SystemCtl(i.serviceFileName, "start"); err != nil {
		return err
	}

	return nil
}

func (i *PghaInstall) MakeSystemdFile(filename string) error {
	logger.Infof("创建启动文件: %s\n", filename)
	if err := i.HandleSystemd(); err != nil {
		return err
	}
	return i.service.SaveTo(filename)
}

func (i *PghaInstall) InitDatabase(role string) error {
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

	l := command.Local{User: i.adminUser}

	var cmd = ""
	switch role {
	case config.PGMonitor:
		cmd = fmt.Sprintf("%s create monitor --pgdata %s --pgctl %s --pgport %d --hostname %s  --auth md5  --ssl-self-signed", i.serverAutoFile, i.dataPath, i.serviceProcessFullName, i.port, i.host)
		if _, stderr, err := l.Sudo(cmd); err != nil {
			return fmt.Errorf("初始化 %s: %v, 标准错误输出: %s", config.PGMonitor, err, stderr)
		}

		if err := i.MonitorChangehba(); err != nil {
			return err
		}

	case config.PGNode:
		if i.pgnode.Onenode {
			initcmd := fmt.Sprintf("%s -D %s -E UTF8 --locale=en_US.utf8 --pwfile=%s", filepath.Join(i.serverBinPath, config.InitDBCmd), i.dataPath, pwfile)
			if _, stderr, err := l.Sudo(initcmd); err != nil {
				return fmt.Errorf("启动pgsql失败: %v, 标准错误输出: %s", err, stderr)
			}

			i.pgHba.Init(i.adminUser)
			if err := i.pgHba.SaveTo(filepath.Join(i.dataPath, config.PgHbaFileName)); err != nil {
				return err
			}

			i.pgHba.Trust_Init(config.PGAutoFailoverUser)
			if err := i.pgHba.SaveTo(filepath.Join(i.dataPath, config.PgHbaFileName)); err != nil {
				return err
			}

			cmd = fmt.Sprintf("PGPASSWORD='%s' %s create postgres --pgdata %s --pgctl %s --pgport %d --hostname %s --dbname %s --username %s --auth md5  --ssl-self-signed", i.adminPassword, i.serverAutoFile, i.dataPath, i.serviceProcessFullName, i.port, i.host, i.adminUser, config.PGAutoFailoverUser)
			cmd = cmd + fmt.Sprintf(" --monitor 'postgres://autoctl_node:%s@%s:%d/pg_auto_failover?sslmode=require' ", config.PGMonitorPasswd, i.pgnode.Mhost, i.pgnode.Mport)
			if _, stderr, err := l.Sudo(cmd); err != nil {
				return fmt.Errorf("初始化 %s: %v, 标准错误输出: %s\n 操作命令: %s", config.PGNode, err, stderr, cmd)
			}
		} else {
			// cmd = fmt.Sprintf("PGPASSWORD='%s' %s create postgres --pgdata %s --pgctl %s --pgport %d --hostname %s --dbname %s --username %s --auth md5  --ssl-self-signed", i.adminPassword, i.serverAutoFile, i.dataPath, i.serviceProcessFullName, i.port, i.host, i.adminUser, config.PGAutoFailoverUser)
			cmd = fmt.Sprintf("%s create postgres --pgdata %s --pgctl %s --pgport %d --hostname %s --dbname %s --username %s --auth md5  --ssl-self-signed", i.serverAutoFile, i.dataPath, i.serviceProcessFullName, i.port, i.host, i.adminUser, config.PGAutoFailoverUser)
			cmd = cmd + fmt.Sprintf(" --monitor 'postgres://autoctl_node:%s@%s:%d/pg_auto_failover?sslmode=require' ", config.PGMonitorPasswd, i.pgnode.Mhost, i.pgnode.Mport)

			if _, stderr, err := l.Sudo(cmd); err != nil {
				return fmt.Errorf("初始化 %s: %v, 标准错误输出: %s", config.PGNode, err, stderr)
			}
		}
	}

	return nil
}

// 创建免密认证文件
func (i *PghaInstall) FlushPGAuth(onlyflush bool) error {
	AdminUserHome := filepath.Join("/home", i.adminUser)
	if !utils.IsDir(AdminUserHome) {
		return fmt.Errorf("系统用户 %s 的家目录 %s 不存在,请验证", i.adminUser, AdminUserHome)
	}
	AuthFile := filepath.Join(AdminUserHome, config.PassHBAFile)
	port := strconv.Itoa(i.port)
	var AuthUserinfo []string
	num := 0
	nodes := strings.Split(i.pgnode.AllNode, ",")
	annotation := "###################### pg_auto_failover " + port + " 配置 ######################"
	failoveruser := "localhost:" + port + ":" + i.adminUser + ":" + config.PGFailoveruser + ":" + config.PGFailoverPasswd
	AuthUserinfo = append(AuthUserinfo, annotation, failoveruser)
	for _, node := range nodes {
		repluser := node + ":replication:" + config.PGRepluser + ":" + config.PGReplicaPasswd
		replpguser := node + ":" + i.adminUser + ":" + config.PGRepluser + ":" + config.PGReplicaPasswd
		replfailuser := node + ":pgautofailover_replicator:" + config.PGRepluser + ":" + config.PGReplicaPasswd
		AuthUserinfo = append(AuthUserinfo, repluser, replpguser, replfailuser)
	}

	if utils.IsExists(AuthFile) {
		// 读取文件内容到内存中
		fileContent, err := ioutil.ReadFile(AuthFile)
		if err != nil {
			return fmt.Errorf("无法读取文件内容：%s", err)
		}

		// 将文件内容转换为字符串
		contentStr := string(fileContent)

		// 遍历列表中的每行数据，检查是否在文件内容中存在并判断写入
		for _, line := range AuthUserinfo {
			if strings.Contains(contentStr, line) {
				num += 1
			}
		}

		if onlyflush || num < 5 {
			logger.Infof("开始刷新 pgpass 认证数据\n")
			if err := command.FlushPGPass(AuthFile, AuthUserinfo); err != nil {
				return fmt.Errorf("刷新 pgpass 失败: %s", err)
			}
		} else {
			logger.Warningf("pgpass 文件内容存在本次认证数据,忽略本次认证配置\n")
		}

	} else {
		if err := command.FlushPGPass(AuthFile, AuthUserinfo); err != nil {
			return fmt.Errorf("刷新 pgpass 失败: %s", err)
		}
	}

	if err := i.ChownDir(AuthFile); err != nil {
		return err
	}

	return nil
}

func (i *PghaInstall) MonitorChangehba() error {
	hbafile := filepath.Join(i.dataPath, config.PgHbaFileName)
	file, err := os.OpenFile(hbafile, os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("hba 文件打开失败 %s", err)
	}

	defer file.Close()

	reader := bufio.NewReader(file)
	for {
		_, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("读取文件失败: %s", err)
		}
	}
	var hostsslhba string
	write := bufio.NewWriter(file)
	hostsslhba = "hostssl   pg_auto_failover    autoctl_node   0.0.0.0/0    md5\n"
	write.WriteString(hostsslhba)
	write.Flush()
	return nil
}

func (i *PghaInstall) MonitorChangePass() error {
	m := &PGManager{
		Host:          config.DefaultPGSocketPath,
		Port:          i.port,
		AdminUser:     i.adminUser,
		AdminPassword: "''",
		AdminDatabase: config.DefaultPGAdminUser,
	}

	if err := m.InitConn(); err != nil {
		return err
	}
	defer m.Conn.DB.Close()

	if err := m.Conn.AlterPassword(config.PGMonitorUser, config.PGMonitorPasswd); err != nil {
		return err
	}

	return m.Conn.ReloadConfig()
	// return nil
}

func (i *PghaInstall) CreateDBUser() error {
	m := &PGManager{
		Host:          config.DefaultPGSocketPath,
		Port:          i.port,
		AdminUser:     i.adminUser,
		AdminPassword: i.adminPassword,
		AdminDatabase: config.DefaultPGAdminUser,
		User:          config.DefaultPGAdminUser,
		Password:      i.pgnode.AdminPassword,
		DBName:        "all",
		Role:          "admin",
		Address:       "",
	}

	if err := m.InitConn(); err != nil {
		return err
	}
	defer m.Conn.DB.Close()

	// 修改pg_autoctl生成的相关用户密码
	if err := m.Conn.AlterPassword(config.PGautomonitoruser, config.PGautomonitorPasswd); err != nil {
		return err
	}

	if err := m.Conn.AlterPassword(config.PGFailoveruser, config.PGFailoverPasswd); err != nil {
		return err
	}

	if err := m.Conn.AlterPassword(config.PGRepluser, config.PGReplicaPasswd); err != nil {
		return err
	}

	if m.Address != "" {
		if err := m.UserGrant(); err != nil {
			return err
		}
	}

	// 创建完隐藏用户之后, 给超级管理员用户设置过期时间
	if i.monitor.AdminPasswordExpireAt != "" {
		if err := m.AlterUserExpireAt(config.DefaultPGAdminUser, i.monitor.AdminPasswordExpireAt); err != nil {
			return err
		}
	}

	// 创建普通用户
	m.User = i.pgnode.Username
	m.Password = i.pgnode.Password
	m.DBName = i.pgnode.Username
	m.Role = "normal"
	m.Address = "localhost,127.0.0.1/32," + i.pgnode.Address

	if err := m.DatabaseCreate(); err != nil {
		return err
	}
	if err := m.UserCreate(); err != nil {
		return err
	}
	return m.AutofailoverGrantuser()

}

func (i *PghaInstall) MakeConfigFile(filename string) error {
	logger.Infof("创建配置文件: %s\n", filename)

	if utils.IsExists(filename) {
		if err := command.MoveFile(filename); err != nil {
			return err
		}
	}
	return i.config.SaveTo(filename)
}

// 检查命令行配置, 处理安装前准备好的相关参数
func (i *PghaInstall) HandlePrepareArgs(pre config.PGAutoFailoverPGNode, cfgFile string) error {
	if cfgFile != "" {
		if utils.IsExists(cfgFile) {
			logger.Infof("从配置文件中获取安装配置: %s\n", cfgFile)
			if err := i.pgnode.Load(cfgFile); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("指定的配置文件不存在: %s", cfgFile)
		}
	}

	i.MergePrepareArgs(pre)
	i.pgnode.InitArgs()
	return nil
}

// 合并命令行配置
func (i *PghaInstall) MergePrepareArgs(pre config.PGAutoFailoverPGNode) {
	logger.Infof("根据命令行参数调整安装配置\n")

	i.pgnode.AdminPassword = pre.AdminPassword
	i.pgnode.AdminPasswordExpireAt = pre.AdminPasswordExpireAt

	i.pgnode.Mhost = pre.Mhost
	i.pgnode.Mport = pre.Mport
	i.pgnode.Host = pre.Host
	i.pgnode.Onenode = pre.Onenode
	i.pgnode.AllNode = pre.AllNode
	i.pgnode.ResourceLimit = pre.ResourceLimit
	// i.pgnode.ResourceLimit = pre.ResourceLimit

	if pre.SystemUser != "" {
		i.pgnode.SystemUser = pre.SystemUser
	}

	if pre.SystemGroup != "" {
		i.pgnode.SystemGroup = pre.SystemGroup
	}

	if pre.AdminAddress != "" {
		i.pgnode.AdminAddress = pre.AdminAddress
	}

	if pre.Username != "" {
		i.pgnode.Username = pre.Username
	}

	if pre.Password != "" {
		i.pgnode.Password = pre.Password
	}

	if pre.Port != 0 {
		i.pgnode.Port = pre.Port
		// i.pgnode.Dir = fmt.Sprintf("%s%d", config.DefaultPGDir, i.pgnode.Port)
	}

	if pre.Dir != "" {
		i.pgnode.Dir = pre.Dir
	}

	if pre.MemorySize != "" {
		i.pgnode.MemorySize = pre.MemorySize
	}
	if pre.BindIP != "" {
		i.pgnode.BindIP = pre.BindIP
	}
	if pre.Address != "" {
		i.pgnode.Address = pre.Address
	}

	if pre.Libraries != "" {
		i.pgnode.Libraries = pre.Libraries
	}

	if pre.Yes {
		i.pgnode.Yes = true
	}

	if pre.NoRollback {
		i.pgnode.NoRollback = true
	}

}

func (i *PghaInstall) Uninstall(role string) error {
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

	if role == config.PGNode {
		logger.Warningf("从集群中删除当前数据节点: %s:%d\n", i.pgnode.Host, i.pgnode.Port)
		l := command.Local{User: i.adminUser}

		cmd := fmt.Sprintf("%s drop node --pgdata %s --destroy", i.serverAutoFile, i.dataPath)

		if _, stderr, err := l.Sudo(cmd); err != nil {
			return fmt.Errorf("初始化 %s: %v, 标准错误输出: %s", config.PGNode, err, stderr)
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

	return nil
}

func (i *PghaInstall) Info() error {
	if i.pgnode.Onenode {
		filename := filepath.Join(environment.GlobalEnv().DbupInfoPath, fmt.Sprintf("%s%d", config.Kinds, i.port))
		info := config.PgsqlInfo{
			Port:      i.port,
			Host:      "127.0.0.1",
			Socket:    config.DefaultPGSocketPath,
			Username:  i.pgnode.Username,
			Password:  i.pgnode.Password,
			Database:  i.pgnode.Username,
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
		logger.Successf("PG用 户:%s\n", i.pgnode.Username)
		logger.Successf("PG密 码:%s\n", i.pgnode.Password)
		logger.Successf("数据库名:%s\n", i.pgnode.Username)
		logger.Successf("数据目录:%s\n", i.dataPath)
		logger.Successf("启动用户:%s\n", i.adminUser)
		logger.Successf("启动方式:systemctl start %s\n", i.serviceFileName)
		logger.Successf("关闭方式:systemctl stop %s\n", i.serviceFileName)
		logger.Successf("重启方式:systemctl restart %s\n", i.serviceFileName)
		logger.Successf("登录命令: %s -U %s -p %d\n", filepath.Join(i.serverBinPath, config.PsqlCmd), i.pgnode.Username, i.port)
	} else {
		logger.Successf("PG从节点初始化[完成]\n")
		logger.Successf("数据目录:%s\n", i.dataPath)
		logger.Successf("启动方式:systemctl start %s\n", i.serviceFileName)
		logger.Successf("关闭方式:systemctl stop %s\n", i.serviceFileName)
		logger.Successf("重启方式:systemctl restart %s\n", i.serviceFileName)
	}
	return nil
}
