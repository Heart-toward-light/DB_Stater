package service

import (
	"dbup/internal/mariadb/config"
	"dbup/internal/utils/command"
	"dbup/internal/utils/logger"
	"fmt"
)

type Backup struct {
	BackupCmd  string
	BackupFile string
	Host       string
	Port       int
	Username   string
	Password   string
}

func NewBackup() *Backup {
	return &Backup{}
}

func (b *Backup) Validator() error {
	logger.Infof("验证参数\n")
	if b.BackupFile == "" {
		return fmt.Errorf("请指定备份文件名")
	}
	if b.Host == "" {
		b.Host = config.DefaultMariaDBlocalhost
	}
	return nil
}

func (b *Backup) Run() error {
	if err := b.Validator(); err != nil {
		return err
	}

	logger.Infof("备份开始\n")

	cmd := fmt.Sprintf("%s  --host='%s' --port=%d --user='%s' --password='%s'  --all-databases  --single-transaction  --triggers --routines  --events  > '%s'", b.BackupCmd, b.Host, b.Port, b.Username, b.Password, b.BackupFile)
	l := command.Local{Timeout: 259200}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("执行 mariadb 备份失败: %v, 标准错误输出: %s", err, stderr)
	}

	logger.Infof("备份完成\n")
	return nil
}
