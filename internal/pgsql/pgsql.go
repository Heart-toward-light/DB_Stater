/*
@Author : WuWeiJian
@Date : 2020-12-03 20:08
*/

package pgsql

import (
	"dbup/internal/environment"
	"dbup/internal/global"
	"dbup/internal/pgsql/config"
	"dbup/internal/pgsql/services"
	"fmt"
)

// Pgsql 结构体
type Pgsql struct {
}

// 所有的pgsql逻辑都在这里开始
func NewPgsql() *Pgsql {
	return &Pgsql{}
}

func (p *Pgsql) Install(pre config.Prepare, cfgFile, packageName string, onlyCheck, onlyInstall bool) error {
	inst := services.NewInstall()
	return inst.Run(pre, cfgFile, packageName, onlyCheck, onlyInstall)
}

func (p *Pgsql) InstallSlave(pre config.Prepare, master string) error {
	inst := services.NewInstall()
	return inst.RunSlave(pre, master)
}

func (p *Pgsql) AddSlave(ssho global.SSHConfig, pre config.Prepare, master string) error {
	inst := services.NewPGManager()
	return inst.AddSlave(ssho, pre, master)
}

func (p *Pgsql) UNInstall(uninst *services.UNInstall) error {
	return uninst.Uninstall()
}

func (p *Pgsql) PGautoUNInstall(uninst *services.UNInstall) error {
	return uninst.PGautofaileoverUninstall()
}

func (p *Pgsql) Backup(backup *services.Backup) error {
	return backup.Run()
}

func (p *Pgsql) BackupTables(backup *services.BackupTables, tables, list string) error {
	return backup.Run(tables, list)
}

func (p *Pgsql) BackupTask(action string, task *services.BackupTask) error {
	if action == "run" {
		return task.Run()
	}

	task.TaskNameFormat = fmt.Sprintf("%s-%d-%s", config.BackupTaskNamePrefix, task.Backup.Port, task.TaskName)
	switch environment.GlobalEnv().GOOS + "_" + action {
	case "windows_list":
		return task.WindowsList()
	case "windows_add":
		return task.WindowsAdd()
	case "windows_del":
		return task.WindowsDel()
	case "linux_list":
		return task.LinuxList()
	case "linux_add":
		return task.LinuxAdd()
	case "linux_del":
		return task.LinuxDel()
	default:
		return fmt.Errorf("不支持的操作系统或操作类型: %s", environment.GlobalEnv().GOOS)
	}
}

func (p *Pgsql) Deploy(c string) error {
	d := services.NewDeploy()
	return d.Run(c)
}

func (p *Pgsql) RemoveDeploy(c string, yes bool) error {
	d := services.NewDeploy()
	return d.RemoveDeploy(c, yes)
}

func (p *Pgsql) UserCreate(m *services.PGManager) error {
	if err := m.InitConn(); err != nil {
		return err
	}
	defer m.Conn.DB.Close()

	return m.UserCreate()
}

func (p *Pgsql) UserGrant(m *services.PGManager) error {
	if err := m.InitConn(); err != nil {
		return err
	}
	defer m.Conn.DB.Close()
	return m.UserGrant()
	// return m.AutofailoverChangehba()
}

func (p *Pgsql) UserGrantPGdata(m *services.PGManager) error {
	if err := m.InitConn(); err != nil {
		return err
	}
	defer m.Conn.DB.Close()
	// return m.UserGrant()
	return m.AutofailoverGrantuser()
}

func (p *Pgsql) DatabaseCreate(m *services.PGManager) error {
	if err := m.InitConn(); err != nil {
		return err
	}
	defer m.Conn.DB.Close()

	return m.DatabaseCreate()
}

func (p *Pgsql) CheckSlaves(m *services.PGManager, s string) error {
	if err := m.InitConn(); err != nil {
		return err
	}
	defer m.Conn.DB.Close()

	return m.CheckSlaves(s)
}

func (p *Pgsql) CheckSelect(m *services.PGManager) error {
	if err := m.InitConn(); err != nil {
		return err
	}
	defer m.Conn.DB.Close()

	return m.CheckSelect()
}

func (p *Pgsql) PGPoolInstall(param config.PgPoolParameter, cfgFile string, onlyCheck bool) error {
	d := services.NewPgPoolInstall()
	return d.Run(param, cfgFile, "", onlyCheck)
}

func (p *Pgsql) PGPoolUNInstall(s *services.PGPoolUNInstall) error {
	return s.Uninstall()
}

func (p *Pgsql) PGPoolClusterDeploy(c string) error {
	d := services.NewPGPoolClusterDeploy()
	return d.Run(c)
}

func (p *Pgsql) RemovePGPoolClusterDeploy(c string, yes bool) error {
	d := services.NewPGPoolClusterDeploy()
	return d.RemovePGPoolClusterDeploy(c, yes)
}

// func (p *Pgsql) PGAutoFailoverDeploy(c string, n bool) error {
// 	d := services.NewPGAutoFailoverDeploy()
// 	return d.Run(c, n)
// }

func (p *Pgsql) MonitorInstall(m config.PGAutoFailoverMonitor, onlyCheck bool) error {
	inst := services.NewPghaInstall()
	return inst.MonitorRun(m, onlyCheck)
}

func (p *Pgsql) PGdataInstall(d config.PGAutoFailoverPGNode, onlyCheck, onlyflushpass bool, cfgFile string) error {
	inst := services.NewPghaInstall()
	return inst.PGdataRun(d, onlyCheck, onlyflushpass, cfgFile)
}

func (p *Pgsql) MHADeploy(c string, n, y bool) error {
	d := services.NewPGAutoFailoverDeploy()
	return d.Run(c, n, y)
}

func (p *Pgsql) MHARemoveDeploy(c string, yes bool) error {
	d := services.NewPGAutoFailoverDeploy()
	return d.RemoveDeploy(c, yes)
}

func (p *Pgsql) PrimaryInstall(pre *config.RepmgrGlobal, port int, onlyCheck bool) error {
	r := services.NewRepmgrInstall(pre)
	return r.PrimaryRun(port, onlyCheck)
}

func (p *Pgsql) StandbyInstall(pre *config.RepmgrGlobal, ip string, port int) error {
	r := services.NewRepmgrInstall(pre)
	return r.StandbyRun(ip, port)
}

func (p *Pgsql) StartOldPrimaryNode(pre *config.RepmgrGlobal, pgport int) error {
	r := services.NewRepmgrInstall(pre)
	return r.StartPrimaryFailNode(pgport)
}

func (p *Pgsql) StartStandbyNode(pre *config.RepmgrGlobal, pgport int) error {
	r := services.NewRepmgrInstall(pre)
	return r.StartstandbyFailNode(pgport)
}
