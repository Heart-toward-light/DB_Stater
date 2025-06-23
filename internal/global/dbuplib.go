/*
@Author : WuWeiJian
@Date : 2021-04-01 18:53
*/

package global

import (
	"dbup/internal/environment"
	"dbup/internal/utils"
	"dbup/internal/utils/command"
	"fmt"
	"os"
	"path/filepath"
)

func InstallDbuplib() error {
	packageFullName := filepath.Join(environment.GlobalEnv().ProgramPath, PackagePath, DbuplibName, fmt.Sprintf(DbuplibPackageName, environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH))
	if err := CheckPackage(environment.GlobalEnv().ProgramPath, packageFullName, DbuplibName); err != nil {
		return err
	}
	if !utils.IsExists(DbuplibPath + "/dbuplib") {
		if err := utils.UntarGz(packageFullName, DbuplibPath); err != nil {
			return err
		}
	}

	if utils.IsExists(LdConfigFile) {
		command.CopyFile(LdConfigFile)
	}
	f, err := os.Create(LdConfigFile)
	defer f.Close()
	if err != nil {
		return err
	}
	if _, err = f.Write([]byte(DbuplibPath + "/dbuplib")); err != nil {
		return err
	}

	cmd := "ldconfig"
	l := command.Local{}
	if _, stderr, err := l.Run(cmd); err != nil {
		return fmt.Errorf("ldconfig失败: %v, 标准错误输出: %s", err, stderr)
	}
	return nil
}
