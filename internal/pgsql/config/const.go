/*
@Author : WuWeiJian
@Date : 2020-12-03 14:29
*/

package config

// 一些默认配置
const (
	DefaultPGCfgFile = "pgsql_install.ini"

	DefaultPGUser       = "pguser"
	DefaultPGDBUserPriv = "DBUSER"
	DefaultPGUserPriv   = "LOGIN CREATEDB"
	DefaultPGAdminUser  = "postgres"
	DefaultPGAdminPass  = "hh123456789"
	DefaultPGPort       = 5432
	DefaultPGDir        = "/opt/pgsql"
	DefaultPGBindIP     = "*"
	DefaultPGAddress    = "0.0.0.0/0"
	DefaultPGPassLength = 16

	DefaultPGVersion     = "12"
	DefaultPGinfoVersion = "12.20"
	DefaultPGSocketPath  = "/tmp/"

	PostgresServiceTemplateFile = "postgres.service.template"
	PGHAServiceTemplateFile     = "pgautofailover.service.template"
)

// 内置隐藏用户
const (
	DefaultPGHideUser = "pgadmin"
	DefaultPGHidePass = "yyjhidehyqlsnytzjlzzh"
	DefaultPGHidePriv = "LOGIN REPLICATION CREATEDB SUPERUSER"
	// repmgr用户
	DefaultPGRepmgrUser = "repmgr"
)

// 正则规则
const (
	RegexpUsername     = "^[a-z_][a-z0-9_]{1,62}$" // 用户名规则, 2到63位小写字母,数字,下划线; 不能数字开头
	RegexpMemorySuffix = "[MGmg][Bb]{0,1}$"        // 内存后缀
	RegexpSpecialChar  = "[:/+@?&=]"               // 密码规则
)

const (
	Kinds             = "pgsql"
	PackageFile       = "pgsql%s_%s_%s.tar.gz"
	ServerDir         = "server"
	ServerFileName    = "postgres"
	ServerProcessName = "pg_ctl"
	DataDir           = "data"
	ConfFileName      = "postgresql.conf"
	ServiceFileName   = "postgres%d.service"
	PgHbaFileName     = "pg_hba.conf"
)

// 其他参数
const (
	PasswordFile = ".pgsql.pw.file"
	PassHBAFile  = ".pgpass"
	InitDBCmd    = "initdb"
	PsqlCmd      = "psql"
)

// pgsql backup 计划任务
const (
	BackupTaskLinuxCronFile    = "/var/spool/cron/root"
	BackupTaskNamePrefix       = "DbupPGSQLBackupTask"
	BackupTaskDefaultSysUser   = "Administrator"
	BackupTaskDefaultTaskName  = "pg_backup"
	BackupTaskDefaultTaskTime  = "02:00"
	BackTaskSysPrivilegesLevel = "HIGHEST"
	RegexpTime                 = "([01]\\d|2[0-3]):([0-5]\\d)"
)

// Deploy 集群模式默认配置
const (
	DeployTmpDir      = "/tmp/tmppgsql"
	DefaultPGReplUser = "pgreplica"
	//DefaultPGReplPass = "hh123456"
	DefaultPGReplPass = "yyjreplhyqlsnytzjlzzh"
	DefaultPGReplPriv = "LOGIN REPLICATION"
)

// PgPool 部署参数
const (
	PGPoolPCPPass             = "yyjpcphyqlsnytzjlzzh" // pcp 的pgpool用户的密码
	DeployPGPoolTmpDir        = "/tmp/tmppgpool"
	PGPoolPort                = 9999
	PGPoolPCPPort             = 9898
	PGPoolWDPort              = 9000
	PGPoolHeartPort           = 9694
	PGPOOLKinds               = "pgpool"
	DefaultPGPoolDir          = "/opt/pgpool"
	PGPOOLPackageFile         = "pgpool%s_%s_%s.tar.gz"
	PGPOOLServerDir           = "server"
	DefaultPGPOOLSocketPath   = "/tmp/"
	PGPOOLServerProcessName   = "pgpool"
	PGPOOLDataDir             = "data"
	PGPOOLConfFileName        = "pgpool.conf"
	PGPOOLServiceFileName     = "pgpool%d.service"
	PGPOOLHbaFileName         = "pool_hba.conf"
	PGPCPFileName             = "pcp.conf"
	FailOverScript            = "failover.sh"
	DefaultPGPOOLVersion      = "4.2"
	PGPoolServiceTemplateFile = "pgpool.service.template"
)

// Repmgr 相关默认参数

const (
	Repmgrprimaryname = "Primary"
	Repmgrstandbyname = "Standby"
	RepmgrInitMode    = "init_repmgr"
	RepmgrAddMode     = "add_repmgr"
)

// Pg_auto_failover 相关默认参数
const (
	DefaultPGMonitorDir = "/opt/pgmonitor"
	PGAutoFailoverCmd   = "pg_autoctl"
	PGAutoFailoverLib   = "libpq.so.5"
	PGAutoFailoverUser  = "pgfailover"
	ServiceMonitorName  = "pgmonitor%d.service"
	ServiceNodeName     = "pgdata%d.service"
	PGAuth              = "trust"
	PGMonitor           = "monitor"
	PGNode              = "node"
	PGautofaile         = "postgresql-auto-failover.conf"
	PGautomonitoruser   = "pgautofailover_monitor"
	PGFailoveruser      = "pgfailover"
	PGRepluser          = "pgautofailover_replicator"
	PGMonitorUser       = "autoctl_node"
	// 此密码已写死在 pg_auto_failover 源码中,PGautomonitorPasswd 对应账号是 pgautofailover_monitor
	PGautomonitorPasswd = "POF2Rm6hkxZSI4d"
	PGMonitorPasswd     = "POF0Rm7#hkxZSI5d"
	PGFailoverPasswd    = "POF1Rm8#hkxZSI6d"
	PGReplicaPasswd     = "POF2Rm9#hkxZSI7d"
	Config_file         = "/home/%s/.config/pg_autoctl%s/pg_autoctl.cfg"
	State_file          = "/home/%s/.local/share/pg_autoctl%s/pg_autoctl.state"
	Init_file           = "/home/%s/.local/share/pg_autoctl%s/pg_autoctl.init"
)
