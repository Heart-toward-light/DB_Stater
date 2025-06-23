// Created by LiuSainan on 2021-12-07 17:09:09

package backupcmd

import (
	"dbup/internal/utils/logger"

	"github.com/spf13/cobra"
)

var logFile string

var rootCmd = &cobra.Command{
	Use:   "dbup-backup",
	Short: "数据库备份工具",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logger.Errorf("%v\n", err)
		//os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&logFile, "log", "", "标准输出写入日志文件")

	// 装载子命令
	rootCmd.AddCommand(
		versionCmd(),
		pgsqlBackupCmd(),
		pgsqlBackupTablesCmd(),
		redisBackupCmd(),
		mongodbBackupCmd(),
	)
}
