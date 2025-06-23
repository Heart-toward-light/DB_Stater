/*
@Author : WuWeiJian
@Date : 2021-01-05 14:53
*/

package redis

import (
	"dbup/internal/environment"
	"dbup/internal/redis/config"
	"dbup/internal/redis/services"
	"fmt"
)

// Pgsql 结构体
type Redis struct {
}

// 所有的redis逻辑都在这里开始
func NewRedis() *Redis {
	return &Redis{}
}

func (r *Redis) Install(param config.Parameters, cfgFile string, onlyCheck bool) error {
	inst := services.NewInstall()
	return inst.Run(param, cfgFile, onlyCheck)
}

func (r *Redis) RedisAddSlave(o config.RedisClusterAddNodeOption) error {
	d := services.NewRedisManager()
	return d.AddSlaveNode(o)
}

func (r *Redis) UNInstall(uninst *services.UNInstall) error {
	return uninst.Uninstall()
}

func (r *Redis) Deploy(c string) error {
	d := services.NewDeploy()
	return d.Run(c)
}

func (r *Redis) RemoveDeploy(c string, yes bool) error {
	d := services.NewDeploy()
	return d.RemoveDeploy(c, yes)
}

func (r *Redis) Backup(backup *services.Backup) error {
	return backup.Run()
}

func (r *Redis) BackupTask(action string, task *services.BackupTask) error {
	if action == "run" {
		return task.Run()
	}

	task.TaskNameFormat = fmt.Sprintf("%s-%d-%s", config.BackupTaskNamePrefix, task.Backup.Port, task.TaskName)
	switch environment.GlobalEnv().GOOS + "_" + action {
	case "linux_list":
		return task.LinuxList()
	case "linux_add":
		return task.LinuxAdd()
	case "linux_del":
		return task.LinuxDel()
	default:
		return fmt.Errorf("不支持的操作系统或操作类型: %s", environment.GlobalEnv().GOOS)
	}
}

func (r *Redis) RedisClusterDeploy(c string) error {
	d := services.NewRedisClusterDeploy()
	return d.Run(c)
}

func (r *Redis) RedisClusterRemoveDeploy(c string, yes bool) error {
	d := services.NewRedisClusterDeploy()
	return d.RemoveCluster(c, yes)
}

func (r *Redis) RedisClusterAddMaster(o config.RedisClusterAddNodeOption) error {
	d := services.NewRedisClusterManager()
	return d.AddNode(o, "master")
}

func (r *Redis) RedisClusterAddSlave(o config.RedisClusterAddNodeOption) error {
	d := services.NewRedisClusterManager()
	return d.AddNode(o, "slave")
}

func (r *Redis) RedisClusterFix(command string, cluster string, password string) error {
	d := services.NewRedisClusterManager()
	return d.RedisClusterFix(command, cluster, password)
}

func (r *Redis) RedisClusterBackup(backup *services.RedisClusterBackup) error {
	return backup.Run()
}
