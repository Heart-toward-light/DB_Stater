/*
@Author : WuWeiJian
@Date : 2021-07-21 11:25
*/

package services

import (
	"bufio"
	"dbup/internal/environment"
	"dbup/internal/redis/config"
	"dbup/internal/utils"
	"dbup/internal/utils/command"
	"dbup/internal/utils/logger"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// 备份定时任务
type BackupTask struct {
	TaskName           string
	TaskNameFormat     string
	TaskTime           string
	SysUser            string
	SysPassword        string
	SysPrivilegesLevel string
	BackupDir          string
	Expire             int
	Backup             *Backup
}

func NewBackupTask() *BackupTask {
	return &BackupTask{
		SysPrivilegesLevel: config.BackTaskSysPrivilegesLevel,
		Backup:             NewBackup(),
	}
}

func (t *BackupTask) Run() error {
	logger.Infof("运行备份任务\n")
	tm := time.Now().Format("20060102150405")
	t.Backup.BackupFile = filepath.Join(t.BackupDir, "redis_backup_"+tm+".rdb")
	if err := t.Backup.Run(); err != nil {
		return err
	}
	return t.DropExpire()
}

func (t *BackupTask) DropExpire() error {
	logger.Infof("删除过期备份\n")

	nTime := time.Now().AddDate(0, 0, -t.Expire)
	expireTime := time.Date(nTime.Year(), nTime.Month(), nTime.Day(), 0, 0, 0, 0, nTime.Location()).Unix()

	f, err := os.Open(t.BackupDir)
	if err != nil {
		return err
	}
	defer f.Close()

	backupFiles, err := f.Readdir(-1)
	if err != nil {
		fmt.Println(err)
	}

	for _, file := range backupFiles {
		if len(file.Name()) > 13 && !file.IsDir() && file.Name()[:13] == "redis_backup_" && file.ModTime().Unix() < expireTime {
			filename := filepath.Join(t.BackupDir, file.Name())
			if err := os.Remove(filename); err != nil {
				logger.Warningf("删除过期备 %s 份失败: %s\n", filename, err)
			}
		}
	}
	return nil
}

func (t *BackupTask) AddValidator() error {
	logger.Infof("验证参数\n")
	r, _ := regexp.Compile(config.RegexpTime)
	if ok := r.MatchString(t.TaskTime); !ok {
		return fmt.Errorf("时间(%s)格式不正确, 例: 凌晨2点6分执行 ( 02:06 )", t.TaskTime)
	}
	return nil
}

func (t *BackupTask) LinuxList() error {
	logger.Infof("列出定时任务列表\n")

	// 检查任务是否存在
	if !utils.IsExists(config.BackupTaskLinuxCronFile) {
		return nil
	}

	file, err := os.Open(config.BackupTaskLinuxCronFile)
	if err != nil {
		return fmt.Errorf("打开计划任务文件失败: %v", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "#"+config.BackupTaskNamePrefix) {
			cron := strings.Trim(line, " ")
			time := strings.SplitN(cron, " ", 3)
			tn := strings.Split(cron, "#")
			tName := strings.SplitN(tn[len(tn)-1], "-", 3)
			if len(time) < 2 {
				return fmt.Errorf("获取备份时间异常\n")
			}
			if len(tName) < 3 {
				return fmt.Errorf("获取备份任务名称异常\n")
			}
			fmt.Printf("备份任务名: %s, 每天备份时间: %s:%s, 备份端口号: %s\n", tName[2], time[1], time[0], tName[1])
		}
	}
	return nil
}

func (t *BackupTask) LinuxAdd() error {
	if err := t.AddValidator(); err != nil {
		return err
	}
	HM := strings.Split(t.TaskTime, ":")

	logger.Infof("添加定时任务: %s\n", t.TaskNameFormat)
	// 检查任务是否存在
	if utils.IsExists(config.BackupTaskLinuxCronFile) {
		file, err := os.Open(config.BackupTaskLinuxCronFile)
		if err != nil {
			return fmt.Errorf("打开计划任务文件失败: %v", err)
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "#"+t.TaskNameFormat) {
				return fmt.Errorf("任务名称已经存在: %s", t.TaskName)
			}
		}
	}

	// 将任务写入定时文件
	if err := command.CopyFile(config.BackupTaskLinuxCronFile); err != nil {
		return err
	}
	file, err := os.OpenFile(config.BackupTaskLinuxCronFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("打开计划任务文件失败: %v", err)
	}
	defer file.Close()

	line := fmt.Sprintf("%s %s * * * %s redis backup-task run --command='%s' --host=%s --port=%d --password='%s' --backupdir='%s' --expire=%d #%s\n", HM[1], HM[0], environment.GlobalEnv().Program, t.Backup.BackupCmd, t.Backup.Host, t.Backup.Port, t.Backup.Password, t.BackupDir, t.Expire, t.TaskNameFormat)
	write := bufio.NewWriter(file)
	if _, err := write.WriteString(line); err != nil {
		return err
	}
	if err := write.Flush(); err != nil {
		return err
	}

	//cmd := fmt.Sprintf("echo '%s %s * * * %s pgsql backup-task run --port=%d --command='%s' --user='%s' --password='%s' --backupdir='%s' --expire=%d #DbupPGSQLBackupTask-%d-%s' >> /var/spool/cron/root", HM[1], HM[0], environment.GlobalEnv().Program, t.Backup.Port, t.Backup.BackupCmd, t.Backup.Username, t.Backup.Password, t.Backup.BackupFile, t.Expire, t.Backup.Port, t.TaskName)
	//l := command.Local{}
	//if _, stderr, err := l.Run(cmd); err != nil {
	//	return fmt.Errorf("添加备份任务失败: %v, 标准错误输出: %s", err, stderr)
	//}
	logger.Infof("设置备份任务成功\n")
	return nil
}

func (t *BackupTask) LinuxDel() error {
	logger.Infof("删除计划任务: %s \n", t.TaskNameFormat)

	// 备份文件
	if utils.IsExists(config.BackupTaskLinuxCronFile) {
		if err := command.CopyFile(config.BackupTaskLinuxCronFile); err != nil {
			return err
		}
	}

	var lines []string

	// 只读方式打开文件(读取)
	file, err := os.Open(config.BackupTaskLinuxCronFile)
	if err != nil {
		return fmt.Errorf("打开计划任务文件失败: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "#"+t.TaskNameFormat) {
			continue
		}
		lines = append(lines, line)
	}

	// 打开文件(写入)
	f, err := os.Create(config.BackupTaskLinuxCronFile)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	for _, cron := range lines {
		if _, err := fmt.Fprintln(w, cron); err != nil {
			return err
		}
	}
	if err := w.Flush(); err != nil {
		return err
	}
	logger.Successf("删除成功\n")
	return nil
}
