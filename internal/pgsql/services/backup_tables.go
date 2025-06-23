/*
@Author : WuWeiJian
@Date : 2021-07-20 12:08
*/

package services

import (
	"dbup/internal/utils"
	"dbup/internal/utils/command"
	"dbup/internal/utils/logger"
	"fmt"
	"os"
	"strings"
)

// pgsql 备份
type BackupTables struct {
	BackupCmd  string
	BackupFile string
	Host       string
	Port       int
	Username   string
	Password   string
	Format     string
	Database   string
	Tables     []string
}

func NewBackupTables() *BackupTables {
	return &BackupTables{}
}

func (b *BackupTables) InitArgs(tables, list string) error {
	logger.Infof("初始化参数\n")

	var tbs []string

	if tables != "" {
		tbs = strings.Split(tables, ",")
	}

	if list != "" {
		ts, err := utils.ReadLineFromFile(list)
		if err != nil {
			return err
		}
		tbs = append(tbs, ts...)
	}

	for _, tb := range tbs {
		tb = strings.Trim(tb, " ")
		tb = strings.Trim(tb, "'")
		tb = strings.Trim(tb, "\"")
		if tb != "" {
			b.Tables = append(b.Tables, tb)
		}
	}

	return nil
}

func (b *BackupTables) Validator() error {
	logger.Infof("验证参数\n")

	if b.Database == "" {
		return fmt.Errorf("请指定要备份的库")
	}

	if b.BackupFile == "" {
		return fmt.Errorf("请指定备份文件")
	}

	if b.Format != "c" && b.Format != "d" && b.Format != "p" && b.Format != "t" {
		return fmt.Errorf("请指定正确的备份格式")
	}
	return nil
}

func (b *BackupTables) Run(tables, list string) error {

	if err := b.Validator(); err != nil {
		return err
	}

	if err := b.InitArgs(tables, list); err != nil {
		return err
	}

	logger.Infof("备份开始\n")

	tablesCmd := ""
	for _, table := range b.Tables {
		tablesCmd += fmt.Sprintf(" -t \"%s\"", table)
	}
	cmd := fmt.Sprintf("%s -h %s -p %d -d %s -U %s -F%s -f %s %s", b.BackupCmd, b.Host, b.Port, b.Database, b.Username, b.Format, b.BackupFile, tablesCmd)

	if err := os.Setenv("PGPASSWORD", b.Password); err != nil {
		return err
	}

	l := command.Local{Timeout: 259200}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("执行pg备份失败: %v, 标准错误输出: %s", err, stderr)
	}

	logger.Infof("备份完成\n")
	return nil
}
