/*
@Author : WuWeiJian
@Date : 2021-07-21 11:04
*/

package cmd

import (
	"dbup/internal/redis"
	"dbup/internal/redis/config"
	"dbup/internal/redis/services"
	"fmt"
	"github.com/spf13/cobra"
)

// dbup redis backup-tasks
func redisBackupTaskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup-task",
		Short: "redis 备份任务管理",
	}
	// 装载命令
	cmd.AddCommand(
		redisBackupTaskListCmd(),
		redisBackupTaskAddCmd(),
		redisBackupTaskDelCmd(),
		redisBackupTaskRunCmd(),
	)
	return cmd
}

// dbup redis backup-task list
func redisBackupTaskListCmd() *cobra.Command {
	var task = services.NewBackupTask()
	cmd := &cobra.Command{
		Use:   "list",
		Short: "redis 备份任务列表",
		RunE: func(cmd *cobra.Command, args []string) error {
			pg := redis.NewRedis()
			return pg.BackupTask("list", task)
		},
	}
	return cmd
}

// dbup redis backup-task add
func redisBackupTaskAddCmd() *cobra.Command {
	var task = services.NewBackupTask()
	cmd := &cobra.Command{
		Use:   "add",
		Short: "redis 运行定时任务",
		RunE: func(cmd *cobra.Command, args []string) error {
			pg := redis.NewRedis()
			return pg.BackupTask("add", task)
		},
	}
	cmd.Flags().StringVarP(&task.Backup.Password, "password", "p", "", "密码")
	cmd.Flags().StringVarP(&task.Backup.Host, "host", "H", "127.0.0.1", "redis 地址")
	cmd.Flags().IntVarP(&task.Backup.Port, "port", "P", 5432, "redis 数据库监听端口")
	cmd.Flags().StringVarP(&task.Backup.BackupCmd, "command", "c", "redis-cli", "redis 备份命令")
	cmd.Flags().StringVarP(&task.BackupDir, "backupdir", "d", "", "redis 备份基目录")
	cmd.Flags().IntVarP(&task.Expire, "expire", "e", 0, "备份过期天数")
	cmd.Flags().StringVarP(&task.TaskName, "taskname", "n", config.BackupTaskDefaultTaskName, "任务名称")
	cmd.Flags().StringVarP(&task.TaskTime, "tasktime", "t", config.BackupTaskDefaultTaskTime, "任务每天开始时间")
	return cmd
}

// dbup redis backup-task del
func redisBackupTaskDelCmd() *cobra.Command {
	var task = services.NewBackupTask()
	cmd := &cobra.Command{
		Use:   "del",
		Short: "redis 删除备份任务",
		RunE: func(cmd *cobra.Command, args []string) error {
			if task.TaskName == "" {
				return fmt.Errorf("请输入要删除的任务名称\n")
			}
			pg := redis.NewRedis()
			return pg.BackupTask("del", task)
		},
	}
	cmd.Flags().StringVarP(&task.TaskName, "taskname", "n", "", "任务名称")
	cmd.Flags().IntVarP(&task.Backup.Port, "port", "P", 5432, "redis 数据库监听端口")
	return cmd
}

// dbup redis backup-task run
func redisBackupTaskRunCmd() *cobra.Command {
	var task = services.NewBackupTask()
	cmd := &cobra.Command{
		Use:   "run",
		Short: "redis 运行定时任务",
		RunE: func(cmd *cobra.Command, args []string) error {
			pg := redis.NewRedis()
			return pg.BackupTask("run", task)
		},
	}
	cmd.Flags().StringVarP(&task.Backup.Password, "password", "p", "", "密码")
	cmd.Flags().StringVarP(&task.Backup.Host, "host", "H", "127.0.0.1", "redis 地址")
	cmd.Flags().IntVarP(&task.Backup.Port, "port", "P", 5432, "redis 数据库监听端口")
	cmd.Flags().StringVarP(&task.Backup.BackupCmd, "command", "c", "redis-cli", "redis 备份命令")
	cmd.Flags().StringVarP(&task.BackupDir, "backupdir", "d", "", "redis 备份基目录")
	cmd.Flags().IntVarP(&task.Expire, "expire", "e", 0, "备份过期天数")
	return cmd
}
