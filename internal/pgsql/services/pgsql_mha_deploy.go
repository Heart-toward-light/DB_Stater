/*
@Author : WuWeiJian
@Date : 2021-02-09 16:39
*/

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

type PGSqlMHADeploy struct {
	Param  config.Parameter
	master *Instance
	slaves []*Instance
}

func NewPGSqlMHADeploy() *PGSqlMHADeploy {
	return &PGSqlMHADeploy{}
}

func (d *PGSqlMHADeploy) Run(c string) error {
	// 初始化参数和配置环节
	if err := d.Param.Load(c); err != nil {
		return err
	}
	d.Param.Server.SetDefault()
	if err := d.Param.Validator(); err != nil {
		return err
	}

	if d.Param.Pgsql.SystemUser == "" {
		d.Param.Pgsql.SystemUser = config.DefaultPGAdminUser
	}

	if d.Param.Pgsql.SystemGroup == "" {
		d.Param.Pgsql.SystemGroup = config.DefaultPGAdminUser
	}

	if err := d.InitRepmgrArg(); err != nil {
		return err
	}

	logger.Infof("初始化部署对象\n")
	if d.Param.Server.Password != "" {
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
		logger.Warningf("安装失败, 开始回滚\n")
		d.UNAllInstall()
		return err
	}

	return nil
}

func (d *PGSqlMHADeploy) RemoveDeploy(c string, yes bool) error {
	// 初始化参数和配置环节
	if err := d.Param.Load(c); err != nil {
		return err
	}
	d.Param.Server.SetDefault()
	if err := d.Param.Server.Validator(); err != nil {
		return err
	}
	if d.Param.Pgsql.Port == 0 {
		return fmt.Errorf("请指定端口号")
	}

	if d.Param.Pgsql.Dir == "" {
		return fmt.Errorf("请指定数据目录")
	}

	d.Param.Pgsql.Libraries = "repmgr"

	logger.Warningf("要删除的集群节点以及数据目录: %s:%d %s\n", d.Param.Server.Master, d.Param.Pgsql.Port, d.Param.Pgsql.Dir)
	for _, ip := range strings.Split(d.Param.Server.Slaves, ",") {
		logger.Warningf("要删除的集群节点以及数据目录: %s:%d %s\n", ip, d.Param.Pgsql.Port, d.Param.Pgsql.Dir)
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
	if d.Param.Server.Password != "" {
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

	d.UNInstall()
	return nil
}

func (d *PGSqlMHADeploy) InitRepmgrArg() error {

	if d.Param.Pgsql.RepmgrDeployMode != "" {
		if d.Param.Pgsql.RepmgrDeployMode != config.RepmgrInitMode && d.Param.Pgsql.RepmgrDeployMode != config.RepmgrAddMode {
			return fmt.Errorf("请指定要部署集群的模式: init_repmgr | add_repmgr ")
		}
	} else {
		d.Param.Pgsql.RepmgrDeployMode = config.RepmgrInitMode
	}

	if d.Param.Pgsql.RepmgrDeployMode == config.RepmgrAddMode {
		Masteriport := fmt.Sprintf("%s:%d", d.Param.Server.Master, d.Param.Pgsql.Port)
		ok, _ := utils.TcpGather(Masteriport)
		if !ok {
			return fmt.Errorf("postgresql 数据库的ip与端口服务 %s 连接异常", Masteriport)
		}

		slaves := strings.Split(d.Param.Server.Slaves, ",")
		if len(slaves) != 2 {
			return fmt.Errorf("扩容 Repmgr 从节点要保证是两个节点")
		}
		// slaves := strings.Split(d., ",")

	}

	if d.Param.Pgsql.Libraries == "" {
		d.Param.Pgsql.Libraries = "repmgr"
	} else if !strings.Contains(d.Param.Pgsql.Libraries, "repmgr") {
		d.Param.Pgsql.Libraries = d.Param.Pgsql.Libraries + ",repmgr"
	}
	if d.Param.Pgsql.RepmgrUser == "" {
		d.Param.Pgsql.RepmgrUser = config.DefaultPGRepmgrUser
	}

	if d.Param.Pgsql.RepmgrPassword == "" {
		d.Param.Pgsql.RepmgrPassword = utils.GeneratePasswd(config.DefaultPGPassLength)
	}

	if d.Param.Pgsql.RepmgrPassword != "" {
		if err := utils.CheckPasswordLever(d.Param.Pgsql.RepmgrPassword); err != nil {
			return err
		}
	}

	if d.Param.Pgsql.RepmgrDBName == "" {
		d.Param.Pgsql.RepmgrDBName = config.DefaultPGRepmgrUser
	}

	return nil
}

func (d *PGSqlMHADeploy) InstallAndInitSlave() error {
	if err := d.Install(); err != nil {
		return err
	}

	switch d.Param.Pgsql.RepmgrDeployMode {
	case config.RepmgrInitMode:
		logger.Infof("添加repmgr用户\n")
		if err := d.master.CreateRepmgrUser(d.Param.Server.Slaves); err != nil {
			return err
		}

		logger.Infof("将pgsql主库实例进行repmgr注册\n")
		time.Sleep(3 * time.Second)
		if err := d.master.RepmgrPrimaryRegister(d.Param.Pgsql); err != nil {
			return err
		}

	case config.RepmgrAddMode:
		logger.Infof("添加repmgr从库授权\n")
		if err := d.master.GrantRepmgrSlaveUser(d.Param.Server.Slaves); err != nil {
			return err
		}
	}
	// 下面是操作从库的相关操作
	logger.Infof("同步从库\n")
	if err := d.RepmgrSlaveCloneAndStart(); err != nil {
		return err
	}

	logger.Infof("3秒后检查主从同步正常, 将pgsql从库注册到repmgr\n")
	time.Sleep(3 * time.Second)
	if err := d.master.CheckSlaves(d.Param.Server.Slaves); err != nil {
		return err
	}
	if err := d.RepmgrStandbyRegister(); err != nil {
		return err
	}

	logger.Infof("3秒后启动守护进程\n")
	time.Sleep(3 * time.Second)
	if err := d.RepmgrDaemon(); err != nil {
		return err
	}

	// logger.Infof("3秒后检查集群状态\n")
	// TODO: repmgr daemon 还没有好检查办法
	// time.Sleep(3 * time.Second)
	// if err := d.master.RepmgrClusterShow(d.Param.Pgsql); err != nil {
	// 	return err
	// }
	// if err := d.master.CheckSlaves(d.Param.Server.Slaves); err != nil {
	// 	return err
	// }
	binfile := filepath.Join(d.Param.Pgsql.Dir, "server", "bin", "repmgr")
	cnfile := filepath.Join(d.Param.Pgsql.Dir, "repmgr", "repmgr.conf")
	logger.Successf("集群搭建成功\n")
	logger.Successf("repmgr 用 户:%s\n", d.Param.Pgsql.RepmgrUser)
	logger.Successf("repmgr 密 码:%s\n", d.Param.Pgsql.RepmgrPassword)
	logger.Successf("查询集群管理信息: sudo -u %s %s -f %s cluster show \n", d.Param.Pgsql.SystemUser, binfile, cnfile)
	logger.Successf("查询repmgr状态: sudo -u %s %s -f %s service status \n", d.Param.Pgsql.SystemUser, binfile, cnfile)

	return nil
}

func (d *PGSqlMHADeploy) Init() error {
	var err error
	if d.master, err = NewInstance(d.Param.Server.TmpDir,
		d.Param.Server.Master,
		d.Param.Server.User,
		d.Param.Server.Password,
		d.Param.Server.SshPort,
		d.Param.Pgsql,
		1001); err != nil {
		return err
	}
	for i, slave := range strings.Split(d.Param.Server.Slaves, ",") {
		s, err := NewInstance(d.Param.Server.TmpDir,
			slave,
			d.Param.Server.User,
			d.Param.Server.Password,
			d.Param.Server.SshPort,
			d.Param.Pgsql,
			1002+i)
		if err != nil {
			return err
		}
		d.slaves = append(d.slaves, s)
	}
	return nil
}

func (d *PGSqlMHADeploy) InitUseKeyFile() error {
	var err error
	if d.master, err = NewInstanceUseKeyFile(d.Param.Server.TmpDir,
		d.Param.Server.Master,
		d.Param.Server.User,
		d.Param.Server.KeyFile,
		d.Param.Server.SshPort,
		d.Param.Pgsql,
		1001); err != nil {
		return err
	}
	for i, slave := range strings.Split(d.Param.Server.Slaves, ",") {
		d.Param.Pgsql.RepmgrNodeID = 10002 + i
		d.Param.Pgsql.RepmgrOwnerIP = slave
		s, err := NewInstanceUseKeyFile(d.Param.Server.TmpDir,
			slave,
			d.Param.Server.User,
			d.Param.Server.KeyFile,
			d.Param.Server.SshPort,
			d.Param.Pgsql,
			1002+i)
		if err != nil {
			return err
		}
		d.slaves = append(d.slaves, s)
	}
	return nil
}

func (d *PGSqlMHADeploy) CheckTmpDir() error {
	logger.Infof("检查目标机器的临时目录\n")
	if err := d.master.CheckTmpDir(); err != nil {
		return err
	}
	for _, slave := range d.slaves {
		if err := slave.CheckTmpDir(); err != nil {
			return err
		}
	}
	return nil
}

func (d *PGSqlMHADeploy) Scp() error {
	logger.Infof("将所需文件复制到目标机器\n")
	source := path.Join(environment.GlobalEnv().ProgramPath, "..")
	logger.Infof("复制到: %s\n", d.master.Host)
	if err := d.master.Scp(source); err != nil {
		return err
	}
	for _, slave := range d.slaves {
		logger.Infof("复制到: %s\n", slave.Host)
		if err := slave.Scp(source); err != nil {
			return err
		}
	}
	return nil
}

func (d *PGSqlMHADeploy) DropTmpDir() {
	logger.Infof("删除目标机器的临时目录\n")
	_ = d.master.DropTmpDir()
	for _, slave := range d.slaves {
		_ = slave.DropTmpDir()
	}
}

//func (d *Deploy) LoopScp() error {
//	source, _ := filepath.Split(environment.GlobalEnv().Program)
//	d.master.Dir = filepath.ToSlash(path.Join(d.master.TmpDir, path.Base(filepath.ToSlash(source))))
//	if !d.master.Conn.IsExists(d.master.TmpDir) {
//		if err := d.master.Conn.MkdirAll(filepath.ToSlash(d.master.TmpDir)); err != nil {
//			return err
//		}
//	}
//	if err := d.master.Conn.Scp(source+"/dbup/internal", filepath.ToSlash(d.master.TmpDir)); err != nil {
//		return err
//	}
//	for _, slave := range d.slaves {
//		slave.Dir = filepath.ToSlash(path.Join(slave.TmpDir, path.Base(filepath.ToSlash(source))))
//		if !slave.Conn.IsExists(slave.TmpDir) {
//			if err := slave.Conn.MkdirAll(filepath.ToSlash(slave.TmpDir)); err != nil {
//				return err
//			}
//		}
//		if err := slave.Conn.Scp(source+"/dbup/internal", filepath.ToSlash(slave.TmpDir)); err != nil {
//			return err
//		}
//	}
//	return nil
//}

//func (d *Deploy) RemoteCmd() error {
//	cmd := fmt.Sprintf("dbup_linux_amd64 pgsql install --yes --no-rollback --port='%d' --username='%s' --password='%s' --memory_size='%s' --dir='%s' --bind_ip='%s' --address='%s'",
//		d.Param.Pgsql.Port,
//		d.Param.Pgsql.Username,
//		d.Param.Pgsql.Password,
//		d.Param.Pgsql.MemorySize,
//		d.Param.Pgsql.Dir,
//		d.Param.Pgsql.BindIP,
//		d.Param.Pgsql.Address,
//	)
//	masterCmd := path.Join(d.master.TmpDir, "bin", cmd)
//	fmt.Println(masterCmd)
//	if _, err := d.master.Conn.Sudo(masterCmd, "", ""); err != nil {
//		return err
//	}
//	for _, slave := range d.slaves {
//		slaveCmd := path.Join(slave.TmpDir, "bin", cmd)
//		if _, err := d.master.Conn.Sudo(slaveCmd, "", ""); err != nil {
//			return err
//		}
//	}
//	return nil
//}

func (d *PGSqlMHADeploy) CheckEnv() error {
	logger.Infof("检查环境\n")
	switch d.Param.Pgsql.RepmgrDeployMode {
	case config.RepmgrInitMode:
		if err := d.master.Install(d.Param.Pgsql, true, false, d.Param.Pgsql.Ipv6); err != nil {
			return err
		}
	case config.RepmgrAddMode:
		if err := d.master.PrimaryInstall(d.Param.Pgsql, true); err != nil {
			return err
		}
	}
	for _, slave := range d.slaves {
		if err := slave.Install(d.Param.Pgsql, true, false, d.Param.Pgsql.Ipv6); err != nil {
			return err
		}
	}
	return nil
}

func (d *PGSqlMHADeploy) Install() error {
	logger.Infof("开始安装\n")
	switch d.Param.Pgsql.RepmgrDeployMode {
	case config.RepmgrInitMode:
		if err := d.master.Install(d.Param.Pgsql, false, false, d.Param.Pgsql.Ipv6); err != nil {
			return err
		}
	case config.RepmgrAddMode:
		if err := d.master.PrimaryInstall(d.Param.Pgsql, false); err != nil {
			return err
		}
	}
	for _, slave := range d.slaves {
		logger.Infof("开始安装从库\n")
		if err := slave.Install(d.Param.Pgsql, false, true, d.Param.Pgsql.Ipv6); err != nil {
			return err
		}
	}
	return nil
}

func (d *PGSqlMHADeploy) UNInstall() {
	logger.Infof("开始卸载清理\n")
	for _, slave := range d.slaves {
		if err := slave.UNInstall(d.Param.Pgsql); err != nil {
			logger.Warningf("卸载节点: %s 失败: %v\n", slave.Host, err)
		}
	}

	if err := d.master.UNInstall(d.Param.Pgsql); err != nil {
		logger.Warningf("卸载节点: %s 失败: %v\n", d.master.Host, err)
	}
}

func (d *PGSqlMHADeploy) UNAllInstall() {
	logger.Infof("开始卸载清理\n")
	for _, slave := range d.slaves {
		if err := slave.UNInstall(d.Param.Pgsql); err != nil {
			logger.Warningf("卸载节点: %s 失败: %v\n", slave.Host, err)
		}
	}

	switch d.Param.Pgsql.RepmgrDeployMode {
	case config.RepmgrInitMode:
		if err := d.master.UNInstall(d.Param.Pgsql); err != nil {
			logger.Warningf("卸载节点: %s 失败: %v\n", d.master.Host, err)
		}
	}
}

func (d *PGSqlMHADeploy) RepmgrSlaveCloneAndStart() error {
	logger.Infof("从库同步数据并启动\n")
	for _, slave := range d.slaves {
		if err := slave.RepmgrStandbyClone(d.Param.Pgsql, d.Param.Server.Master); err != nil {
			return err
		}
		if err := slave.ChownData(d.Param.Pgsql.SystemUser, d.Param.Pgsql.SystemGroup); err != nil {
			return err
		}
		if err := slave.RepmgrStartPostgreSql(d.Param.Pgsql); err != nil {
			return err
		}
	}
	return nil
}

func (d *PGSqlMHADeploy) RepmgrStandbyRegister() error {
	for _, slave := range d.slaves {
		if err := slave.RepmgrStandbyRegister(d.Param.Pgsql); err != nil {
			return err
		}
	}
	return nil
}

func (d *PGSqlMHADeploy) RepmgrDaemon() error {
	logger.Infof("启动repmgr守护进程\n")
	if err := d.master.RepmgrDaemon(d.Param.Pgsql); err != nil {
		return err
	}

	for _, slave := range d.slaves {
		if err := slave.RepmgrDaemon(d.Param.Pgsql); err != nil {
			return err
		}
	}
	return nil
}
