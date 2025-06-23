/*
@Author : WuWeiJian
@Date : 2020-12-25 16:48
*/

package services

import (
	"dbup/internal/pgsql/dao"
	"dbup/internal/utils/command"
	"dbup/internal/utils/diskutil"
	"dbup/internal/utils/logger"
	"fmt"
	"os"
)

// pgsql 备份
type Backup struct {
	BackupCmd string
	BackupDir string
	Host      string
	Port      int
	Username  string
	Password  string
}

func NewBackup() *Backup {
	return &Backup{}
}

func (b *Backup) Validator() error {
	logger.Infof("验证参数\n")
	if b.BackupDir == "" {
		return fmt.Errorf("请指定备份目录")
	}

	// 验证备份目录大小是否大于数据大小
	conn, err := dao.NewPgConn(b.Host, b.Port, b.Username, b.Password, b.Username)
	if err != nil {
		return err
	}
	dbsize, err := conn.DBSize()
	if err != nil {
		return err
	}

	disksize, err := diskutil.GetFreeDiskByte(b.BackupDir)
	if err != nil {
		return err
	}

	var add uint64 = 5 * 1024 * 1024 * 1024

	logger.Infof("数据库大小(Byte): %d", dbsize+add)
	logger.Infof("磁盘剩余量(Byte): %d", disksize)

	if dbsize+add > disksize {
		return fmt.Errorf("备份目录剩余磁盘空间不足, 需要至少: %d 字节空间", dbsize+add)
	}
	return nil
}

func (b *Backup) Run() error {
	if err := b.Validator(); err != nil {
		return err
	}

	logger.Infof("备份开始\n")

	if err := os.Setenv("PGPASSWORD", b.Password); err != nil {
		return err
	}

	cmd := fmt.Sprintf("%s -R -Fp -Xs -v -h %s -p %d -U %s -P -D %s", b.BackupCmd, b.Host, b.Port, b.Username, b.BackupDir)
	l := command.Local{Timeout: 259200}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("执行pg备份失败: %v, 标准错误输出: %s", err, stderr)
	}

	logger.Infof("备份完成\n")
	return nil
}
