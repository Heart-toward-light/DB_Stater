/*
@Author : WuWeiJian
@Date : 2021-03-15 17:41
*/

package services

import (
	"dbup/internal/global"
	"dbup/internal/pgsql/config"
	"dbup/internal/utils"
	"dbup/internal/utils/command"
	"dbup/internal/utils/logger"
	"fmt"
	"path/filepath"
)

// 卸载pgsql的总控制逻辑
type UNInstall struct {
	SystemUser string
	Port       int
	BasePath   string
	Repmgr     bool
	RepmgrRole string
	AutopgRole string
}

func (i UNInstall) Uninstall() error {
	servicePath := global.ServicePath
	serviceFileName := fmt.Sprintf(config.ServiceFileName, i.Port)
	ResourceLimitDir := fmt.Sprintf("/etc/systemd/system/%s.d", serviceFileName)
	serviceFileFullName := filepath.Join(servicePath, serviceFileName)

	logger.Warningf("停止进程, 并删除启动文件: %s\n", serviceFileFullName)
	if serviceFileFullName != "" && utils.IsExists(serviceFileFullName) {
		if err := command.SystemCtl(serviceFileName, "stop"); err != nil {
			return fmt.Errorf("停止pgsql失败: %s\n", err)
		} else {
			logger.Warningf("停止pgsql成功\n")
		}
		if err := command.MoveFile(serviceFileFullName); err != nil {
			return fmt.Errorf("删除启动文件失败: %s\n", err)
		} else {
			logger.Warningf("删除启动文件成功\n")
		}
	}

	if err := command.SystemdReload(); err != nil {
		logger.Warningf("systemctl daemon-reload 失败\n")
	}

	if utils.IsDir(ResourceLimitDir) {
		logger.Warningf("删除资源配置目录: %s\n", ResourceLimitDir)
		if err := command.MoveFile(ResourceLimitDir); err != nil {
			return fmt.Errorf("删除资源配置目录失败: %s\n", err)
		} else {
			logger.Warningf("删除资源配置目录成功\n")
		}
	}

	logger.Warningf("删除安装目录: %s\n", i.BasePath)
	if i.BasePath != "" && utils.IsDir(i.BasePath) {
		if err := command.MoveFile(i.BasePath); err != nil {
			return fmt.Errorf("删除安装目录失败: %s\n", err)
		} else {
			logger.Warningf("删除安装目录成功\n")
		}
	}
	return nil
}

func (i UNInstall) PGautofaileoverUninstall() error {
	servicePath := global.ServicePath
	var serviceFileName string
	if i.AutopgRole == "monitor" {
		serviceFileName = fmt.Sprintf(config.ServiceMonitorName, i.Port)
	} else if i.AutopgRole == "pgdata" {
		serviceFileName = fmt.Sprintf(config.ServiceNodeName, i.Port)
	} else {
		return fmt.Errorf("请指定卸载 pg_auto_failover 的角色 monitor|pgdata\n")
	}

	serviceFileFullName := filepath.Join(servicePath, serviceFileName)
	pgsqlcmd := filepath.Join(i.BasePath, "server", "bin", "pg_autoctl")
	dataPath := filepath.Join(i.BasePath, "data")

	logger.Warningf("停止进程, 并删除启动文件: %s\n", serviceFileFullName)

	if serviceFileFullName != "" && utils.IsExists(serviceFileFullName) {
		if err := command.SystemCtl(serviceFileName, "stop"); err != nil {
			return fmt.Errorf("停止pgsql失败: %s\n", err)
		} else {
			logger.Warningf("停止pgsql成功\n")
		}
		if err := command.MoveFile(serviceFileFullName); err != nil {
			return fmt.Errorf("删除启动文件失败: %s\n", err)
		} else {
			logger.Warningf("删除启动文件成功\n")
		}
	}

	if err := command.SystemdReload(); err != nil {
		logger.Warningf("systemctl daemon-reload 失败\n")
	}

	l := command.Local{}
	var ins_cmd string
	var tmp_cmd string
	if i.AutopgRole == "pgdata" {
		logger.Warningf("从集群中删除当前数据节点相关文件\n")
		ins_cmd = fmt.Sprintf("sudo -u %s %s drop node --pgdata %s --destroy ", i.SystemUser, pgsqlcmd, dataPath)
		tmp_cmd = fmt.Sprintf("sudo -u %s rm -rf /tmp/pg_autoctl%s", i.SystemUser, i.BasePath)
	} else if i.AutopgRole == "monitor" {
		logger.Warningf("删除监控节点相关文件\n")
		ins_cmd = fmt.Sprintf("sudo -u %s %s drop monitor --pgdata %s --destroy ", i.SystemUser, pgsqlcmd, dataPath)
		tmp_cmd = fmt.Sprintf("sudo -u %s rm -rf /tmp/pg_autoctl%s", i.SystemUser, i.BasePath)
	}

	// 清理数据环境sys
	l.Sudo(ins_cmd)
	l.Sudo(tmp_cmd)

	var filelist []string
	// 判断实例相关文件并移除
	// config_file := fmt.Sprintf("/home/%s/.config/pg_autoctl/%s/pg_autoctl.cfg", i.SystemUser, dataPath)
	// state_file := fmt.Sprintf("/home/%s/.local/share/pg_autoctl/%s/pg_autoctl.state", i.SystemUser, dataPath)
	// init_file := fmt.Sprintf("/home/%s/.local/share/pg_autoctl/%s/pg_autoctl.init", i.SystemUser, dataPath)

	Cfile := fmt.Sprintf(config.Config_file, i.SystemUser, dataPath)
	Sfile := fmt.Sprintf(config.State_file, i.SystemUser, dataPath)
	Ifile := fmt.Sprintf(config.Init_file, i.SystemUser, dataPath)

	filelist = append(filelist, Cfile, Sfile, Ifile)

	for _, filename := range filelist {
		if utils.IsExists(filename) {
			if err := command.MoveFile(filename); err != nil {
				return fmt.Errorf("移除多余文件失败: %s\n", err)
			}
		}
	}

	logger.Warningf("删除安装目录: %s\n", i.BasePath)
	if i.BasePath != "" && utils.IsDir(i.BasePath) {
		if err := command.MoveFile(i.BasePath); err != nil {
			return fmt.Errorf("删除安装目录失败: %s\n", err)
		} else {
			logger.Warningf("删除安装目录成功\n")
		}
	}
	return nil
}
