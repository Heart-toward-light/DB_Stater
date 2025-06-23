package service

import (
	"dbup/internal/environment"
	"dbup/internal/mariadb/config"
	"dbup/internal/utils/logger"
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

type GaleraDeploy struct {
	option     config.MariaDBDeployOptions
	masterhead *MariaDBInstance
	masterlist []*MariaDBInstance
}

func NewGaleraDeploy() *GaleraDeploy {
	return &GaleraDeploy{}
}

func (d *GaleraDeploy) Run(c string) error {

	if err := d.option.Load(c); err != nil {
		return err
	}

	d.option.MariaDB.GaleraParameter()

	// 验证 Galera 相关参数配置
	if err := d.option.GaleraValidator(); err != nil {
		return err
	}

	// g.Galera.Wsrep_cluster_address = fmt.Sprintf("gcomm://%s", g.Server.Address)

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

func (d *GaleraDeploy) Init() error {
	var err error
	ips := strings.Split(d.option.Server.Address, ",")
	if d.masterhead, err = NewmariaDBInstance(d.option.Server.TmpDir,
		ips[0],
		d.option.Server.User,
		d.option.Server.Password,
		d.option.Server.SshPort,
		d.option.MariaDB); err != nil {
		return err
	}
	for _, masters := range ips[1:] {
		s, err := NewmariaDBInstance(d.option.Server.TmpDir,
			masters,
			d.option.Server.User,
			d.option.Server.Password,
			d.option.Server.SshPort,
			d.option.MariaDB)
		if err != nil {
			return err
		}
		s.Inst.Option.Join = ips[0]
		d.masterlist = append(d.masterlist, s)
	}

	return nil
}

func (d *GaleraDeploy) InitUseKeyFile() error {
	var err error
	ips := strings.Split(d.option.Server.Address, ",")
	if d.masterhead, err = NewmariaDBInstanceUseKeyFile(d.option.Server.TmpDir,
		ips[0],
		d.option.Server.User,
		d.option.Server.KeyFile,
		d.option.Server.SshPort,
		d.option.MariaDB); err != nil {
		return err
	}
	for _, masters := range ips[1:] {
		s, err := NewmariaDBInstanceUseKeyFile(d.option.Server.TmpDir,
			masters,
			d.option.Server.User,
			d.option.Server.KeyFile,
			d.option.Server.SshPort,
			d.option.MariaDB)
		if err != nil {
			return err
		}
		s.Inst.Option.Join = ips[0]
		d.masterlist = append(d.masterlist, s)
	}

	return nil
}

func (d *GaleraDeploy) CheckTmpDir() error {
	logger.Infof("检查目标机器的临时目录\n")
	if err := d.masterhead.CheckTmpDir(); err != nil {
		return err
	}
	for _, slave := range d.masterlist {
		if err := slave.CheckTmpDir(); err != nil {
			return err
		}
	}

	return nil
}

func (d *GaleraDeploy) DropTmpDir() {
	logger.Infof("删除目标机器的临时目录\n")
	_ = d.masterhead.DropTmpDir()

	for _, slave := range d.masterlist {
		_ = slave.DropTmpDir()
	}
}

func (d *GaleraDeploy) Scp() error {
	logger.Infof("将所需文件复制到目标机器\n")
	source := path.Join(environment.GlobalEnv().ProgramPath, "..")
	logger.Infof("复制到: %s\n", d.masterhead.Host)
	if err := d.masterhead.Scp(source); err != nil {
		return err
	}
	for _, slave := range d.masterlist {
		logger.Infof("复制到: %s\n", slave.Host)
		if err := slave.Scp(source); err != nil {
			return err
		}
	}

	return nil
}

func (d *GaleraDeploy) CheckEnv() error {

	logger.Infof("检查环境\n")
	if err := d.masterhead.GaleraInstall(true, true, d.option.Server.Address); err != nil {
		return err
	}
	for _, slave := range d.masterlist {
		if err := slave.GaleraInstall(true, false, d.option.Server.Address); err != nil {
			return err
		}
	}

	return nil
}

func (d *GaleraDeploy) Install() error {
	logger.Infof("开始安装\n")
	if err := d.masterhead.GaleraInstall(false, true, d.option.Server.Address); err != nil {
		return err
	}

	for _, masters := range d.masterlist {
		if err := masters.GaleraInstall(false, false, d.option.Server.Address); err != nil {
			return err
		}
	}

	return nil
}

func (d *GaleraDeploy) InstallAndInitSlave() error {
	if err := d.Install(); err != nil {
		return err
	}

	logger.Successf("MariaDB Galera 集群安装完成\n")
	logger.Successf("MariaDB Galera 管理用户:root\n")
	logger.Successf("MariaDB Galera 管理密码:%s\n", d.option.MariaDB.Password)
	logger.Successf("启动方式:systemctl start %s\n", fmt.Sprintf(config.ServiceFileName, d.option.MariaDB.Port))
	logger.Successf("关闭方式:systemctl stop %s\n", fmt.Sprintf(config.ServiceFileName, d.option.MariaDB.Port))
	logger.Successf("重启方式:systemctl restart %s\n", fmt.Sprintf(config.ServiceFileName, d.option.MariaDB.Port))
	logger.Successf("本地登录命令: %s  -uroot -p'%s' --host 127.0.0.1 --port %d\n", filepath.Join(d.option.MariaDB.Dir, "bin", "mariadb"), d.option.MariaDB.Password, d.option.MariaDB.Port)

	return nil
}

func (d *GaleraDeploy) UNInstall() {
	logger.Infof("开始卸载清理\n")
	if err := d.masterhead.UNInstall(); err != nil {
		logger.Warningf("卸载节点: %s 失败: %v\n", d.masterhead.Host, err)
	}
	for _, masters := range d.masterlist {
		if err := masters.UNInstall(); err != nil {
			logger.Warningf("卸载节点: %s 失败: %v\n", masters.Host, err)
		}
	}

}
