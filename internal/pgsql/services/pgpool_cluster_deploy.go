/*
@Author : WuWeiJian
@Date : 2021-04-25 11:50
*/

package services

import (
	"dbup/internal/environment"
	"dbup/internal/pgsql/config"
	"dbup/internal/utils/logger"
	"fmt"
	"os"
	"path"
	"strings"
	"time"
)

type PGPoolClusterDeploy struct {
	Param    config.PGPoolClusterParameter
	PGPools  []*PGPoolInstance
	PGMaster *Instance
	PGSlave  *Instance
}

func NewPGPoolClusterDeploy() *PGPoolClusterDeploy {
	return &PGPoolClusterDeploy{}
}

func (d *PGPoolClusterDeploy) Run(c string) error {
	// 初始化参数和配置环节
	if err := d.Param.Load(c); err != nil {
		return err
	}
	d.InitArgs()
	if err := d.Param.Validator(); err != nil {
		return err
	}

	if d.Param.Pgsql.SystemUser == "" {
		d.Param.Pgsql.SystemUser = config.DefaultPGAdminUser
	}

	if d.Param.Pgsql.SystemGroup == "" {
		d.Param.Pgsql.SystemGroup = config.DefaultPGAdminUser
	}

	logger.Infof("初始化部署对象\n")
	if d.Param.Server.SshPassword != "" {
		if err := d.InitIns(); err != nil {
			return err
		}
	} else {
		if err := d.InitInsUseKeyFile(); err != nil {
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

	if err := d.InstallAndInitCluster(); err != nil {
		if !d.Param.NoRollback {
			logger.Warningf("安装失败, 开始回滚\n")
			d.UNInstall()
		}
		return err
	}
	return nil
}

func (d *PGPoolClusterDeploy) RemovePGPoolClusterDeploy(c string, yes bool) error {
	// 初始化参数和配置环节
	if err := d.Param.Load(c); err != nil {
		return err
	}
	// d.InitArgs()
	d.Param.Server.SetDefault()
	if err := d.Param.Server.Validator(); err != nil {
		return err
	}
	if d.Param.Pgsql.Port == 0 {
		return fmt.Errorf("请指定PGSQL端口号")
	}

	if d.Param.Pgsql.Dir == "" {
		return fmt.Errorf("请指定PGSQL数据目录")
	}

	if d.Param.PGPool.Port == 0 {
		return fmt.Errorf("请指定PGPOOL端口号")
	}

	if d.Param.PGPool.Dir == "" {
		return fmt.Errorf("请指定PGPOOL数据目录")
	}

	logger.Warningf("要删除的集群节点以及数据目录: %s:%d %s\n", d.Param.Server.Master, d.Param.Pgsql.Port, d.Param.Pgsql.Dir)
	logger.Warningf("要删除的集群节点以及数据目录: %s:%d %s\n", d.Param.Server.Slave, d.Param.Pgsql.Port, d.Param.Pgsql.Dir)
	for _, ip := range strings.Split(d.Param.Server.PGPools, ",") {
		logger.Warningf("要删除的集群节点以及数据目录: %s:%d %s\n", ip, d.Param.PGPool.Port, d.Param.PGPool.Dir)
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
		if err := d.InitIns(); err != nil {
			return err
		}
	} else {
		if err := d.InitInsUseKeyFile(); err != nil {
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

func (d *PGPoolClusterDeploy) InitArgs() {
	d.Param.Server.SetDefault()
	d.Param.Pgsql.InitArgs()
	d.Param.PGPool.PGPoolIP = d.Param.Server.PGPools
	d.Param.PGPool.Username = d.Param.Pgsql.Username
	d.Param.PGPool.Password = d.Param.Pgsql.Password
	d.Param.PGPool.Address = d.Param.Pgsql.Address
	d.Param.PGPool.PGMaster = d.Param.Server.Master
	d.Param.PGPool.PGSlave = d.Param.Server.Slave
	d.Param.PGPool.PGPort = d.Param.Pgsql.Port
	d.Param.PGPool.PGDir = d.Param.Pgsql.Dir
	d.Param.PGPool.InitArgs()
}

func (d *PGPoolClusterDeploy) InitIns() error {
	var err error
	if d.PGMaster, err = NewInstance(d.Param.Server.TmpDir,
		d.Param.Server.Master,
		d.Param.Server.SshUser,
		d.Param.Server.SshPassword,
		d.Param.Server.SshPort,
		d.Param.Pgsql,
		0); err != nil {
		return err
	}
	if d.PGSlave, err = NewInstance(d.Param.Server.TmpDir,
		d.Param.Server.Slave,
		d.Param.Server.SshUser,
		d.Param.Server.SshPassword,
		d.Param.Server.SshPort,
		d.Param.Pgsql,
		0); err != nil {
		return err
	}
	for i, pgpool := range strings.Split(d.Param.Server.PGPools, ",") {
		d.Param.PGPool.NodeID = i
		// if err := d.Param.PGPool.Validator(); err != nil {
		// 	return err
		// }
		pl, err := NewPGPoolInstance(d.Param.Server.TmpDir,
			pgpool,
			d.Param.Server.SshUser,
			d.Param.Server.SshPassword,
			d.Param.Server.SshPort,
			d.Param.PGPool)
		if err != nil {
			return err
		}
		d.PGPools = append(d.PGPools, pl)

	}
	return nil
}

func (d *PGPoolClusterDeploy) InitInsUseKeyFile() error {
	var err error
	if d.PGMaster, err = NewInstanceUseKeyFile(d.Param.Server.TmpDir,
		d.Param.Server.Master,
		d.Param.Server.SshUser,
		d.Param.Server.SshKeyFile,
		d.Param.Server.SshPort,
		d.Param.Pgsql,
		0); err != nil {
		return err
	}
	if d.PGSlave, err = NewInstanceUseKeyFile(d.Param.Server.TmpDir,
		d.Param.Server.Slave,
		d.Param.Server.SshUser,
		d.Param.Server.SshKeyFile,
		d.Param.Server.SshPort,
		d.Param.Pgsql,
		0); err != nil {
		return err
	}
	for i, pgpool := range strings.Split(d.Param.Server.PGPools, ",") {
		d.Param.PGPool.NodeID = i
		// if err := d.Param.PGPool.Validator(); err != nil {
		// 	return err
		// }

		pl, err := NewPGPoolInstanceUseKeyFile(d.Param.Server.TmpDir,
			pgpool,
			d.Param.Server.SshUser,
			d.Param.Server.SshKeyFile,
			d.Param.Server.SshPort,
			d.Param.PGPool)
		if err != nil {
			return err
		}
		d.PGPools = append(d.PGPools, pl)

	}
	return nil
}

func (d *PGPoolClusterDeploy) InstallAndInitCluster() error {
	if err := d.PGMaster.Install(d.Param.Pgsql, false, false, d.Param.Pgsql.Ipv6); err != nil {
		return err
	}
	if err := d.PGSlave.Install(d.Param.Pgsql, false, true, d.Param.Pgsql.Ipv6); err != nil {
		return err
	}

	logger.Infof("添加主从同步用户\n")
	//if err := d.master.AddPgHba(strings.Split(d.Param.Server.Slaves, ",")); err != nil {
	//	return err
	//}
	// if err := d.PGMaster.CreateReplUser(d.Param.Server.Slave); err != nil {
	// 	return err
	// }

	logger.Infof("添加用户的pgpool机器访问权限\n")
	if err := d.PGMaster.UserGrant(config.DefaultPGReplUser, config.DefaultPGAdminUser, d.Param.Server.PGPools); err != nil {
		return err
	}
	if err := d.PGMaster.UserGrant(config.DefaultPGReplUser, config.DefaultPGAdminUser, "0.0.0.0/0"); err != nil {
		return err
	}
	if err := d.PGMaster.UserGrant(d.Param.Pgsql.Username, d.Param.Pgsql.Username, d.Param.Server.PGPools); err != nil {
		return err
	}

	if err := d.ReplicaSlave(); err != nil {
		return err
	}

	logger.Infof("5秒后检查主从集群状态\n")
	time.Sleep(5 * time.Second)
	if err := d.PGMaster.CheckSlaves(d.Param.Server.Slave); err != nil {
		return err
	}
	logger.Successf("从库正常, 开始安装pgpool\n")
	for _, pgpool := range d.PGPools {
		if err := pgpool.Install(false); err != nil {
			return err
		}
	}
	logger.Infof("30秒后检查pgpool状态\n")
	time.Sleep(30 * time.Second)
	for _, pgpool := range d.PGPools {
		if err := pgpool.CheckSelect(d.Param.PGPool.Port, d.Param.PGPool.Username, d.Param.PGPool.Password, d.Param.PGPool.Username); err != nil {
			return fmt.Errorf("pgpool(%s) 状态异常: %v\n", pgpool.Host, err)
		}
	}
	logger.Successf("集群搭建成功\n")
	return nil
}

func (d *PGPoolClusterDeploy) CheckTmpDir() error {
	logger.Infof("检查目标机器的临时目录\n")
	if err := d.PGMaster.CheckTmpDir(); err != nil {
		return err
	}
	if err := d.PGSlave.CheckTmpDir(); err != nil {
		return err
	}
	for _, pgpool := range d.PGPools {
		if err := pgpool.CheckTmpDir(); err != nil {
			return err
		}
	}
	return nil
}

func (d *PGPoolClusterDeploy) Scp() error {
	logger.Infof("将所需文件复制到目标机器\n")
	source := path.Join(environment.GlobalEnv().ProgramPath, "..")
	logger.Infof("复制pgsql到: %s\n", d.PGMaster.Host)
	if err := d.PGMaster.Scp(source); err != nil {
		return err
	}
	logger.Infof("复制pgsql到: %s\n", d.PGSlave.Host)
	if err := d.PGSlave.Scp(source); err != nil {
		return err
	}
	for _, pgpool := range d.PGPools {
		logger.Infof("复制pgpool到: %s\n", pgpool.Host)
		if err := pgpool.Scp(source); err != nil {
			return err
		}
	}
	return nil
}

func (d *PGPoolClusterDeploy) DropTmpDir() {
	logger.Infof("删除目标机器的临时目录\n")
	_ = d.PGMaster.DropTmpDir()
	_ = d.PGSlave.DropTmpDir()
	for _, pgpool := range d.PGPools {
		_ = pgpool.DropTmpDir()
	}
}

func (d *PGPoolClusterDeploy) CheckEnv() error {
	logger.Infof("检查环境\n")
	if err := d.PGMaster.Install(d.Param.Pgsql, true, false, d.Param.Pgsql.Ipv6); err != nil {
		return err
	}
	if err := d.PGSlave.Install(d.Param.Pgsql, true, false, d.Param.Pgsql.Ipv6); err != nil {
		return err
	}
	for _, pgpool := range d.PGPools {
		if err := pgpool.Install(true); err != nil {
			return err
		}
	}
	return nil
}

func (d *PGPoolClusterDeploy) Install() error {
	logger.Infof("开始安装\n")
	if err := d.PGMaster.Install(d.Param.Pgsql, false, false, d.Param.Pgsql.Ipv6); err != nil {
		return err
	}
	if err := d.PGSlave.Install(d.Param.Pgsql, false, false, d.Param.Pgsql.Ipv6); err != nil {
		return err
	}
	for _, pgpool := range d.PGPools {
		if err := pgpool.Install(false); err != nil {
			return err
		}
	}
	return nil
}

func (d *PGPoolClusterDeploy) UNInstall() {
	logger.Infof("开始卸载清理\n")
	if err := d.PGMaster.UNInstall(d.Param.Pgsql); err != nil {
		logger.Warningf("卸载节点: %s 失败: %v\n", d.PGMaster.Host, err)
	}
	if err := d.PGSlave.UNInstall(d.Param.Pgsql); err != nil {
		logger.Warningf("卸载节点: %s 失败: %v\n", d.PGSlave.Host, err)
	}
	for _, pgpool := range d.PGPools {
		if err := pgpool.UNInstall(); err != nil {
			logger.Warningf("卸载节点: %s 失败: %v\n", pgpool.Host, err)
		}
	}
}

func (d *PGPoolClusterDeploy) ReplicaSlave() error {
	logger.Infof("初始化从库\n")
	// if err := d.PGSlave.Replication(d.Param.Server.Master); err != nil {
	// 	return err
	// }
	if err := d.PGSlave.ChownData(d.Param.Pgsql.SystemUser, d.Param.Pgsql.SystemGroup); err != nil {
		return err
	}
	if err := d.PGSlave.SystemCtl("start"); err != nil {
		return err
	}
	return nil
}
