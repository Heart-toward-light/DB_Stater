package services

import (
	"dbup/internal/environment"
	"dbup/internal/global"
	"dbup/internal/pgsql/config"
	"dbup/internal/utils"
	"dbup/internal/utils/command"
	"dbup/internal/utils/logger"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type UPgrade struct {
	Dir         string
	Port        int
	OldVersion  string
	Servicename string
	// EmoloyDir   string
	Yes bool
}

func NewUPgrade() *UPgrade {
	return &UPgrade{}
}

func (u *UPgrade) Validator() error {
	u.Servicename = fmt.Sprintf(config.ServiceFileName, u.Port)
	serviceFileFullName := filepath.Join(global.ServicePath, u.Servicename)
	pgsqlfile := filepath.Join(u.Dir, "server/bin", config.ServerFileName)
	logger.Infof("验证参数\n")
	if u.Dir == "" {
		return fmt.Errorf("请指定安装主目录路径")
	} else {
		if !command.IsExists(pgsqlfile) {
			return fmt.Errorf("安装主目录下未发现 %s 执行文件", pgsqlfile)
		}
	}

	if u.Port == 0 {
		return fmt.Errorf("请指定要升级得pgsql实例端口")
	}

	if !command.IsExists(serviceFileFullName) {
		return fmt.Errorf("实例的 service 启停文件 %s 不存在", u.Servicename)
	}

	// 验证版本
	if err := u.VersionComparison(); err != nil {
		return err
	}

	if !utils.PortInUse(u.Port) {
		return fmt.Errorf("端口 %d 服务未启用", u.Port)
	}

	return nil
}

// 验证版本信息
func (u *UPgrade) VersionComparison() error {

	var err error
	u.OldVersion, err = command.PGsqlVersion(u.Dir)
	if err != nil {
		return err
	}

	Bigversion := strings.Split(u.OldVersion, ".")[0]
	if Bigversion != config.DefaultPGVersion {
		return fmt.Errorf("升级只支持同一个大版本 Postgresql %s 中进行小版本迭代升级", config.DefaultPGVersion)
	}

	result := command.CompareVersion(u.OldVersion, config.DefaultPGinfoVersion)
	switch result {
	case 1:
		return fmt.Errorf(" postgresql 老版本 %s 不能大于新版本 %s ", u.OldVersion, config.DefaultPGinfoVersion)
	case 0:
		return fmt.Errorf(" postgresql 老版本 %s 不能等于新版本 %s ", u.OldVersion, config.DefaultPGinfoVersion)
	}

	// logger.Infof("检测当前版本: %s\n", u.OldVersion)
	return nil
}

func (u *UPgrade) Run() error {
	if err := u.Validator(); err != nil {
		return err
	}

	if !u.Yes {
		var yes string
		logger.Warningf("升级版本需要重启本地 pgsql 实例,端口:%d\n", u.Port)
		logger.Warningf("是否确认重启进行升级[y|n]:")
		if _, err := fmt.Scanln(&yes); err != nil {
			return err
		}
		if strings.ToUpper(yes) != "Y" && strings.ToUpper(yes) != "YES" {
			os.Exit(0)
		}
	}

	logger.Infof("开始升级\n")
	if err := u.UPgradeDB(); err != nil {
		// 	u.RemoveTmp()
		return err
	}
	logger.Successf("升级完成\n")

	return nil
}

func (u *UPgrade) InitDefaultpkg() error {
	logger.Infof("开始检查默认升级版本 %s 的升级包\n", config.DefaultPGinfoVersion)
	PackageFullName := filepath.Join(environment.GlobalEnv().ProgramPath, global.PackagePath, config.Kinds, fmt.Sprintf(config.PackageFile, config.DefaultPGVersion, environment.GlobalEnv().GOOS, environment.GlobalEnv().GOARCH))

	// 解压安装包
	logger.Infof("解压升级包: %s 到 %s \n", PackageFullName, u.Dir)
	if err := utils.UntarGz(PackageFullName, u.Dir); err != nil {
		return err
	}
	// }

	// 检查依赖
	serverFileFullName := filepath.Join(u.Dir, "server/bin/", config.ServerFileName)
	if missLibs, err := global.Checkldd(serverFileFullName); err != nil {
		return err
	} else {
		if len(missLibs) != 0 {
			LibList := []string{"libssl.so.10", "libcrypto.so.10", "libtinfo.so.5", "libncurses.so.5"}
			SySLibs := []string{"/lib64", "/lib"}
			for _, missLib := range missLibs {
				re := regexp.MustCompile(`\s+`)
				result := re.ReplaceAllString(missLib.Info, "")
				Libname := strings.Split(result, "=")[0]
				for _, s := range LibList {
					if strings.Contains(s, Libname) {
						logger.Warningf("安装出现缺失的Lib文件 %s , 开始进行自动补齐\n", Libname)
						Libfullname := filepath.Join(u.Dir, "server/lib/newlib", Libname)
						if !utils.IsExists(Libfullname) {
							return fmt.Errorf("当前安装包不包含lib文件: %s", Libfullname)
						}
						for _, syslibpath := range SySLibs {
							syslibfullname := filepath.Join(syslibpath, Libname)
							if !utils.IsExists(syslibfullname) {
								if err := command.CopyFileDir(Libfullname, syslibpath); err != nil {
									return err
								}
								if err := os.Chmod(syslibfullname, 0755); err != nil {
									return err
								}
							}
						}
					}
				}
			}
		}
	}
	return nil
}

func (u *UPgrade) UPgradeDB() error {
	logger.Infof("开始关停老版本实例\n")
	if err := command.SystemCtl(u.Servicename, "stop"); err != nil {
		return err
	}

	sourcedir := filepath.Join(u.Dir, "server")
	if err := command.MoveFile(sourcedir); err != nil {
		return fmt.Errorf("修改依赖路径 %s 失败: %s", sourcedir, err)
	}

	if err := u.InitDefaultpkg(); err != nil {
		return err
	}

	sourcedata := filepath.Join(u.Dir, "data")
	l := command.Local{Timeout: 259200}
	if user, group, err := command.GetUserInfo(sourcedata); err != nil {
		return err
	} else {
		chowncmd := fmt.Sprintf("chown -R %s:%s %s", user, group, u.Dir)
		if _, stderr, err := l.Run(chowncmd); err != nil {
			return fmt.Errorf("执行修改路径所属权限失败: %v, 标准错误输出: %s", err, stderr)
		}
	}

	logger.Infof("开始启动实例\n")
	if err := command.SystemCtl(u.Servicename, "start"); err != nil {
		return err
	}
	// logger.Infof("开始替换老版本实例依赖文件\n")
	// emoloyDir := filepath.Join(u.EmoloyDir, "server")
	// if err := command.MoveFile(sourcedir); err != nil {
	// 	return fmt.Errorf("修改依赖路径 %s 失败: %s", sourcedir, err)
	// }

	// if err := command.MoveDir(emoloyDir, sourcedir); err != nil {
	// 	return err
	// }
	// if err := command.MoveDir(emoloyDir, serviceFileFullName); err != nil {
	// 	return err
	// }

	return nil
}
