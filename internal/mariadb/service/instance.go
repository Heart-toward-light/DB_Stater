package service

import (
	"dbup/internal/environment"
	"dbup/internal/mariadb/config"
	"dbup/internal/utils/command"
	"fmt"
	"path"
	"path/filepath"
)

type MariaDBInstance struct {
	DbupCmd string
	Host    string
	TmpDir  string
	Inst    *MariaDBInstall
	Conn    *command.Connection
	//spool
}

func NewmariaDBInstance(tmp, host, user, password string, port int, option config.MariaDBOptions) (*MariaDBInstance, error) {
	conn, err := command.NewConnection(host, user, password, port, 30)
	if err != nil {
		return nil, fmt.Errorf("在机器: %s 上, 建立ssh连接失败: %v", host, err)
	}
	inst := NewMariaDBInstall(&option)
	// cmd := fmt.Sprintf("dbup_%s_%s", environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH)
	cmd := "dbup"
	return &MariaDBInstance{DbupCmd: cmd, Host: host, TmpDir: tmp, Inst: inst, Conn: conn}, nil
}

func NewmariaDBInstanceUseKeyFile(tmp, host, user, keyfile string, port int, option config.MariaDBOptions) (*MariaDBInstance, error) {
	conn, err := command.NewConnectionUseKeyFile(host, user, keyfile, port, 30)
	if err != nil {
		return nil, fmt.Errorf("在机器: %s 上, 建立ssh连接失败: %v", host, err)
	}
	inst := NewMariaDBInstall(&option)
	// cmd := fmt.Sprintf("dbup_%s_%s", environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH)
	cmd := "dbup"
	return &MariaDBInstance{DbupCmd: cmd, Host: host, TmpDir: tmp, Inst: inst, Conn: conn}, nil
}

func (i *MariaDBInstance) CheckTmpDir() error {
	if i.Conn.IsExists(filepath.ToSlash(i.TmpDir)) {
		if !i.Conn.IsDir(filepath.ToSlash(i.TmpDir)) {
			return fmt.Errorf("在机器: %s 上, 目标文件(%s)已经存在", i.Host, i.TmpDir)
		}
		b, err := i.Conn.IsEmpty(filepath.ToSlash(i.TmpDir))
		if err != nil {
			return fmt.Errorf("在机器: %s 上, 判断目录(%s)是否为空失败: %v", i.Host, i.TmpDir, err)
		}
		if !b {
			return fmt.Errorf("在机器: %s 上, 目标目录(%s)不为空", i.Host, i.TmpDir)
		}
	}
	return nil
}

func (i *MariaDBInstance) DropTmpDir() error {
	cmd := fmt.Sprintf("cd %s; rm -rf *", filepath.ToSlash(i.TmpDir))
	if stdout, err := i.Conn.Run(cmd); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd, err, stdout)
	}
	return nil
}

func (i *MariaDBInstance) Scp(source string) error {
	if err := i.Conn.MkdirAll(filepath.ToSlash(path.Join(i.TmpDir, "bin"))); err != nil {
		return fmt.Errorf("在机器: %s 上, 创建目录(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "bin")), err)
	}

	if err := i.Conn.MkdirAll(filepath.ToSlash(path.Join(i.TmpDir, "systemd"))); err != nil {
		return fmt.Errorf("在机器: %s 上, 创建目录(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "systemd")), err)
	}

	if err := i.Conn.MkdirAll(filepath.ToSlash(path.Join(i.TmpDir, "package/mariadb"))); err != nil {
		return fmt.Errorf("在机器: %s 上, 创建目录(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "package/mariadb")), err)
	}

	if err := i.Conn.Scp(path.Join(source, "bin", i.DbupCmd), filepath.ToSlash(path.Join(i.TmpDir, "bin", i.DbupCmd))); err != nil {
		return fmt.Errorf("在机器: %s 上, scp文件(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "bin", i.DbupCmd)), err)
	}
	if err := i.Conn.Scp(path.Join(source, "package", "md5"), filepath.ToSlash(path.Join(i.TmpDir, "package", "md5"))); err != nil {
		return fmt.Errorf("在机器: %s 上, scp文件(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "package", "md5")), err)
	}

	if err := i.Conn.Scp(path.Join(source, "systemd", config.MariaDBServiceTemplateFile), filepath.ToSlash(path.Join(i.TmpDir, "systemd", config.MariaDBServiceTemplateFile))); err != nil {
		return fmt.Errorf("在机器: %s 上, scp文件(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "systemd", config.MariaDBServiceTemplateFile)), err)
	}

	// TODO: 根据目标机器的操作系统类型选择包
	mariadbPackage := fmt.Sprintf("mariadb%s-%s-%s.tar.gz", config.DefaultMariaDBVersion, environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH)
	if err := i.Conn.Scp(path.Join(source, "package", "mariadb", mariadbPackage), filepath.ToSlash(path.Join(i.TmpDir, "package", "mariadb", mariadbPackage))); err != nil {
		return fmt.Errorf("在机器: %s 上, scp文件(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "package", "mariadb", mariadbPackage)), err)
	}

	if err := i.Conn.Chmod(filepath.ToSlash(path.Join(i.TmpDir, "bin", i.DbupCmd)), 0755); err != nil {
		return fmt.Errorf("在机器: %s 上, chmod目录(%s)权限失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "bin", i.DbupCmd)), err)
	}
	return nil
}

func (i *MariaDBInstance) Install(onlyCheck, addslave bool, autoincrement int) error {
	cmd := fmt.Sprintf("%s mariadb install --repluser='%s'  --replpassword='%s' --yes --port=%d  --password='%s'  --autoincrement=%d --memory=%s --dir='%s'  --owner-ip='%s' --join='%s' --system-user='%s' --system-group='%s' --resource-limit='%s' --log='%s'",
		i.DbupCmd,
		i.Inst.Option.Repluser,
		i.Inst.Option.ReplPassword,
		i.Inst.Option.Port,
		i.Inst.Option.Password,
		autoincrement,
		i.Inst.Option.Memory,
		i.Inst.Option.Dir,
		i.Host,
		i.Inst.Option.Join,
		i.Inst.Option.SystemUser,
		i.Inst.Option.SystemGroup,
		i.Inst.Option.ResourceLimit,
		filepath.ToSlash(path.Join(environment.GlobalEnv().HomePath, "dbup_mariadb_install.log")))

	if addslave {
		cmd = cmd + " --add-slave=true"
	}

	if onlyCheck {
		cmd = cmd + " --only-check"
	}

	cmd = path.Join(i.TmpDir, "bin", cmd)
	if stdout, err := i.Conn.Sudo(cmd, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd, err, stdout)
	}
	// logger.Warningf("owner ip 是: %s | Join ip 是: %s | Password 是 %s", i.Host, i.Inst.Option.Join, i.Inst.Option.Password)
	return nil
}

func (i *MariaDBInstance) InstallSlave(onlyCheck bool, addslave bool) error {
	cmd := fmt.Sprintf("%s mariadb install  --yes --repluser='%s'  --replpassword='%s'  --bakuser=%s --bakpassword='%s' --port=%d  --password='%s'  --memory=%s --dir='%s'  --owner-ip='%s' --join='%s' --system-user='%s' --system-group='%s' --resource-limit='%s' --log='%s'",
		i.DbupCmd,
		i.Inst.Option.Repluser,
		i.Inst.Option.ReplPassword,
		i.Inst.Option.Backupuser,
		i.Inst.Option.BackupPassword,
		i.Inst.Option.Port,
		i.Inst.Option.Password,
		i.Inst.Option.Memory,
		i.Inst.Option.Dir,
		i.Host,
		i.Inst.Option.Join,
		i.Inst.Option.SystemUser,
		i.Inst.Option.SystemGroup,
		i.Inst.Option.ResourceLimit,
		filepath.ToSlash(path.Join(environment.GlobalEnv().HomePath, "dbup_mariadb_install.log")))

	if addslave {
		cmd = cmd + " --add-slave=true --Backupdata=true"
	}

	if onlyCheck {
		cmd = cmd + " --only-check"
	}

	cmd = path.Join(i.TmpDir, "bin", cmd)
	if stdout, err := i.Conn.Sudo(cmd, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd, err, stdout)
	}

	return nil
}

func (i *MariaDBInstance) GaleraInstall(onlyCheck bool, onenode bool, clusteraddress string) error {
	cmd := fmt.Sprintf("%s mariadb install  --yes --port=%d  --password='%s'  --memory=%s --dir='%s'  --owner-ip='%s'  --system-user='%s' --system-group='%s' --resource-limit='%s' --log='%s'",
		i.DbupCmd,
		i.Inst.Option.Port,
		i.Inst.Option.Password,
		i.Inst.Option.Memory,
		i.Inst.Option.Dir,
		i.Host,
		i.Inst.Option.SystemUser,
		i.Inst.Option.SystemGroup,
		i.Inst.Option.ResourceLimit,
		// clusteraddress,
		filepath.ToSlash(path.Join(environment.GlobalEnv().HomePath, "dbup_mariadb_install.log")))

	if onenode {
		cmd = cmd + " --galera --onenode"
	} else {
		cmd = cmd + " --galera "
	}

	if onlyCheck {
		cmd = cmd + " --only-check"
	}

	cmd = cmd + fmt.Sprintf(" --cluster_address='%s' ", clusteraddress)

	cmd = path.Join(i.TmpDir, "bin", cmd)
	if stdout, err := i.Conn.Sudo(cmd, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd, err, stdout)
	}
	// logger.Warningf("owner ip 是: %s | Join ip 是: %s | Password 是 %s", i.Host, i.Inst.Option.Join, i.Inst.Option.Password)
	return nil
}

func (i *MariaDBInstance) UNInstall() error {
	cmd := fmt.Sprintf("%s mariadb uninstall --yes --port='%d' --dir='%s' --log='%s'",
		i.DbupCmd,
		i.Inst.Option.Port,
		i.Inst.Option.Dir,
		filepath.ToSlash(path.Join(environment.GlobalEnv().HomePath, "dbup_mariadb_uninstall.log")))

	cmd = path.Join(i.TmpDir, "bin", cmd)
	if stdout, err := i.Conn.Sudo(cmd, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd, err, stdout)
	}
	return nil
}
