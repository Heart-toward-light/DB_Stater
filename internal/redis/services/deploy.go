/*
@Author : WuWeiJian
@Date : 2021-04-13 11:01
*/

package services

import (
	"dbup/internal/environment"
	"dbup/internal/redis/config"
	"dbup/internal/utils/logger"
	"fmt"
	"os"
	"path"
	"strings"
	"time"
)

type Deploy struct {
	param  config.Parameter
	master *Instance
	slaves []*Instance
}

func NewDeploy() *Deploy {
	return &Deploy{}
}

func (d *Deploy) Run(c string) error {
	// 初始化参数和配置环节
	if err := d.param.Load(c); err != nil {
		return err
	}

	d.param.Server.SetDefault()

	if err := d.param.Validator(); err != nil {
		return err
	}

	d.param.Redis.InitArgs()
	logger.Infof("初始化部署对象\n")
	if d.param.Server.Password != "" {
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
	if err := d.param.Load(c); err != nil {
		return err
	}

	d.param.Server.SetDefault()

	if err := d.param.Server.Validator(); err != nil {
		return err
	}

	if d.param.Redis.Port == 0 {
		return fmt.Errorf("请指定要删除集群的端口号\n")
	}

	if d.param.Redis.Dir == "" {
		return fmt.Errorf("请指定要删除集群的数据目录\n")
	}

	logger.Warningf("要删除的集群节点以及数据目录: %s:%d %s\n", d.param.Server.Master, d.param.Redis.Port, d.param.Redis.Dir)
	for _, ip := range strings.Split(d.param.Server.Slaves, ",") {
		logger.Warningf("要删除的集群节点以及数据目录: %s:%d %s\n", ip, d.param.Redis.Port, d.param.Redis.Dir)
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

	if d.param.Server.Password != "" {
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

	if err := d.ReplicaSlave(); err != nil {
		return err
	}

	logger.Infof("5秒后检查集群状态\n")
	time.Sleep(5 * time.Second)
	for _, slave := range d.slaves {
		if err := slave.CheckSlaves(); err != nil {
			return err
		}
	}

	logger.Successf("从库正常\n")
	logger.Successf("集群搭建成功\n")
	return nil
}

func (d *Deploy) Init() error {
	var err error
	if d.master, err = NewInstance(d.param.Server.TmpDir,
		d.param.Server.Master,
		d.param.Server.User,
		d.param.Server.Password,
		d.param.Server.SshPort,
		d.param.Redis); err != nil {
		return err
	}
	for _, slave := range strings.Split(d.param.Server.Slaves, ",") {
		s, err := NewInstance(d.param.Server.TmpDir,
			slave,
			d.param.Server.User,
			d.param.Server.Password,
			d.param.Server.SshPort,
			d.param.Redis)
		if err != nil {
			return err
		}
		d.slaves = append(d.slaves, s)
	}
	return nil
}

func (d *Deploy) InitUseKeyFile() error {
	var err error
	if d.master, err = NewInstanceUseKeyFile(d.param.Server.TmpDir,
		d.param.Server.Master,
		d.param.Server.User,
		d.param.Server.KeyFile,
		d.param.Server.SshPort,
		d.param.Redis); err != nil {
		return err
	}
	for _, slave := range strings.Split(d.param.Server.Slaves, ",") {
		s, err := NewInstanceUseKeyFile(d.param.Server.TmpDir,
			slave,
			d.param.Server.User,
			d.param.Server.KeyFile,
			d.param.Server.SshPort,
			d.param.Redis)
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

func (d *Deploy) CheckEnv() error {
	logger.Infof("检查环境\n")
	if err := d.master.Install(false, true, d.param.Redis.Ipv6); err != nil {
		return err
	}
	for _, slave := range d.slaves {
		if err := slave.Install(false, true, d.param.Redis.Ipv6); err != nil {
			return err
		}
	}
	return nil
}

func (d *Deploy) Install() error {
	logger.Infof("开始安装\n")
	if err := d.master.Install(false, false, d.param.Redis.Ipv6); err != nil {
		return err
	}
	for _, slave := range d.slaves {
		if err := slave.Install(false, false, d.param.Redis.Ipv6); err != nil {
			return err
		}
	}
	return nil
}

func (d *Deploy) UNInstall() {
	logger.Infof("开始卸载清理\n")
	if err := d.master.UNInstall(); err != nil {
		logger.Warningf("卸载节点: %s 失败: %v\n", d.master.Host, err)
	}
	for _, slave := range d.slaves {
		if err := slave.UNInstall(); err != nil {
			logger.Warningf("卸载节点: %s 失败: %v\n", slave.Host, err)
		}
	}
}

func (d *Deploy) ReplicaSlave() error {
	logger.Infof("初始化从库\n")
	for _, slave := range d.slaves {
		if err := slave.Replication(d.param.Server.Master, d.master.Inst.port); err != nil {
			return err
		}

	}
	return nil
}
