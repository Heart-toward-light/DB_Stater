// Created by LiuSainan on 2021-12-06 17:41:32

package service

import (
	"dbup/internal/utils/command"
	"dbup/internal/utils/logger"
	"fmt"
)

// redis 备份
type Backup struct {
	BackupCmd  string
	BackupFile string
	Host       string
	Port       int
	Username   string
	Password   string
	AuthDB     string
}

func NewBackup() *Backup {
	return &Backup{}
}

func (b *Backup) Validator() error {
	logger.Infof("验证参数\n")
	if b.BackupFile == "" {
		return fmt.Errorf("请指定备份文件名")
	}
	return nil
}

func (b *Backup) Run() error {
	if err := b.Validator(); err != nil {
		return err
	}

	logger.Infof("备份开始\n")

	// mongodump --authenticationDatabase="admin" --host="127.0.0.1" --port=35011 --username="monitor" --password="08b5411f848a2581a41672a759c87380" --numParallelCollections=16 --gzip --archive="test.20150716.gz"
	cmd := fmt.Sprintf("%s --authenticationDatabase='%s' --host='%s' --port=%d --username='%s' --password='%s' --gzip --archive='%s'", b.BackupCmd, b.AuthDB, b.Host, b.Port, b.Username, b.Password, b.BackupFile)
	l := command.Local{Timeout: 259200}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("执行redis备份失败: %v, 标准错误输出: %s", err, stderr)
	}

	logger.Infof("备份完成\n")
	return nil
}
