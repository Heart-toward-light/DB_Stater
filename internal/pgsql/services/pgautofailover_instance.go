package services

import (
	"dbup/internal/environment"
	"dbup/internal/pgsql/config"
	"dbup/internal/utils/command"
	"fmt"
	"path"
	"path/filepath"
)

type AutoInstance struct {
	DbupCmd string
	Host    string
	NodeID  int
	TmpDir  string
	Inst    *PghaInstall
	Conn    *command.Connection
}

func NewMonitorInstance(tmp, host, user, password string, port int, mon config.PGAutoFailoverMonitor, nodeID int) (*AutoInstance, error) {
	conn, err := command.NewConnection(host, user, password, port, 30)
	if err != nil {
		return nil, fmt.Errorf("在机器: %s 上, 建立ssh连接失败: %v", host, err)
	}

	inst := NewPghaInstall()
	inst.HandleArgs(config.PGMonitor)

	// cmd := fmt.Sprintf("dbup_%s_%s", environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH)
	cmd := "dbup"
	return &AutoInstance{DbupCmd: cmd, Host: host, NodeID: nodeID, TmpDir: tmp, Inst: inst, Conn: conn}, nil
}

func NewPGdataInstance(tmp, host, user, password string, port int, pgdata config.PGAutoFailoverPGNode, nodeID int) (*AutoInstance, error) {
	conn, err := command.NewConnection(host, user, password, port, 30)
	if err != nil {
		return nil, fmt.Errorf("在机器: %s 上, 建立ssh连接失败: %v", host, err)
	}
	inst := NewPghaInstall()
	if err := inst.HandlePrepareArgs(pgdata, ""); err != nil {
		return nil, fmt.Errorf("在机器: %s 上, 初始化install失败: %v", host, err)
	}
	inst.HandleArgs(config.PGNode)

	// cmd := fmt.Sprintf("dbup_%s_%s", environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH)
	cmd := "dbup"
	return &AutoInstance{DbupCmd: cmd, Host: host, NodeID: nodeID, TmpDir: tmp, Inst: inst, Conn: conn}, nil
}

func NewMonitorInstanceUseKeyFile(tmp, host, user, keyfile string, port int, mon config.PGAutoFailoverMonitor, nodeID int) (*AutoInstance, error) {
	conn, err := command.NewConnectionUseKeyFile(host, user, keyfile, port, 30)
	if err != nil {
		return nil, fmt.Errorf("在机器: %s 上, 建立ssh连接失败: %v", host, err)
	}

	inst := NewPghaInstall()
	inst.HandleArgs(config.PGMonitor)

	cmd := "dbup"
	return &AutoInstance{DbupCmd: cmd, Host: host, NodeID: nodeID, TmpDir: tmp, Inst: inst, Conn: conn}, nil
}

func NewPGdataInstanceUseKeyFile(tmp, host, user, keyfile string, port int, pgdata config.PGAutoFailoverPGNode, nodeID int) (*AutoInstance, error) {
	conn, err := command.NewConnectionUseKeyFile(host, user, keyfile, port, 30)
	if err != nil {
		return nil, fmt.Errorf("在机器: %s 上, 建立ssh连接失败: %v", host, err)
	}
	inst := NewPghaInstall()
	if err := inst.HandlePrepareArgs(pgdata, ""); err != nil {
		return nil, fmt.Errorf("在机器: %s 上, 初始化install失败: %v", host, err)
	}
	inst.HandleArgs(config.PGNode)

	cmd := "dbup"
	return &AutoInstance{DbupCmd: cmd, Host: host, NodeID: nodeID, TmpDir: tmp, Inst: inst, Conn: conn}, nil
}

func (i *AutoInstance) CheckTmpDir() error {
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

func (i *AutoInstance) DropTmpDir() error {
	cmd := fmt.Sprintf("cd %s; rm -rf *", filepath.ToSlash(i.TmpDir))
	if stdout, err := i.Conn.Run(cmd); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd, err, stdout)
	}
	return nil
}

func (i *AutoInstance) Scp(source string) error {
	if err := i.Conn.MkdirAll(filepath.ToSlash(path.Join(i.TmpDir, "bin"))); err != nil {
		return fmt.Errorf("在机器: %s 上, 创建目录(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "bin")), err)
	}

	if err := i.Conn.MkdirAll(filepath.ToSlash(path.Join(i.TmpDir, "systemd"))); err != nil {
		return fmt.Errorf("在机器: %s 上, 创建目录(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "systemd")), err)
	}

	if err := i.Conn.MkdirAll(filepath.ToSlash(path.Join(i.TmpDir, "package/pgsql"))); err != nil {
		return fmt.Errorf("在机器: %s 上, 创建目录(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "pgsql")), err)
	}

	if err := i.Conn.Scp(path.Join(source, "bin", i.DbupCmd), filepath.ToSlash(path.Join(i.TmpDir, "bin", i.DbupCmd))); err != nil {
		return fmt.Errorf("在机器: %s 上, scp文件(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "bin", i.DbupCmd)), err)
	}

	if err := i.Conn.Scp(path.Join(source, "package", "md5"), filepath.ToSlash(path.Join(i.TmpDir, "package", "md5"))); err != nil {
		return fmt.Errorf("在机器: %s 上, scp文件(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "package", "md5")), err)
	}

	if err := i.Conn.Scp(path.Join(source, "systemd", config.PGHAServiceTemplateFile), filepath.ToSlash(path.Join(i.TmpDir, "systemd", config.PGHAServiceTemplateFile))); err != nil {
		return fmt.Errorf("在机器: %s 上, scp文件(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "systemd", config.PGHAServiceTemplateFile)), err)
	}

	// TODO: 根据目标机器的操作系统类型选择包
	pgsqlPackage := fmt.Sprintf("pgsql%s_%s_%s.tar.gz", config.DefaultPGVersion, environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH)
	if err := i.Conn.Scp(path.Join(source, "package", "pgsql", pgsqlPackage), filepath.ToSlash(path.Join(i.TmpDir, "package", "pgsql", pgsqlPackage))); err != nil {
		return fmt.Errorf("在机器: %s 上, scp文件(%s)失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "package", "pgsql", pgsqlPackage)), err)
	}

	if err := i.Conn.Chmod(filepath.ToSlash(path.Join(i.TmpDir, "bin", i.DbupCmd)), 0755); err != nil {
		return fmt.Errorf("在机器: %s 上, chmod目录(%s)权限失败: %v", i.Host, filepath.ToSlash(path.Join(i.TmpDir, "bin", i.DbupCmd)), err)
	}
	return nil
}

func (i *AutoInstance) MonitorInstall(m config.PGAutoFailoverMonitor, onlyCheck bool) error {
	cmd := fmt.Sprintf("%s pgsql-mha MonitorCreate  --dir='%s' --host='%s' --port=%d  --system-user='%s' --system-group='%s' --yes --log='%s'",
		i.DbupCmd,
		m.Dir,
		m.Host,
		m.Port,
		m.SystemUser,
		m.SystemGroup,
		filepath.ToSlash(path.Join(environment.GlobalEnv().HomePath, "dbup_pg_auto_failover_install.log")))

	if onlyCheck {
		cmd = cmd + " --only-check"
	}

	cmd = path.Join(i.TmpDir, "bin", cmd)
	// logger.Warningf("Monitor 安装命令: %s\n", cmd)
	if stdout, err := i.Conn.Sudo(cmd, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd, err, stdout)
	}
	return nil
}

func (i *AutoInstance) PGdataInstall(p config.PGAutoFailoverPGNode, pghost string, onlyCheck, onenode bool) error {
	cmd := fmt.Sprintf("%s pgsql-mha PGdataCreate --yes  --monitor-host='%s'  --monitor-port=%d  --allnode='%s' --port=%d --host='%s' --admin-password='%s' --admin-password-expire-at='%s' --username='%s'  --password='%s' --memory-size='%s' --dir='%s' --bind-ip='%s' --address='%s' --libraries='%s'  --resource-limit='%s'  --system-user='%s' --system-group='%s' --log='%s'",
		i.DbupCmd,
		p.Mhost,
		p.Mport,
		p.AllNode,
		p.Port,
		pghost,
		p.AdminPassword,
		p.AdminPasswordExpireAt,
		p.Username,
		p.Password,
		p.MemorySize,
		p.Dir,
		p.BindIP,
		p.Address,
		p.Libraries,
		p.ResourceLimit,
		p.SystemUser,
		p.SystemGroup,
		filepath.ToSlash(path.Join(environment.GlobalEnv().HomePath, "dbup_pg_auto_failover_install.log")))

	if onlyCheck {
		cmd = cmd + " --only-check"
	}

	if onenode {
		cmd = cmd + " --onenode"
	}

	cmd = path.Join(i.TmpDir, "bin", cmd)
	// logger.Warningf("Node 安装命令: %s\n", cmd)
	if stdout, err := i.Conn.Sudo(cmd, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd, err, stdout)
	}
	return nil
}

func (i *AutoInstance) PGdataFlusPass(p config.PGAutoFailoverPGNode, pghost string) error {
	cmd := fmt.Sprintf("%s pgsql-mha PGdataCreate --yes --only-flushpass  --allnode='%s'  --system-user='%s' --system-group='%s' --log='%s'",
		i.DbupCmd,
		p.AllNode,
		p.SystemUser,
		p.SystemGroup,
		filepath.ToSlash(path.Join(environment.GlobalEnv().HomePath, "dbup_pg_auto_failover_install.log")))

	cmd = path.Join(i.TmpDir, "bin", cmd)
	if stdout, err := i.Conn.Sudo(cmd, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd, err, stdout)
	}
	return nil
}

func (i *AutoInstance) UNInstall(port int, dir, role, systemuser string) error {
	cmd := fmt.Sprintf("%s pgsql-mha uninstall --yes --port='%d' --dir='%s' --system-user='%s' --log='%s' ",
		i.DbupCmd,
		port,
		dir,
		systemuser,
		filepath.ToSlash(path.Join(environment.GlobalEnv().HomePath, "dbup_pg_auto_failover_uninstall.log")))

	switch role {
	case config.PGMonitor:
		cmd = cmd + " -r monitor"
	case config.PGNode:
		cmd = cmd + " -r pgdata"
	}

	cmd = path.Join(i.TmpDir, "bin", cmd)

	if stdout, err := i.Conn.Sudo(cmd, "", ""); err != nil {
		return fmt.Errorf("在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s", i.Host, cmd, err, stdout)
	}
	return nil
}
