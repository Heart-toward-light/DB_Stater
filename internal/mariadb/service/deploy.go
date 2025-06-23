package service

import (
	"dbup/internal/environment"
	"dbup/internal/mariadb/config"
	"dbup/internal/mariadb/dao"
	"dbup/internal/utils/command"
	"dbup/internal/utils/logger"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type MariaDBDeploy struct {
	option config.MariaDBDeployOptions
	master *MariaDBInstance
	slaves []*MariaDBInstance
}

func NewmariadbDeploy() *MariaDBDeploy {
	return &MariaDBDeploy{}
}

func (d *MariaDBDeploy) Run(c string) error {
	//初始化参数和配置环节
	if err := d.option.Load(c); err != nil {
		return err
	}

	d.option.MariaDB.Parameter()
	d.option.Server.SetDefault()

	if err := d.option.Validator(); err != nil {
		return err
	}

	logger.Infof("初始化部署对象\n")
	if d.option.Server.Password != "" {
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
		if !d.option.NoRollback {
			logger.Warningf("安装失败, 开始回滚\n")
			d.UNInstall()
		}
		return err
	}

	return nil
}

func (d *MariaDBDeploy) RemoveCluster(c string, yes bool) error {
	// 初始化参数和配置环节
	if err := d.option.Load(c); err != nil {
		return err
	}
	d.option.Server.SetDefault()
	if err := d.option.Server.Validator(); err != nil {
		return err
	}

	if d.option.MariaDB.Port == 0 {
		return fmt.Errorf("请指定要删除集群的端口号")
	}

	if d.option.MariaDB.Dir == "" {
		return fmt.Errorf("请指定要删除集群的数据目录")
	}

	for _, ip := range strings.Split(d.option.Server.Address, ",") {
		logger.Warningf("要删除的集群节点以及数据目录: %s:%d %s\n", ip, d.option.MariaDB.Port, d.option.MariaDB.Dir)
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
	if d.option.Server.Password != "" {
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

func (d *MariaDBDeploy) Init() error {
	var err error
	ips := strings.Split(d.option.Server.Address, ",")
	if d.master, err = NewmariaDBInstance(d.option.Server.TmpDir,
		ips[0],
		d.option.Server.User,
		d.option.Server.Password,
		d.option.Server.SshPort,
		d.option.MariaDB); err != nil {
		return err
	}
	for _, slave := range ips[1:] {
		s, err := NewmariaDBInstance(d.option.Server.TmpDir,
			slave,
			d.option.Server.User,
			d.option.Server.Password,
			d.option.Server.SshPort,
			d.option.MariaDB)
		if err != nil {
			return err
		}
		s.Inst.Option.Join = ips[0]
		d.slaves = append(d.slaves, s)
	}

	return nil
}

func (d *MariaDBDeploy) InitUseKeyFile() error {
	var err error
	ips := strings.Split(d.option.Server.Address, ",")
	if d.master, err = NewmariaDBInstanceUseKeyFile(d.option.Server.TmpDir,
		ips[0],
		d.option.Server.User,
		d.option.Server.KeyFile,
		d.option.Server.SshPort,
		d.option.MariaDB); err != nil {
		return err
	}
	for _, slave := range ips[1:] {
		s, err := NewmariaDBInstanceUseKeyFile(d.option.Server.TmpDir,
			slave,
			d.option.Server.User,
			d.option.Server.KeyFile,
			d.option.Server.SshPort,
			d.option.MariaDB)
		if err != nil {
			return err
		}
		s.Inst.Option.Join = ips[0]
		d.slaves = append(d.slaves, s)
	}

	return nil
}

func (d *MariaDBDeploy) CheckTmpDir() error {
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

func (d *MariaDBDeploy) DropTmpDir() {
	logger.Infof("删除目标机器的临时目录\n")
	_ = d.master.DropTmpDir()

	for _, slave := range d.slaves {
		_ = slave.DropTmpDir()
	}
}

func (d *MariaDBDeploy) Scp() error {
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

func (d *MariaDBDeploy) CheckEnv() error {

	logger.Infof("检查环境\n")
	if err := d.master.Install(true, false, 1); err != nil {
		return err
	}
	for _, slave := range d.slaves {
		if err := slave.Install(true, false, 1); err != nil {
			return err
		}
	}

	return nil
}

func (d *MariaDBDeploy) Install() error {
	logger.Infof("开始安装\n")
	if d.option.MariaDB.Clustermode == "MM" {
		d.option.MariaDB.AutoIncrement = 2
		if err := d.master.Install(false, false, d.option.MariaDB.AutoIncrement); err != nil {
			return err
		}
		for _, slave := range d.slaves {
			if err := slave.Install(false, d.option.MariaDB.AddSlave, d.option.MariaDB.AutoIncrement+1); err != nil {
				return err
			}
		}
	} else if d.option.MariaDB.Clustermode == "MS" {
		d.option.MariaDB.AddSlave = true
		if err := d.master.Install(false, false, d.option.MariaDB.AutoIncrement); err != nil {
			return err
		}
		for _, slave := range d.slaves {
			if err := slave.Install(false, d.option.MariaDB.AddSlave, d.option.MariaDB.AutoIncrement); err != nil {
				return err
			}
		}
	}

	return nil
}

func (d *MariaDBDeploy) UNInstall() {
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

func (d *MariaDBDeploy) InstallAndInitSlave() error {
	if err := d.Install(); err != nil {
		return err
	}

	// 检查集群状态
	logger.Infof("等待集群状态:\n")
	switch d.option.MariaDB.Clustermode {
	case config.MariaDBModeMS:
		stat := false
		for i := 1; i <= 6; i++ {
			time.Sleep(3 * time.Second)
			if err := d.CheckReplicaSetStatus(); err == nil {
				stat = true
				break
			} else {
				logger.Warningf("%v \n", err)
			}
		}

		if !stat {
			return fmt.Errorf("主从状态异常")
		}

		logger.Infof("主库地址: %s:%d \n", d.master.Host, d.option.MariaDB.Port)
		for _, slave := range d.slaves {
			logger.Infof("从库地址: %s:%d \n", slave.Host, d.option.MariaDB.Port)
		}

		logger.Successf("MariaDB 主从集群搭建成功\n")
	case config.MariaDBModeMM:

		time.Sleep(3 * time.Second)

		if err := d.CheckReplicaSetStatus(); err != nil {
			return err
		}

		if err := d.MMChangeSlave(); err != nil {
			return err
		}

		time.Sleep(3 * time.Second)

		if err := d.CheckMasterReplicaStatus(); err != nil {
			return err
		}

		logger.Infof("主库地址: %s:%d \n", d.master.Host, d.option.MariaDB.Port)
		for _, slave := range d.slaves {
			logger.Infof("备主库地址: %s:%d \n", slave.Host, d.option.MariaDB.Port)
		}

		logger.Successf("MariaDB 双主集群搭建成功\n")
	}

	logger.Successf("MariaDB 管理用户:root\n")
	logger.Successf("MariaDB 管理密码:%s\n", d.option.MariaDB.Password)
	logger.Successf("MariaDB 复制用户:%s\n", d.option.MariaDB.Repluser)
	logger.Successf("MariaDB 复制密码:%s\n", d.option.MariaDB.ReplPassword)
	logger.Successf("启动方式:systemctl start %s\n", fmt.Sprintf(config.ServiceFileName, d.option.MariaDB.Port))
	logger.Successf("关闭方式:systemctl stop %s\n", fmt.Sprintf(config.ServiceFileName, d.option.MariaDB.Port))
	logger.Successf("重启方式:systemctl restart %s\n", fmt.Sprintf(config.ServiceFileName, d.option.MariaDB.Port))
	logger.Successf("本地登录命令: %s  -uroot -p'%s' --host 127.0.0.1 --port %d\n", filepath.Join(d.option.MariaDB.Dir, "bin", "mariadb"), d.option.MariaDB.Password, d.option.MariaDB.Port)

	return nil
}

func (d *MariaDBDeploy) MMChangeSlave() error {
	// fmt.Sprintf("cd %s; rm -rf *", filepath.ToSlash(i.TmpDir))
	conn, err := command.NewConnection(d.master.Host, d.option.Server.User, d.option.Server.Password, d.option.Server.SshPort, 30)
	if err != nil {
		return fmt.Errorf("在机器: %s 上, 建立ssh连接失败: %v", d.master.Host, err)
	}

	for _, slave := range d.slaves {
		change_cmd := fmt.Sprintf("%s  -uroot -p'%s' --host 127.0.0.1 --port %d -e "+
			"\"CHANGE MASTER TO MASTER_HOST='%s', MASTER_PORT=%d, MASTER_USER='%s', MASTER_PASSWORD='%s',"+
			" MASTER_USE_GTID=slave_pos; start slave; \" ", filepath.Join(d.option.MariaDB.Dir, "bin", "mariadb"), d.option.MariaDB.Password,
			d.option.MariaDB.Port, slave.Host, d.option.MariaDB.Port, d.option.MariaDB.Repluser, d.option.MariaDB.ReplPassword)

		if stdout, err := conn.Run(change_cmd); err != nil {
			return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", d.master.Host, change_cmd, err, stdout)
		}
	}
	return nil
}

func (d *MariaDBDeploy) CheckMasterReplicaStatus() error {
	conn, err := dao.NewMariaDBConn(d.master.Host, d.option.MariaDB.Port, d.option.MariaDB.Repluser, d.option.MariaDB.ReplPassword, "")
	if err != nil {
		return err
	}
	defer conn.DB.Close()

	status, err := conn.ShowSlaveStatus()
	if err != nil {
		return fmt.Errorf("主库 %s:%d 同步状态异常: %s ", d.master.Host, d.option.MariaDB.Port, err)
	}

	for k, v := range status {
		if v.Valid {

			if k == "Slave_IO_Running" && v.String != "Yes" {
				return fmt.Errorf("主库 %s:%d IO线程同步状态异常", d.master.Host, d.option.MariaDB.Port)
			}

			if k == "Slave_SQL_Running" && v.String != "Yes" {
				return fmt.Errorf("主库 %s:%d SQL线程同步状态异常", d.master.Host, d.option.MariaDB.Port)
			}

		}
	}

	return nil
}

func (d *MariaDBDeploy) CheckReplicaSetStatus() error {

	for _, slave := range d.slaves {
		conn, err := dao.NewMariaDBConn(slave.Host, d.option.MariaDB.Port, d.option.MariaDB.Repluser, d.option.MariaDB.ReplPassword, "")
		if err != nil {
			return err
		}
		defer conn.DB.Close()

		status, err := conn.ShowSlaveStatus()
		if err != nil {
			return fmt.Errorf("从库 %s:%d 同步状态异常: %s ", slave.Host, d.option.MariaDB.Port, err)
		}

		for k, v := range status {
			if v.Valid {

				if k == "Slave_IO_Running" && v.String != "Yes" {
					return fmt.Errorf("从库 %s:%d IO线程同步状态异常", slave.Host, d.option.MariaDB.Port)
				}

				if k == "Slave_SQL_Running" && v.String != "Yes" {
					return fmt.Errorf("从库 %s:%d SQL线程同步状态异常", slave.Host, d.option.MariaDB.Port)
				}

			}
		}
	}

	return nil
}
