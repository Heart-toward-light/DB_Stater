/*
@Author : WuWeiJian
@Date : 2020-12-02 17:42
*/

package cmd

import (
	"dbup/internal/environment"
	"dbup/internal/utils/logger"

	"github.com/spf13/cobra"
)

var logFile string

var rootCmd = &cobra.Command{
	Use:   "dbup",
	Short: "数据库管理工具",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if logFile != "" {
			logger.SetLogFile(logFile)
		}
		e, err := environment.NewEnvironment()
		if err != nil {
			return err
		}
		environment.SetGlobalEnv(e)
		return nil
	},
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
		pgsqlCmd(),
		pgsqlMHACmd(),
		redisCmd(),
		redisClusterCmd(),
		mongodbCmd(),
		prometheusCmd(),
		mariadbCmd(),
	)
}
