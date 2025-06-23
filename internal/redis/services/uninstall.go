/*
@Author : WuWeiJian
@Date : 2021-04-13 15:04
*/

package services

import (
	"dbup/internal/global"
	"dbup/internal/redis/config"
	"dbup/internal/utils"
	"dbup/internal/utils/command"
	"dbup/internal/utils/logger"
	"fmt"
	"path/filepath"
)

// 卸载redis的总控制逻辑
type UNInstall struct {
	Port     int
	BasePath string
}

func (i UNInstall) Uninstall() error {
	servicePath := global.ServicePath
	serviceFileName := fmt.Sprintf(config.ServiceFileName, i.Port)
	serviceFileFullName := filepath.Join(servicePath, serviceFileName)
	logger.Warningf("停止进程, 并删除启动文件: %s\n", serviceFileFullName)
	if serviceFileFullName != "" && utils.IsExists(serviceFileFullName) {
		if err := command.SystemCtl(serviceFileName, "stop"); err != nil {
			return fmt.Errorf("停止redis失败: %s\n", err)
		} else {
			logger.Warningf("停止redis成功\n")
		}
		if err := command.MoveFile(serviceFileFullName); err != nil {
			return fmt.Errorf("删除启动文件失败: %s\n", err)
		} else {
			logger.Warningf("删除启动文件成功\n")
		}
	}

	if err := command.SystemdReload(); err != nil {
		return err
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
