/*
@Author : WuWeiJian
@Date : 2020-12-25 16:10
*/

package cmd

import (
	"dbup/internal/pgsql"
	"dbup/internal/pgsql/config"
	"dbup/internal/pgsql/services"
	"fmt"

	"github.com/spf13/cobra"
)

// dbup pgsql backup-tasks
func pgsqlBackupTaskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup-task",
		Short: "pgsql 备份任务管理",
	}
	// 装载命令
	cmd.AddCommand(
		pgsqlBackupTaskListCmd(),
		pgsqlBackupTaskAddCmd(),
		pgsqlBackupTaskDelCmd(),
		pgsqlBackupTaskRunCmd(),
	)
	return cmd
}

// dbup pgsql backup-task list
func pgsqlBackupTaskListCmd() *cobra.Command {
	var task = services.NewBackupTask()
	cmd := &cobra.Command{
		Use:   "list",
		Short: "pgsql 备份任务列表",
		RunE: func(cmd *cobra.Command, args []string) error {
			pg := pgsql.NewPgsql()
			return pg.BackupTask("list", task)
		},
	}
	return cmd
}

// dbup pgsql backup-task add
func pgsqlBackupTaskAddCmd() *cobra.Command {
	var task = services.NewBackupTask()
	cmd := &cobra.Command{
		Use:   "add",
		Short: "pgsql 运行定时任务",
		RunE: func(cmd *cobra.Command, args []string) error {
			pg := pgsql.NewPgsql()
			return pg.BackupTask("add", task)
		},
	}
	cmd.Flags().StringVarP(&task.Backup.Username, "user", "u", "pguser", "用户名")
	cmd.Flags().StringVarP(&task.Backup.Password, "password", "p", "", "密码")
	cmd.Flags().StringVarP(&task.Backup.Host, "host", "H", "127.0.0.1", "pgsql 地址")
	cmd.Flags().IntVarP(&task.Backup.Port, "port", "P", 5432, "pgsql 数据库监听端口")
	cmd.Flags().StringVarP(&task.Backup.BackupCmd, "command", "c", "pg_basebackup", "pgsql 备份命令")
	cmd.Flags().StringVarP(&task.BackupDir, "backupdir", "d", "", "pgsql 备份基目录")
	cmd.Flags().IntVarP(&task.Expire, "expire", "e", 0, "备份过期天数")
	cmd.Flags().StringVarP(&task.TaskName, "taskname", "n", config.BackupTaskDefaultTaskName, "任务名称")
	cmd.Flags().StringVarP(&task.TaskTime, "tasktime", "t", config.BackupTaskDefaultTaskTime, "任务每天开始时间")
	cmd.Flags().StringVar(&task.SysUser, "sysuser", config.BackupTaskDefaultSysUser, "操作系统用户")
	cmd.Flags().StringVar(&task.SysPassword, "syspassword", "", "操作系统用户密码")
	return cmd
}

// dbup pgsql backup-task del
func pgsqlBackupTaskDelCmd() *cobra.Command {
	var task = services.NewBackupTask()
	cmd := &cobra.Command{
		Use:   "del",
		Short: "pgsql 删除备份任务",
		RunE: func(cmd *cobra.Command, args []string) error {
			if task.TaskName == "" {
				return fmt.Errorf("请输入要删除的任务名称\n")
			}
			pg := pgsql.NewPgsql()
			return pg.BackupTask("del", task)
		},
	}
	cmd.Flags().StringVarP(&task.TaskName, "taskname", "n", "", "任务名称")
	cmd.Flags().IntVarP(&task.Backup.Port, "port", "P", 5432, "pgsql 数据库监听端口")
	return cmd
}

// dbup pgsql backup-task run
func pgsqlBackupTaskRunCmd() *cobra.Command {
	var task = services.NewBackupTask()
	cmd := &cobra.Command{
		Use:   "run",
		Short: "pgsql 运行定时任务",
		RunE: func(cmd *cobra.Command, args []string) error {
			pg := pgsql.NewPgsql()
			return pg.BackupTask("run", task)
		},
	}
	cmd.Flags().StringVarP(&task.Backup.Username, "user", "u", "pguser", "用户名")
	cmd.Flags().StringVarP(&task.Backup.Password, "password", "p", "", "密码")
	cmd.Flags().StringVarP(&task.Backup.Host, "host", "H", "127.0.0.1", "pgsql 地址")
	cmd.Flags().IntVarP(&task.Backup.Port, "port", "P", 5432, "pgsql 数据库监听端口")
	cmd.Flags().StringVarP(&task.Backup.BackupCmd, "command", "c", "pg_basebackup", "pgsql 备份命令")
	cmd.Flags().StringVarP(&task.BackupDir, "backupdir", "d", "", "pgsql 备份基目录")
	cmd.Flags().IntVarP(&task.Expire, "expire", "e", 0, "备份过期天数")
	return cmd
}
