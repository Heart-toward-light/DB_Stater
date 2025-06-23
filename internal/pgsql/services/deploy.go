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
	"strings"
	"time"
)

type Deploy struct {
	Param  config.Parameter
	master *Instance
	slaves []*Instance
}

func NewDeploy() *Deploy {
	return &Deploy{}
}

func (d *Deploy) Run(c string) error {
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
		d.UNInstall()
		return err
	}
	return nil
}

func (d *Deploy) RemoveDeploy(c string, yes bool) error {
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

func (d *Deploy) InstallAndInitSlave() error {
	if err := d.Install(); err != nil {
		return err
	}

	logger.Infof("添加主从同步用户\n")
	//if err := d.master.AddPgHba(strings.Split(d.Param.Server.Slaves, ",")); err != nil {
	//	return err
	//}

	PGReplPass := utils.GeneratePasswd(config.DefaultPGPassLength)
	if err := d.master.CreateReplUser(d.Param.Server.Slaves, PGReplPass); err != nil {
		return err
	}
	if err := d.ReplicaSlave(PGReplPass); err != nil {
		return err
	}

	logger.Infof("5秒后检查集群状态\n")
	time.Sleep(5 * time.Second)
	if err := d.master.CheckSlaves(d.Param.Server.Slaves); err != nil {
		return err
	}

	logger.Successf("复制用户名: %s\n", config.DefaultPGReplUser)
	logger.Successf("复制用户密码: %s\n", PGReplPass)
	logger.Successf("从库正常\n")
	logger.Successf("集群搭建成功\n")
	return nil
}

func (d *Deploy) Init() error {
	var err error
	if d.master, err = NewInstance(d.Param.Server.TmpDir,
		d.Param.Server.Master,
		d.Param.Server.User,
		d.Param.Server.Password,
		d.Param.Server.SshPort,
		d.Param.Pgsql,
		0); err != nil {
		return err
	}
	for _, slave := range strings.Split(d.Param.Server.Slaves, ",") {
		s, err := NewInstance(d.Param.Server.TmpDir,
			slave,
			d.Param.Server.User,
			d.Param.Server.Password,
			d.Param.Server.SshPort,
			d.Param.Pgsql,
			0)
		if err != nil {
			return err
		}
		d.slaves = append(d.slaves, s)
	}
	return nil
}

func (d *Deploy) InitUseKeyFile() error {
	var err error
	if d.master, err = NewInstanceUseKeyFile(d.Param.Server.TmpDir,
		d.Param.Server.Master,
		d.Param.Server.User,
		d.Param.Server.KeyFile,
		d.Param.Server.SshPort,
		d.Param.Pgsql,
		0); err != nil {
		return err
	}
	for _, slave := range strings.Split(d.Param.Server.Slaves, ",") {
		s, err := NewInstanceUseKeyFile(d.Param.Server.TmpDir,
			slave,
			d.Param.Server.User,
			d.Param.Server.KeyFile,
			d.Param.Server.SshPort,
			d.Param.Pgsql,
			0)
		if err != nil {
			return err
		}
		d.slaves = append(d.slaves, s)
	}
	return nil
}

func (d *Deploy) CheckTmpDir() error {
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

func (d *Deploy) Scp() error {
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

func (d *Deploy) DropTmpDir() {
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

func (d *Deploy) CheckEnv() error {
	logger.Infof("检查环境\n")
	if err := d.master.Install(d.Param.Pgsql, true, false, d.Param.Pgsql.Ipv6); err != nil {
		return err
	}
	for _, slave := range d.slaves {
		if err := slave.Install(d.Param.Pgsql, true, false, d.Param.Pgsql.Ipv6); err != nil {
			return err
		}
	}
	return nil
}

func (d *Deploy) Install() error {
	logger.Infof("开始安装\n")
	if err := d.master.Install(d.Param.Pgsql, false, false, d.Param.Pgsql.Ipv6); err != nil {
		return err
	}
	for _, slave := range d.slaves {
		if err := slave.Install(d.Param.Pgsql, false, true, d.Param.Pgsql.Ipv6); err != nil {
			return err
		}
	}
	return nil
}

func (d *Deploy) UNInstall() {
	logger.Infof("开始卸载清理\n")
	if err := d.master.UNInstall(d.Param.Pgsql); err != nil {
		logger.Warningf("卸载节点: %s 失败: %v\n", d.master.Host, err)
	}
	for _, slave := range d.slaves {
		if err := slave.UNInstall(d.Param.Pgsql); err != nil {
			logger.Warningf("卸载节点: %s 失败: %v\n", slave.Host, err)
		}
	}
}

func (d *Deploy) ReplicaSlave(PGReplPass string) error {
	logger.Infof("初始化从库\n")
	for _, slave := range d.slaves {
		//if err := slave.SystemCtl("stop"); err != nil {
		//	return err
		//}
		//if err := slave.RemoveData(); err != nil {
		//	return err
		//}
		if err := slave.Replication(d.Param.Server.Master, PGReplPass); err != nil {
			return err
		}
		if err := slave.ChownData(d.Param.Pgsql.SystemUser, d.Param.Pgsql.SystemGroup); err != nil {
			return err
		}
		if err := slave.SystemCtl("start"); err != nil {
			return err
		}
	}
	return nil
}
