/*
@Author : WuWeiJian
@Date : 2020-12-02 17:42
*/

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// const Version = "1.0.15"
var _version string

func versionCmd() *cobra.Command {
	// 定义二级命令: version
	var cmd = &cobra.Command{
		Use:   "version",
		Short: "dbup 的版本",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("dbup version %s\n", _version)
		},
	}

	return cmd
}
