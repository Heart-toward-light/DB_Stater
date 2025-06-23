package services

import (
	"dbup/internal/environment"
	"dbup/internal/pgsql/config"
	"dbup/internal/utils"
	"dbup/internal/utils/logger"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type PGAutoFailoverDeploy struct {
	Param      config.PGAutoFailoverParameter
	Monitor    *AutoInstance
	PGmaster   *AutoInstance
	PGslave    []*AutoInstance
	NewPGslave []*AutoInstance
	NewPGdata  bool
}

func NewPGAutoFailoverDeploy() *PGAutoFailoverDeploy {
	return &PGAutoFailoverDeploy{}
}

func (d *PGAutoFailoverDeploy) Run(c string, n, y bool) error {

	// 初始化参数和配置环节
	if err := d.Param.Load(c); err != nil {
		return err
	}

	d.InitParameter()

	d.Param.Server.SetDefault()

	if d.NewPGdata {
		if !y {
			logger.Warningf("配置检测到有新增从节点 %s:%d 需要加入到集群中\n", d.Param.Server.NewPGnode, d.Param.Pgnode.Port)
			logger.Warningf("请确认[y|n]:")

			var yes string
			if _, err := fmt.Scanln(&yes); err != nil {
				return err
			}
			if strings.ToUpper(yes) != "Y" && strings.ToUpper(yes) != "YES" {
				os.Exit(0)
			}
		}

		if err := d.Param.NewPGCheck(); err != nil {
			return err
		}
	}

	if err := d.Param.Validator(); err != nil {
		return err
	}

	logger.Infof("初始化部署对象\n")
	if d.Param.Server.SshPassword != "" {
		if err := d.Init(); err != nil {
			return err
		}
	} else {
		if err := d.InitUseKeyFile(); err != nil {
			return err
		}
	}

	if err := d.CheckTmpDir(); err != nil {
		return err
	}
	defer d.DropTmpDir()

	if err := d.Scp(); err != nil {
		return err
	}

	if err := d.CheckEnv(); err != nil {
		return err
	}

	if err := d.InstallAndInitSlave(); err != nil {
		if !n {
			logger.Warningf("安装失败, 开始回滚\n")
			if d.NewPGdata {
				d.UNInstallNewPGdata()
			} else {
				d.UNInstall()
			}
		}
		return err
	}

	return nil
}

func (d *PGAutoFailoverDeploy) InitParameter() {
	d.Param.Pgmonitor.Host = d.Param.Server.Monitor
	d.Param.Pgnode.Mhost = d.Param.Server.Monitor
	d.Param.Pgnode.Mport = d.Param.Pgmonitor.Port

	if d.Param.Pgnode.Username == "" {
		d.Param.Pgnode.Username = config.DefaultPGUser
	}

	if d.Param.Pgmonitor.SystemUser == "" {
		d.Param.Pgmonitor.SystemUser = config.DefaultPGAdminUser
	}

	if d.Param.Pgnode.SystemUser == "" {
		d.Param.Pgnode.SystemUser = config.DefaultPGAdminUser
	}

	if d.Param.Pgnode.Password == "" {
		d.Param.Pgnode.Password = utils.GeneratePasswd(config.DefaultPGPassLength)
	}

	if d.Param.Server.NewPGnode != "" {
		d.NewPGdata = true
	} else {
		d.NewPGdata = false
	}

	// 生成所有数据节点信息进行弱密码配置
	allnodes := ""
	pgnodes := strings.Split(d.Param.Server.PGNode, ",")
	for _, node := range pgnodes {
		allnodes += fmt.Sprintf("%s:%d,", node, d.Param.Pgnode.Port)
	}
	if d.NewPGdata {
		newpgnodes := strings.Split(d.Param.Server.NewPGnode, ",")
		for _, newnode := range newpgnodes {
			allnodes += fmt.Sprintf("%s:%d,", newnode, d.Param.Pgnode.Port)
		}
	}
	d.Param.Pgnode.AllNode = allnodes[:len(allnodes)-1]

}

func (d *PGAutoFailoverDeploy) RemoveDeploy(c string, yes bool) error {
	// 初始化参数和配置环节
	if err := d.Param.Load(c); err != nil {
		return err
	}

	d.InitParameter()

	d.Param.Server.SetDefault()

	if err := d.Param.Validator(); err != nil {
		return err
	}

	if d.Param.Pgmonitor.Port == 0 {
		return fmt.Errorf("请指定 Monitor 端口号")
	}

	if d.Param.Pgmonitor.Dir == "" {
		return fmt.Errorf("请指定 Monitor 主路径")
	}

	if d.Param.Pgnode.Port == 0 {
		return fmt.Errorf("请指定 PGdata 端口号")
	}

	if d.Param.Pgnode.Dir == "" {
		return fmt.Errorf("请指定 PGdata 主路径")
	}

	logger.Warningf("准备删除 pg_auto_failover 集群\n")
	logger.Warningf("要删除的集群 Monitor 节点以及数据目录: %s:%d %s\n", d.Param.Pgmonitor.Host, d.Param.Pgmonitor.Port, d.Param.Pgmonitor.Dir)
	for _, ip := range strings.Split(d.Param.Server.PGNode, ",") {
		logger.Warningf("要删除的集群 PGdata 节点以及数据目录: %s:%d %s\n", ip, d.Param.Pgnode.Port, d.Param.Pgnode.Dir)
	}
	if d.NewPGdata {
		for _, ip := range strings.Split(d.Param.Server.NewPGnode, ",") {
			logger.Warningf("要删除的集群 PGdata 节点以及数据目录: %s:%d %s\n", ip, d.Param.Pgnode.Port, d.Param.Pgnode.Dir)
		}
	}

	if !yes {
		logger.Warningf("删除集群是危险操作,会将整个集群中的数据完全删除, 不可恢复\n")
		logger.Warningf("是否确认删除[y|n]:")

		var yes string
		if _, err := fmt.Scanln(&yes); err != nil {
			return err
		}
		if strings.ToUpper(yes) != "Y" && strings.ToUpper(yes) != "YES" {
			os.Exit(0)
		}
	}

	logger.Infof("初始化删除对象\n")

	if d.Param.Server.SshPassword != "" {
		if err := d.Init(); err != nil {
			return err
		}
	} else {
		if err := d.InitUseKeyFile(); err != nil {
			return err
		}
	}

	if err := d.CheckTmpDir(); err != nil {
		return err
	}
	defer d.DropTmpDir()

	if err := d.Scp(); err != nil {
		return err
	}

	if d.NewPGdata {
		d.UNInstallNewPGdata()
	}
	d.UNInstall()

	return nil
}

func (d *PGAutoFailoverDeploy) Init() error {
	var err error
	if !d.NewPGdata {
		if d.Monitor, err = NewMonitorInstance(d.Param.Server.TmpDir,
			d.Param.Server.Monitor,
			d.Param.Server.SshUser,
			d.Param.Server.SshPassword,
			d.Param.Server.SshPort,
			d.Param.Pgmonitor,
			0); err != nil {
			return err
		}
	} else {
		newpgnodes := strings.Split(d.Param.Server.NewPGnode, ",")
		for _, newpgnode := range newpgnodes[1:] {
			s, err := NewPGdataInstance(d.Param.Server.TmpDir,
				newpgnode,
				d.Param.Server.SshUser,
				d.Param.Server.SshPassword,
				d.Param.Server.SshPort,
				d.Param.Pgnode,
				0)
			if err != nil {
				return err
			}
			d.NewPGslave = append(d.NewPGslave, s)
		}
	}

	pgnodes := strings.Split(d.Param.Server.PGNode, ",")
	if d.PGmaster, err = NewPGdataInstance(d.Param.Server.TmpDir,
		pgnodes[0],
		d.Param.Server.SshUser,
		d.Param.Server.SshPassword,
		d.Param.Server.SshPort,
		d.Param.Pgnode,
		0); err != nil {
		return err
	}
	for _, pgnode := range pgnodes[1:] {
		s, err := NewPGdataInstance(d.Param.Server.TmpDir,
			pgnode,
			d.Param.Server.SshUser,
			d.Param.Server.SshPassword,
			d.Param.Server.SshPort,
			d.Param.Pgnode,
			0)
		if err != nil {
			return err
		}
		d.PGslave = append(d.PGslave, s)
	}

	return nil
}

func (d *PGAutoFailoverDeploy) InitUseKeyFile() error {
	var err error
	if !d.NewPGdata {
		if d.Monitor, err = NewMonitorInstanceUseKeyFile(d.Param.Server.TmpDir,
			d.Param.Server.Monitor,
			d.Param.Server.SshUser,
			d.Param.Server.SshKeyFile,
			d.Param.Server.SshPort,
			d.Param.Pgmonitor,
			0); err != nil {
			return err
		}
	} else {
		newpgnodes := strings.Split(d.Param.Server.NewPGnode, ",")
		for _, newpgnode := range newpgnodes[1:] {
			s, err := NewPGdataInstance(d.Param.Server.TmpDir,
				newpgnode,
				d.Param.Server.SshUser,
				d.Param.Server.SshKeyFile,
				d.Param.Server.SshPort,
				d.Param.Pgnode,
				0)
			if err != nil {
				return err
			}
			d.NewPGslave = append(d.NewPGslave, s)
		}
	}

	pgnodes := strings.Split(d.Param.Server.PGNode, ",")
	if d.PGmaster, err = NewPGdataInstanceUseKeyFile(d.Param.Server.TmpDir,
		pgnodes[0],
		d.Param.Server.SshUser,
		d.Param.Server.SshKeyFile,
		d.Param.Server.SshPort,
		d.Param.Pgnode,
		0); err != nil {
		return err
	}
	for _, pgnode := range pgnodes[1:] {
		s, err := NewPGdataInstanceUseKeyFile(d.Param.Server.TmpDir,
			pgnode,
			d.Param.Server.SshUser,
			d.Param.Server.SshKeyFile,
			d.Param.Server.SshPort,
			d.Param.Pgnode,
			0)
		if err != nil {
			return err
		}
		d.PGslave = append(d.PGslave, s)
	}

	return nil
}

func (d *PGAutoFailoverDeploy) CheckTmpDir() error {
	logger.Infof("检查目标机器的临时目录\n")
	if !d.NewPGdata {
		if err := d.Monitor.CheckTmpDir(); err != nil {
			return err
		}
	} else {
		for _, newslave := range d.NewPGslave {
			if err := newslave.CheckTmpDir(); err != nil {
				return err
			}
		}
	}

	if err := d.PGmaster.CheckTmpDir(); err != nil {
		return err
	}

	for _, slave := range d.PGslave {
		if err := slave.CheckTmpDir(); err != nil {
			return err
		}
	}
	return nil
}

func (d *PGAutoFailoverDeploy) DropTmpDir() {
	logger.Infof("删除目标机器的临时目录\n")
	if !d.NewPGdata {
		_ = d.Monitor.DropTmpDir()
	} else {
		for _, newpgnode := range d.NewPGslave {
			_ = newpgnode.DropTmpDir()
		}
	}

	for _, pgnode := range d.PGslave {
		_ = pgnode.DropTmpDir()
	}

	_ = d.PGmaster.DropTmpDir()
}

func (d *PGAutoFailoverDeploy) Scp() error {
	logger.Infof("将所需文件复制到目标机器\n")
	source := path.Join(environment.GlobalEnv().ProgramPath, "..")

	if !d.NewPGdata {
		logger.Infof("复制到: %s\n", d.Monitor.Host)
		if err := d.Monitor.Scp(source); err != nil {
			return err
		}
	} else {
		for _, newpgnode := range d.NewPGslave {
			logger.Infof("复制到新增从节点: %s\n", newpgnode.Host)
			if err := newpgnode.Scp(source); err != nil {
				return err
			}
		}
	}

	logger.Infof("复制到: %s\n", d.PGmaster.Host)
	if err := d.PGmaster.Scp(source); err != nil {
		return err
	}

	for _, pgnode := range d.PGslave {
		logger.Infof("复制到: %s\n", pgnode.Host)
		if err := pgnode.Scp(source); err != nil {
			return err
		}
	}

	return nil
}

func (d *PGAutoFailoverDeploy) CheckEnv() error {
	logger.Infof("检查环境\n")

	if !d.NewPGdata {
		if err := d.Monitor.MonitorInstall(d.Param.Pgmonitor, true); err != nil {
			return err
		}
	} else {
		for _, newpgnode := range d.NewPGslave {
			if err := newpgnode.PGdataInstall(d.Param.Pgnode, newpgnode.Host, true, false); err != nil {
				return err
			}
		}
	}

	if err := d.PGmaster.PGdataInstall(d.Param.Pgnode, d.PGmaster.Host, true, true); err != nil {
		return err
	}

	for _, slave := range d.PGslave {
		if err := slave.PGdataInstall(d.Param.Pgnode, slave.Host, true, false); err != nil {
			return err
		}
	}
	return nil
}

func (d *PGAutoFailoverDeploy) InstallAndInitSlave() error {

	if err := d.Install(); err != nil {
		return err
	}

	if !d.NewPGdata {
		monitor_autoctl_path := filepath.Join(d.Param.Pgmonitor.Dir, config.ServerDir, "bin/pg_autoctl")
		node_autoctl_path := filepath.Join(d.Param.Pgnode.Dir, config.ServerDir, "bin/pg_autoctl")
		psql_path := filepath.Join(d.Param.Pgnode.Dir, config.ServerDir, "bin/psql")
		monitor_data_path := filepath.Join(d.Param.Pgmonitor.Dir, "data")
		node_data_path := filepath.Join(d.Param.Pgnode.Dir, "data")
		logger.Successf("Postgresql PG_auto_failover 集群安装完成\n")
		logger.Successf("系统管理用户: %s\n", d.Param.Pgmonitor.SystemUser)
		logger.Successf("业务连接DB: %s\n", d.Param.Pgnode.Username)
		logger.Successf("业务连接用户: %s\n", d.Param.Pgnode.Username)
		logger.Successf("业务连接密码: %s\n", d.Param.Pgnode.Password)
		logger.Successf("监控机查询集群状态: sudo -u %s %s show state --pgdata  %s \n", d.Param.Pgmonitor.SystemUser, monitor_autoctl_path, monitor_data_path)
		logger.Successf("数据节点查询集群URI: sudo -u %s %s show uri --pgdata  %s \n", d.Param.Pgmonitor.SystemUser, node_autoctl_path, node_data_path)
		logger.Successf("数据节点管理员链接方式: sudo -u %s %s  -p %d -d postgres\n", d.Param.Pgmonitor.SystemUser, psql_path, d.Param.Pgnode.Port)
	}
	return nil
}

func (d *PGAutoFailoverDeploy) Install() error {
	logger.Infof("开始安装\n")
	if !d.NewPGdata {
		if err := d.Monitor.MonitorInstall(d.Param.Pgmonitor, false); err != nil {
			return err
		}
		time.Sleep(10 * time.Second)
	} else {
		for _, newpgnode := range d.NewPGslave {
			if err := newpgnode.PGdataInstall(d.Param.Pgnode, newpgnode.Host, false, false); err != nil {
				return err
			}
		}
	}

	if err := d.PGmaster.PGdataInstall(d.Param.Pgnode, d.PGmaster.Host, false, true); err != nil {
		return err
	}

	for _, slave := range d.PGslave {
		if err := slave.PGdataInstall(d.Param.Pgnode, slave.Host, false, false); err != nil {
			return err
		}
	}
	return nil
}

func (d *PGAutoFailoverDeploy) UNInstall() {
	logger.Infof("开始卸载清理\n")
	for _, slave := range d.PGslave {
		if err := slave.UNInstall(d.Param.Pgnode.Port, d.Param.Pgnode.Dir, config.PGNode, d.Param.Pgnode.SystemUser); err != nil {
			logger.Warningf("卸载数据节点: %s 失败: %v\n", slave.Host, err)
		}
	}

	if err := d.PGmaster.UNInstall(d.Param.Pgnode.Port, d.Param.Pgnode.Dir, config.PGNode, d.Param.Pgnode.SystemUser); err != nil {
		logger.Warningf("卸载数据节点: %s 失败: %v\n", d.PGmaster.Host, err)
	}

	if err := d.Monitor.UNInstall(d.Param.Pgmonitor.Port, d.Param.Pgmonitor.Dir, config.PGMonitor, d.Param.Pgmonitor.SystemUser); err != nil {
		logger.Warningf("卸载监控节点: %s 失败: %v\n", d.Monitor.Host, err)
	}
}

func (d *PGAutoFailoverDeploy) UNInstallNewPGdata() {
	logger.Infof("开始卸载清理新增数据节点\n")
	for _, newpgnode := range d.NewPGslave {
		if err := newpgnode.UNInstall(d.Param.Pgnode.Port, d.Param.Pgnode.Dir, config.PGNode, d.Param.Pgnode.SystemUser); err != nil {
			logger.Warningf("卸载新增的数据节点: %s 失败: %v\n", newpgnode.Host, err)
		}
	}
}
