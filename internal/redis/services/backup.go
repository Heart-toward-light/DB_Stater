/*
@Author : WuWeiJian
@Date : 2021-07-20 11:34
*/

package services

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
	Password   string
}

func NewBackup() *Backup {
	return &Backup{}
}

func (b *Backup) Validator() error {
	logger.Infof("验证参数\n")
	if b.BackupFile == "" {
		return fmt.Errorf("请指定备份目录")
	}
	return nil
}

func (b *Backup) Run() error {
	if err := b.Validator(); err != nil {
		return err
	}

	logger.Infof("备份开始\n")

	cmd := fmt.Sprintf("%s -h %s -p %d -a %s --rdb %s", b.BackupCmd, b.Host, b.Port, b.Password, b.BackupFile)
	l := command.Local{Timeout: 259200}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("执行redis备份失败: %v, 标准错误输出: %s", err, stderr)
	}

	logger.Infof("备份完成\n")
	return nil
}
