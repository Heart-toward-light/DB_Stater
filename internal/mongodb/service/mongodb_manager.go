// Created by LiuSainan on 2021-11-24 14:23:18

package service

import (
	"dbup/internal/environment"
	"dbup/internal/global"
	"dbup/internal/mongodb/config"
	"dbup/internal/utils/logger"
	"path"
	"path/filepath"
)

// 安装redis
type MongoDBManager struct {
}

func NewMongoDBManager() *MongoDBManager {
	return &MongoDBManager{}
}

func (d *MongoDBManager) AddSlaveNode(ssho global.SSHConfig, o config.MongodbOptions) error {

	logger.Infof("验证参数\n")
	if err := ssho.Validator(); err != nil {
		return err
	}

	if ssho.TmpDir == "" {
		ssho.TmpDir = config.DeployTmpDir
	}

	if ssho.Password == "" && ssho.KeyFile == "" {
		ssho.KeyFile = filepath.Join(environment.GlobalEnv().HomePath, ".ssh", "id_rsa")
	}

	var node *MongoDBInstance
	var err error
	if ssho.Password != "" {
		node, err = NewMongoDBInstance(ssho.TmpDir,
			ssho.Host,
			ssho.Username,
			ssho.Password,
			ssho.Port,
			o)
		if err != nil {
			return err
		}
	} else {
		node, err = NewMongoDBInstanceUseKeyFile(ssho.TmpDir,
			ssho.Host,
			ssho.Username,
			ssho.KeyFile,
			ssho.Port,
			o)
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

	if err := node.Install(true, o.Arbiter, o.NoRollback, o.Ipv6); err != nil {
		return err
	}

	logger.Infof("开始安装 MongoDB 从库实例\n")
	if err := node.Install(false, o.Arbiter, o.NoRollback, o.Ipv6); err != nil {
		logger.Warningf("安装失败\n")
		// _ = node.UNInstall()
		return err
	}

	logger.Infof("节点: %s:%d 添加成功\n", ssho.Host, o.Port)
	logger.Successf("登录命令: %s --authenticationDatabase admin -u %s -p '%s' --host %s --port %d\n", "mongo", o.Username, o.Password, ssho.Host, o.Port)
	logger.Successf("\n")
	logger.Successf("请自行检查主从数据同步进度\n")

	return nil
}
