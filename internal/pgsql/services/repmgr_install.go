package services

import (
	"dbup/internal/environment"
	"dbup/internal/global"
	"dbup/internal/pgsql/config"
	"dbup/internal/utils"
	"dbup/internal/utils/command"
	"dbup/internal/utils/logger"
	"fmt"
	"math/rand"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"
)

type Repmgr_Install struct {
	repprepare      *config.RepmgrGlobal
	repmgr          *config.RepmgrConfig
	Conn            *command.Connection
	dbport          int
	masterip        string
	masterport      int
	packageFullName string
}

func NewRepmgrInstall(repprepare *config.RepmgrGlobal) *Repmgr_Install {
	return &Repmgr_Install{
		repprepare: repprepare,
	}
}

func (p *Repmgr_Install) PrimaryRun(dbport int, onlyCheck bool) error {

	if err := p.InitAndCheck(dbport); err != nil {
		return err
	}

	if onlyCheck {
		return nil
	}

	if !p.repprepare.Yes {
		var yes string
		logger.Successf("Repmgr 主节点安装端口: %d\n", dbport)
		logger.Successf("Repmgr 主节点安装主路径: %s\n", p.repprepare.Dir)
		logger.Warningf("Repmgr 主节点安装需要重启 postgresql 实例来加载主配置文件\n")
		logger.Successf("是否确认安装[y|n]:")
		if _, err := fmt.Scanln(&yes); err != nil {
			return err
		}
		if strings.ToUpper(yes) != "Y" && strings.ToUpper(yes) != "YES" {
			os.Exit(0)
		}
	}

	if err := p.InstallAndInitDB(); err != nil {
		// if !p.repprepare.NoRollback {
		// 	logger.Warningf("安装失败, 开始回滚\n")
		// 	p.PrimaryUninstall()
		// }
		return err
	}

	if err := p.PrimaryInfo(); err != nil {
		return err
	}

	return nil
}

func (p *Repmgr_Install) StandbyRun(masterip string, masterport int) error {

	if err := p.StandbyParameterCheck(masterip, masterport); err != nil {
		return err
	}

	if !p.repprepare.Yes {
		var yes string
		logger.Successf("Repmgr 从节点默认安装端口: %d\n", p.masterport)
		logger.Successf("Repmgr 从节点安装主路径: %s\n", p.repprepare.Dir)
		logger.Warningf("Repmgr 从节点初始化需要全量拉取主库数据恢复从库, 数据量较大时间会比较长\n")
		logger.Successf("是否确认安装[y|n]:")
		if _, err := fmt.Scanln(&yes); err != nil {
			return err
		}
		if strings.ToUpper(yes) != "Y" && strings.ToUpper(yes) != "YES" {
			os.Exit(0)
		}
	}

	if err := p.InstallStandbyDB(); err != nil {
		if !p.repprepare.NoRollback {
			logger.Warningf("安装失败, 开始回滚\n")
			p.StandbyUninstall()
		}
		return err
	}

	if err := p.StandbyInfo(); err != nil {
		return err
	}

	return nil
}

func (p *Repmgr_Install) StartPrimaryFailNode(pgport int) error {
	if utils.PortInUse(pgport) {
		return fmt.Errorf("postgresql 本地端口 %d 已被占用, 请检查", pgport)
	}

	if !utils.IsDir(p.repprepare.Dir) {
		return fmt.Errorf("postgresql 数据主目录不存在, 请检查: %s", p.repprepare.Dir)
	}

	P := &PGManager{
		Host:          p.repprepare.RepmgrOwnerIP,
		Port:          pgport,
		AdminUser:     p.repprepare.RepmgrUser,
		AdminPassword: p.repprepare.RepmgrPassword,
		AdminDatabase: p.repprepare.RepmgrDBName,
	}

	if err := P.InitConn(); err != nil {
		return err
	}
	defer P.Conn.DB.Close()

	if err := P.CheckSelect(); err != nil {
		return err
	}

	if err := p.Repmgrrejoin(pgport); err != nil {
		return err
	}

	time.Sleep(5 * time.Second)

	if err := p.Repmgrstart(); err != nil {
		return err
	}

	return nil
}

func (p *Repmgr_Install) StartstandbyFailNode(pgport int) error {
	if utils.PortInUse(pgport) {
		return fmt.Errorf("postgresql 本地端口 %d 已被占用, 请检查", pgport)
	}

	if !utils.IsDir(p.repprepare.Dir) {
		return fmt.Errorf("postgresql 数据主目录不存在, 请检查: %s", p.repprepare.Dir)
	}

	if err := p.RepmgrStartPostgreSql(); err != nil {
		logger.Errorf("PostgreSql 服务启动失败: %s\n", err)
	}

	time.Sleep(3 * time.Second)

	if err := p.Repmgrstart(); err != nil {
		return err
	}

	return nil
}

func (p *Repmgr_Install) InstallStandbyDB() error {
	// 创建用户
	if err := p.CreateUser(); err != nil {
		return err
	}

	// 解压安装包
	if err := os.MkdirAll(p.repprepare.Dir, 0700); err != nil {
		return err
	}

	p.packageFullName = filepath.Join(environment.GlobalEnv().ProgramPath, global.PackagePath, config.Kinds, fmt.Sprintf(config.PackageFile, config.DefaultPGVersion, environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH))

	logger.Infof("解压安装包: %s 到 %s \n", p.packageFullName, p.repprepare.Dir)
	if err := utils.UntarGz(p.packageFullName, p.repprepare.Dir); err != nil {
		return err
	}

	// if err := os.MkdirAll(config.DefaultPGSocketPath, 0755); err != nil {
	// 	return err
	// }
	// if err := p.ChownDir(config.DefaultPGSocketPath); err != nil {
	// 	return err
	// }

	if err := p.RepmgrMkdir(); err != nil {
		return err
	}

	// repmgr 初始化
	logger.Infof("开始初始化 repmgr 相关配置\n")
	p.PrimaryRepmgrConfig(config.Repmgrstandbyname)

	if err := p.MakeRepmgrConfig(filepath.Join(p.repprepare.Dir, "repmgr", "repmgr.conf")); err != nil {
		return err
	}

	if err := p.ChownDir(filepath.Join(p.repprepare.Dir, "repmgr")); err != nil {
		return err
	}

	logger.Infof("开始同步主库 %s:%d 数据\n", p.masterip, p.masterport)

	if err := p.RepmgrStandbyClone(); err != nil {
		return err
	}

	logger.Infof("启动从库实例 %s:%d\n", p.repprepare.RepmgrOwnerIP, p.masterport)
	if err := p.RepmgrStartPostgreSql(); err != nil {
		return err
	}

	logger.Infof("开始注册 Standby repmgr \n")
	if err := p.RepmgrStandbyRegister(); err != nil {
		return err
	}

	logger.Infof("5秒后启动 Standby repmgr 守护进程\n")
	time.Sleep(5 * time.Second)

	if err := p.Repmgrstart(); err != nil {
		return err
	}

	return nil
}

func (p *Repmgr_Install) InitAndCheck(dbport int) error {

	logger.Infof("开始验证相关参数\n")

	p.dbport = dbport

	if err := p.PrimaryParameterCheck(); err != nil {
		return err
	}

	dbipport := fmt.Sprintf("%s:%d", p.repprepare.RepmgrOwnerIP, p.dbport)

	ok, _ := utils.TcpGather(dbipport)
	if !ok {
		return fmt.Errorf("postgresql 数据库的ip与端口服务 %s 连接异常", dbipport)
	}

	if _, err := utils.IsEmpty(p.repprepare.Dir); err != nil {
		return fmt.Errorf("postgresql 数据目录主路经下不能为空: %s", p.repprepare.Dir)
	}

	if _, err := utils.IsEmpty(filepath.Join(p.repprepare.Dir, "data")); err != nil {
		return fmt.Errorf("postgresql 数据目录不能为空: %s", filepath.Join(p.repprepare.Dir, "data"))
	}

	return nil
}

func (p *Repmgr_Install) RepmgrMkdir() error {
	logger.Infof("创建 repmgr 所需要的主目录\n")
	if err := os.MkdirAll(filepath.Join(p.repprepare.Dir, "repmgr"), 0700); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(p.repprepare.Dir, "logs"), 0700); err != nil {
		return err
	}

	if err := p.ChownDir(filepath.Join(p.repprepare.Dir, "logs")); err != nil {
		return err
	}

	return nil
}

func (p *Repmgr_Install) InstallAndInitDB() error {

	logger.Infof("开始安装 repmgr 主节点\n")

	if err := p.CreateandChangeRepmgr(); err != nil {
		return err
	}

	servicename := fmt.Sprintf("postgres%d.service", p.dbport)
	servicefile := fmt.Sprintf("%s/%s", global.ServicePath, servicename)

	if err := p.RepmgrMkdir(); err != nil {
		return err
	}

	if err := p.PrimaryPgrestart(servicename, servicefile); err != nil {
		return err
	}

	// repmgr 初始化
	p.PrimaryRepmgrConfig(config.Repmgrprimaryname)

	if err := p.MakeRepmgrConfig(filepath.Join(p.repprepare.Dir, "repmgr", "repmgr.conf")); err != nil {
		return err
	}

	if err := p.ChownDir(filepath.Join(p.repprepare.Dir, "repmgr")); err != nil {
		return err
	}

	logger.Infof("开始注册本地 repmgr 主节点\n")

	if err := p.RepmgrPrimaryRegister(); err != nil {
		return err
	}

	logger.Infof("5秒后启动 repmgr 守护进程\n")
	time.Sleep(5 * time.Second)

	if err := p.Repmgrstart(); err != nil {
		return err
	}

	logger.Infof("移除自启动文件: %s\n", servicename)
	if err := os.Remove(servicefile); err != nil {
		logger.Warningf("移除自启动文件失败: %s\n", err)
	}

	if err := command.SystemdReload(); err != nil {
		logger.Warningf("systemctl daemon-reload 失败\n")
	}

	return nil
}

func (p *Repmgr_Install) CreateandChangeRepmgr() error {
	m := &PGManager{
		Host:          config.DefaultPGSocketPath,
		Port:          p.dbport,
		AdminUser:     p.repprepare.AdminUser,
		AdminPassword: p.repprepare.AdminPassword,
		AdminDatabase: config.DefaultPGAdminUser,

		User:     p.repprepare.RepmgrUser,
		Password: p.repprepare.RepmgrPassword,
		DBName:   p.repprepare.RepmgrDBName,
		Role:     "admin",
		// Address:       p.Priprepare.Host,
	}

	if err := m.InitConn(); err != nil {
		return err
	}
	defer m.Conn.DB.Close()

	// 修改主配置文件
	if err := m.ConfigChangeRepmgr(); err != nil {
		return err
	}

	// 初始换 repmgr db
	if err := m.DatabaseCreate(); err != nil {
		return err
	}

	// 初始化 repmgr 用户相关权限
	m.Address = "local,127.0.0.1," + p.repprepare.RepmgrOwnerIP

	if err := m.UserCreate(); err != nil {
		return err
	}

	if err := m.UserGrant(); err != nil {
		return err
	}

	m.DBName = "replication"
	if err := m.UserGrant(); err != nil {
		return err
	}

	return nil
}

func (p *Repmgr_Install) PrimaryPgrestart(servicename, servicefile string) error {

	if utils.IsExists(servicefile) {
		if err := command.SystemCtl(servicename, "stop"); err != nil {
			return err
		}
		time.Sleep(2 * time.Second)
		if err := p.RepmgrStartPostgreSql(); err != nil {
			logger.Errorf("PostgreSql 服务启动失败: %s\n", err)
		}

	} else {
		if err := p.RepmgrRestartPostgreSql(); err != nil {
			logger.Errorf("PostgreSql 服务重启失败: %s\n", err)
		}
	}

	return nil
}

func (p *Repmgr_Install) PrimaryRepmgrConfig(role string) {

	var dbport int
	switch role {
	case config.Repmgrprimaryname:
		dbport = p.dbport
	case config.Repmgrstandbyname:
		dbport = p.masterport
	}

	p.repmgr = config.NewRepmgrConfig()
	p.repmgr.NodeId = p.repprepare.RepmgrNodeID
	p.repmgr.NodeName = fmt.Sprintf("'%s'", p.repprepare.RepmgrOwnerIP)
	p.repmgr.Conninfo = fmt.Sprintf("'host=%s port=%d user=%s password=%s dbname=%s connect_timeout=%d'", p.repprepare.RepmgrOwnerIP, dbport, p.repprepare.RepmgrUser, p.repprepare.RepmgrPassword, p.repprepare.RepmgrDBName, 5)
	// 判断主路径是否存在
	p.repmgr.PgBindir = fmt.Sprintf("'%s'", filepath.Join(p.repprepare.Dir, "server", "bin"))
	p.repmgr.DataDirectory = fmt.Sprintf("'%s'", filepath.Join(p.repprepare.Dir, "data"))
	p.repmgr.LogFile = fmt.Sprintf("'%s'", filepath.Join(p.repprepare.Dir, "logs", "repmgr.log"))
	p.repmgr.PromoteCommand = fmt.Sprintf("'PGPASSWORD=%s %s standby promote -f %s --log-level NOTICE --verbose --log-to-file'", p.repprepare.RepmgrPassword, filepath.Join(p.repprepare.Dir, "server", "bin", "repmgr"), filepath.Join(p.repprepare.Dir, "repmgr", "repmgr.conf"))
	p.repmgr.FollowCommand = fmt.Sprintf("'PGPASSWORD=%s %s standby follow -f %s -W --log-level DEBUG --verbose --log-to-file --upstream-node-id=%%n'", p.repprepare.RepmgrPassword, filepath.Join(p.repprepare.Dir, "server", "bin", "repmgr"), filepath.Join(p.repprepare.Dir, "repmgr", "repmgr.conf"))
}

func (p *Repmgr_Install) PrimaryValidator() error {
	logger.Infof("验证参数\n")

	return nil
}

func (p *Repmgr_Install) MakeRepmgrConfig(filename string) error {
	logger.Infof("创建repmgr配置文件: %s\n", filename)
	return p.repmgr.SaveTo(filename)
}

func (p *Repmgr_Install) PrimaryParameterCheck() error {

	if p.repprepare.Yes {
		p.repprepare.Yes = true
	}

	rand.Seed(time.Now().Unix())

	if p.repprepare.RepmgrNodeID == 0 {
		p.repprepare.RepmgrNodeID = rand.Intn(10000)
	}

	if p.repprepare.RepmgrDBName == "" {
		p.repprepare.RepmgrDBName = "repmgr"
	}

	if err := utils.IsIPv4(p.repprepare.RepmgrOwnerIP); err != nil {
		return fmt.Errorf("不是可用的IP地址不可访问")
	}

	if p.repprepare.RepmgrPassword == "" {
		p.repprepare.RepmgrPassword = utils.GeneratePasswd(config.DefaultPGPassLength)
	}

	if p.repprepare.RepmgrPassword != "" {
		if err := utils.CheckPasswordLever(p.repprepare.RepmgrPassword); err != nil {
			return fmt.Errorf("指定创建的用户 %s 密码不符合规范，请参考示例: %s ", p.repprepare.RepmgrUser, utils.GeneratePasswd(16))
		}
	}

	if p.repprepare.RepmgrUser == "" {
		p.repprepare.RepmgrUser = config.DefaultPGRepmgrUser
	}

	return nil
}

func (p *Repmgr_Install) StandbyParameterCheck(masterip string, masterport int) error {

	p.masterip = masterip
	p.masterport = masterport
	rand.Seed(time.Now().Unix())

	if p.repprepare.Yes {
		p.repprepare.Yes = true
	}

	if err := utils.IsIPv4(p.repprepare.RepmgrOwnerIP); err != nil {
		return fmt.Errorf("指定的本地ip地址 %s 不可访问，请检查", p.masterip)
	}

	if err := utils.IsIPv4(p.masterip); err != nil {
		return fmt.Errorf("pgsql 主库的 ip 地址 %s 不可访问，请检查", p.masterip)
	}

	dbipport := fmt.Sprintf("%s:%d", p.masterip, p.masterport)

	ok, _ := utils.TcpGather(dbipport)
	if !ok {
		return fmt.Errorf("postgresql 主库服务地址 %s 连接异常", dbipport)
	}

	if p.repprepare.RepmgrNodeID == 0 {
		p.repprepare.RepmgrNodeID = rand.Intn(10000)
	}

	if utils.IsDir(p.repprepare.Dir) {
		return fmt.Errorf("postgresql 数据主目录已存在, 请检查: %s", p.repprepare.Dir)
	}

	if utils.PortInUse(p.masterport) {
		return fmt.Errorf("postgresql 本地端口 %d 已被占用, 请检查", p.masterport)
	}

	if p.repprepare.RepmgrDBName == "" {
		p.repprepare.RepmgrDBName = "repmgr"
	}

	if p.repprepare.RepmgrUser == "" {
		p.repprepare.RepmgrUser = config.DefaultPGRepmgrUser
	}

	return nil
}

func (p *Repmgr_Install) ChownDir(path string) error {
	cmd := fmt.Sprintf("chown -R %s:%s %s", p.repprepare.SystemUser, p.repprepare.SystemGroup, path)
	l := command.Local{}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("修改数据目录所属用户失败: %v, 标准错误输出: %s", err, stderr)
	}
	return nil
}

func (p *Repmgr_Install) RepmgrPrimaryRegister() error {
	// sudo -u postgres /opt/pgsql5432/server/bin/repmgr -f /opt/pgsql5432/repmgr/repmgr.conf primary register
	cmd := fmt.Sprintf("sudo -u %s PGPASSWORD='%s' %s -f %s primary register",
		p.repprepare.SystemUser,
		p.repprepare.RepmgrPassword,
		filepath.Join(p.repprepare.Dir, "server", "bin", "repmgr"),
		filepath.Join(p.repprepare.Dir, "repmgr", "repmgr.conf"))

	l := command.Local{}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", p.repprepare.RepmgrOwnerIP, cmd, err, stderr)
	}

	return nil
}

func (p *Repmgr_Install) RepmgrPrimaryUNregister() error {
	// sudo -u postgres /opt/pgsql5432/server/bin/repmgr -f /opt/pgsql5432/repmgr/repmgr.conf primary register
	cmd := fmt.Sprintf("sudo -u %s PGPASSWORD='%s' %s -f %s primary unregister -F",
		p.repprepare.SystemUser,
		p.repprepare.RepmgrPassword,
		filepath.Join(p.repprepare.Dir, "server", "bin", "repmgr"),
		filepath.Join(p.repprepare.Dir, "repmgr", "repmgr.conf"))

	l := command.Local{}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", p.repprepare.RepmgrOwnerIP, cmd, err, stderr)
	}

	return nil
}

func (p *Repmgr_Install) RepmgrStandbyUNRegister() error {
	// sudo -u postgres /opt/pgsql5432/server/bin/repmgr -f /opt/pgsql5432/repmgr/repmgr.conf standby register
	cmd := fmt.Sprintf("sudo -u %s PGPASSWORD='%s' %s -f %s standby unregister",
		p.repprepare.SystemUser,
		p.repprepare.RepmgrPassword,
		filepath.Join(p.repprepare.Dir, "server", "bin", "repmgr"),
		filepath.Join(p.repprepare.Dir, "repmgr", "repmgr.conf"))

	l := command.Local{}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", p.repprepare.RepmgrOwnerIP, cmd, err, stderr)
	}
	return nil
}

func (p *Repmgr_Install) RepmgrStandbyRegister() error {
	// sudo -u postgres /opt/pgsql5432/server/bin/repmgr -f /opt/pgsql5432/repmgr/repmgr.conf standby register
	cmd := fmt.Sprintf("sudo -u %s PGPASSWORD='%s' %s -f %s standby register",
		p.repprepare.SystemUser,
		p.repprepare.RepmgrPassword,
		filepath.Join(p.repprepare.Dir, "server", "bin", "repmgr"),
		filepath.Join(p.repprepare.Dir, "repmgr", "repmgr.conf"))

	l := command.Local{}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", p.repprepare.RepmgrOwnerIP, cmd, err, stderr)
	}
	return nil
}

func (p *Repmgr_Install) Repmgrstart() error {
	// sudo -u postgres /opt/pgsql5432/server/bin/repmgrd -d -f /opt/pgsql5432/repmgr/repmgr.conf --pid-file /opt/pgsql5432/repmgr/repmgrd.pid
	cmd := fmt.Sprintf("sudo -u %s %s -d -f %s --pid-file %s",
		p.repprepare.SystemUser,
		filepath.Join(p.repprepare.Dir, "server", "bin", "repmgrd"),
		filepath.Join(p.repprepare.Dir, "repmgr", "repmgr.conf"),
		filepath.Join(p.repprepare.Dir, "repmgr", "repmgrd.pid"))

	l := command.Local{}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", p.repprepare.RepmgrOwnerIP, cmd, err, stderr)
	}
	return nil
}

func (p *Repmgr_Install) Repmgrrejoin(pgport int) error {
	cmd := fmt.Sprintf("sudo -u %s PGPASSWORD='%s' %s -f %s node rejoin -d 'host=%s port=%d user=%s password=%s dbname=%s connect_timeout=5' ",
		p.repprepare.SystemUser,
		p.repprepare.RepmgrPassword,
		filepath.Join(p.repprepare.Dir, "server", "bin", "repmgr"),
		filepath.Join(p.repprepare.Dir, "repmgr", "repmgr.conf"),
		p.repprepare.RepmgrOwnerIP,
		pgport,
		p.repprepare.RepmgrUser,
		p.repprepare.RepmgrPassword,
		p.repprepare.RepmgrDBName)

	l := command.Local{}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", p.repprepare.RepmgrOwnerIP, cmd, err, stderr)
	}

	return nil
}

func (p *Repmgr_Install) RepmgrStandbyClone() error {
	//  sudo -u postgres /opt/pgsql5432/server/bin/repmgr -f /opt/pgsql5432/repmgr/repmgr.conf -h 10.249.105.53 -p 5432 -U repmgr -d repmgr standby clone
	cmd := fmt.Sprintf("sudo -u %s PGPASSWORD='%s' %s -f %s -h %s -p %d -U %s -d %s standby clone",
		p.repprepare.SystemUser,
		p.repprepare.RepmgrPassword,
		filepath.Join(p.repprepare.Dir, "server", "bin", "repmgr"),
		filepath.Join(p.repprepare.Dir, "repmgr", "repmgr.conf"),
		p.masterip,
		p.masterport,
		p.repprepare.RepmgrUser,
		p.repprepare.RepmgrDBName)

	l := command.Local{}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", p.repprepare.RepmgrOwnerIP, cmd, err, stderr)
	}
	return nil
}

func (p *Repmgr_Install) RepmgrStartPostgreSql() error {
	// sudo -u postgres  /opt/pgsql5432/server/bin/pg_ctl start -D /opt/pgsql5432/data -l /opt/pgsql5432/logs/pgsql.log
	cmd := fmt.Sprintf("sudo -u %s %s start -D %s -l %s",
		p.repprepare.SystemUser,
		filepath.Join(p.repprepare.Dir, "server", "bin", "pg_ctl"),
		filepath.Join(p.repprepare.Dir, "data"),
		filepath.Join(p.repprepare.Dir, "logs", "postgres.log"))

	l := command.Local{}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", p.repprepare.RepmgrOwnerIP, cmd, err, stderr)
	}

	return nil
}

func (p *Repmgr_Install) RepmgrRestartPostgreSql() error {
	// sudo -u postgres  /opt/pgsql5432/server/bin/pg_ctl start -D /opt/pgsql5432/data -l /opt/pgsql5432/logs/pgsql.log
	cmd := fmt.Sprintf("sudo -u %s %s restart -D %s -l %s",
		p.repprepare.SystemUser,
		filepath.Join(p.repprepare.Dir, "server", "bin", "pg_ctl"),
		filepath.Join(p.repprepare.Dir, "data"),
		filepath.Join(p.repprepare.Dir, "logs", "postgres.log"))

	l := command.Local{}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", p.repprepare.RepmgrOwnerIP, cmd, err, stderr)
	}

	return nil
}

func (p *Repmgr_Install) StandbyUninstall() error {
	l := command.Local{User: p.repprepare.SystemUser}
	cmd1 := fmt.Sprintf("kill $(cat %s)", filepath.Join(p.repprepare.Dir, "repmgr", "repmgrd.pid"))
	if _, stderr, err := l.Sudo(cmd1); err != nil {
		logger.Warningf("停止repmgr daemon 失败: %s,标准错误输出: %s\n", err, stderr)
	}

	cmd2 := fmt.Sprintf("%s stop -D %s -s -m immediate",
		filepath.Join(p.repprepare.Dir, "server", "bin", "pg_ctl"),
		filepath.Join(p.repprepare.Dir, "data"))
	if _, stderr, err := l.Sudo(cmd2); err != nil {
		logger.Warningf("停止pgsql 失败: %s,标准错误输出: %s\n", err, stderr)
	}

	logger.Warningf("删除安装目录: %s\n", p.repprepare.Dir)
	if p.repprepare.Dir != "" && utils.IsDir(p.repprepare.Dir) {
		if err := os.RemoveAll(p.repprepare.Dir); err != nil {
			logger.Warningf("删除安装目录失败: %s\n", err)
		} else {
			logger.Warningf("删除安装目录成功\n")
		}
	}

	return nil
}

func (p *Repmgr_Install) PrimaryUninstall() error {
	repmgrdir := filepath.Join(p.repprepare.Dir, "repmgr")
	repmgrfile := fmt.Sprintf("%s/repmgr.conf", repmgrdir)
	logdir := filepath.Join(p.repprepare.Dir, "logs")

	l := command.Local{User: p.repprepare.SystemUser}
	cmd1 := fmt.Sprintf("kill $(cat %s)", filepath.Join(p.repprepare.Dir, "repmgr", "repmgrd.pid"))
	if _, stderr, err := l.Sudo(cmd1); err != nil {
		logger.Warningf("停止repmgr daemon 失败: %s,标准错误输出: %s\n", err, stderr)
	}

	logger.Warningf("取消 repmgr 注册\n")
	if utils.IsExists(repmgrfile) {
		if err := p.RepmgrPrimaryUNregister(); err != nil {
			logger.Warningf("取消 repmgr 注册失败: %s\n", err)
		}

		if err := os.RemoveAll(repmgrdir); err != nil {
			logger.Warningf("删除创建的 repmgr 目录失败: %s\n", err)
		} else {
			logger.Warningf("删除创建的 repmgr 目录成功\n")
		}
	}

	if utils.IsDir(logdir) {
		if err := os.RemoveAll(repmgrdir); err != nil {
			logger.Warningf("删除创建的 logs 目录失败: %s\n", err)
		} else {
			logger.Warningf("删除创建的 logs 目录成功\n")
		}
	}

	return nil
}

func (p *Repmgr_Install) CreateUser() error {
	logger.Infof("创建启动用户: %s\n", p.repprepare.SystemUser)
	u, err := user.Lookup(p.repprepare.SystemUser)
	if err == nil { // 如果用户已经存在,则i.adminGroup设置为真正的所属组名
		g, _ := user.LookupGroupId(u.Gid)
		p.repprepare.SystemGroup = g.Name
		return nil
	}
	// groupadd -f <group-name>
	groupAdd := fmt.Sprintf("%s -f %s", command.GroupAddCmd, p.repprepare.SystemGroup)

	// useradd -g <group-name> <user-name>
	userAdd := fmt.Sprintf("%s -g %s %s", command.UserAddCmd, p.repprepare.SystemGroup, p.repprepare.SystemUser)

	l := command.Local{}
	if _, stderr, err := l.Run(groupAdd); err != nil {
		return fmt.Errorf("创建用户组(%s)失败: %v, 标准错误输出: %s", p.repprepare.SystemGroup, err, stderr)
	}
	if _, stderr, err := l.Run(userAdd); err != nil {
		return fmt.Errorf("创建用户(%s)失败: %v, 标准错误输出: %s", p.repprepare.SystemUser, err, stderr)
	}
	return nil
}

func (p *Repmgr_Install) PrimaryInfo() error {

	binfile := filepath.Join(p.repprepare.Dir, "server", "bin", "repmgr")
	cnfile := filepath.Join(p.repprepare.Dir, "repmgr", "repmgr.conf")
	//logger.Successf("\n")
	logger.Successf("repmgr 主节点初始化[完成]\n")
	logger.Successf("repmgr 用 户:%s\n", p.repprepare.RepmgrUser)
	logger.Successf("repmgr 密 码:%s\n", p.repprepare.RepmgrPassword)
	logger.Successf("数据库名:%s\n", p.repprepare.RepmgrDBName)
	logger.Successf("配置目录:%s\n", filepath.Join(p.repprepare.Dir, "repmgr"))
	logger.Successf("查询集群管理信息: sudo -u %s %s -f %s cluster show \n", p.repprepare.SystemUser, binfile, cnfile)
	logger.Successf("查询repmgr状态: sudo -u %s %s -f %s service status \n", p.repprepare.SystemUser, binfile, cnfile)
	return nil
}

func (p *Repmgr_Install) StandbyInfo() error {

	binfile := filepath.Join(p.repprepare.Dir, "server", "bin", "repmgr")
	cnfile := filepath.Join(p.repprepare.Dir, "repmgr", "repmgr.conf")

	logger.Successf("repmgr 从节点与从数据库初始化[完成]\n")
	logger.Successf("repmgr 用 户:%s\n", p.repprepare.RepmgrUser)
	logger.Successf("repmgr 密 码:%s\n", p.repprepare.RepmgrPassword)
	logger.Successf("查询集群管理信息: sudo -u %s %s -f %s cluster show \n", p.repprepare.SystemUser, binfile, cnfile)
	logger.Successf("查询repmgr状态: sudo -u %s %s -f %s service status \n", p.repprepare.SystemUser, binfile, cnfile)
	return nil
}
