/*
@Author : WuWeiJian
@Date : 2021-01-06 10:31
*/

package config

// 一些默认配置
const (
	DefaultRedisCfgFile = "redis_install.ini"

	DefaultRedisPort       = 6379
	DefaultRedisDir        = "/opt/redis"
	DefaultRedisPassLength = 16

	DefaultRedisVersion = "6"

	DefaultRedisSystemUser  = "redis"
	DefaultRedisSystemGroup = "redis"

	RedisServiceTemplateFile = "redis.service.template"
)

// 正则规则
const (
	RegexpUsername     = "^[a-z_][a-z0-9_]{1,62}$" // 用户名规则, 2到63位小写字母,数字,下划线; 不能数字开头
	RegexpMemorySuffix = "[MGmg][Bb]{0,1}$"        // 内存后缀
	ReplBacklogSizeMB  = 512
)

const (
	Kinds           = "redis"
	PackageFile     = "redis%s_%s_%s.tar.gz"
	ServerDir       = "server"
	ServerFileName  = "redis-server"
	ClientFileName  = "redis-cli"
	DataDir         = "data"
	LogsDir         = "logs"
	ConfFileName    = "redis.conf"
	ServiceFileName = "redis%d.service"
)

// Deploy 集群模式默认配置
const (
	DeployTmpDir = "/tmp/tmpredis"
)

// redis backup 计划任务
const (
	BackupTaskLinuxCronFile    = "/var/spool/cron/root"
	BackupTaskNamePrefix       = "DbupRedisBackupTask"
	BackupTaskDefaultTaskName  = "redis_backup"
	BackupTaskDefaultTaskTime  = "02:00"
	BackTaskSysPrivilegesLevel = "HIGHEST"
	RegexpTime                 = "([01]\\d|2[0-3]):([0-5]\\d)"
)

// redis cluster deploy 集群模式默认配置
const (
	RedisClusterDeployTmpDir = "/tmp/tmpredisclusterdeploy"
)
