// Created by LiuSainan on 2021-11-24 11:48:55

package services

import (
	"dbup/internal/environment"
	"dbup/internal/redis/config"
	"dbup/internal/utils/logger"
	"path"
	"path/filepath"
)

// 安装redis
type RedisManager struct {
}

func NewRedisManager() *RedisManager {
	return &RedisManager{}
}

func (d *RedisManager) AddSlaveNode(o config.RedisClusterAddNodeOption) error {

	logger.Infof("验证参数\n")
	if err := o.ValidatorHost(); err != nil {
		return err
	}

	o.Parameter.InitPortDir()

	if o.SSHConfig.Password == "" && o.SSHConfig.KeyFile == "" {
		o.SSHConfig.KeyFile = filepath.Join(environment.GlobalEnv().HomePath, ".ssh", "id_rsa")
	}

	var node *Instance
	var err error
	if o.SSHConfig.Password != "" {
		node, err = NewInstance(o.TmpDir,
			o.Host,
			o.SSHConfig.Username,
			o.SSHConfig.Password,
			o.SSHConfig.Port,
			o.Parameter)
		if err != nil {
			return err
		}
	} else {
		node, err = NewInstanceUseKeyFile(o.TmpDir,
			o.Host,
			o.SSHConfig.Username,
			o.SSHConfig.KeyFile,
			o.SSHConfig.Port,
			o.Parameter)
		if err != nil {
			return err
		}
	}

	if err := node.CheckTmpDir(); err != nil {
		return err
	}

	defer node.DropTmpDir()

	logger.Infof("将安装包复制到目标机器\n")
	source := path.Join(environment.GlobalEnv().ProgramPath, "..")
	if err := node.Scp(source); err != nil {
		return err
	}

	if err := node.Install(false, true, o.IPV6); err != nil {
		return err
	}

	logger.Infof("开始安装redis实例\n")
	if err := node.Install(false, false, o.IPV6); err != nil {
		logger.Warningf("安装失败, 开始回滚\n")
		// _ = node.UNInstall()
		return err
	}

	logger.Infof("新从库节点: %s:%d 添加成功\n", o.Host, o.Parameter.Port)
	logger.Infof("主从同步正常, 请自行观察数据同步是否完成\n")

	return nil
}
