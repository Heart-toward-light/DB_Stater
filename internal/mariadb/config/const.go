// Created by LiuSainan on 2022-06-09 11:52:40

package config

// 一些默认配置
const (
	DefaultMariaDBPort           = 3306
	DefaultGalerabaseport        = 4567
	DefaultMariaDBSystemUser     = "mariadb"
	DefaultMariaDBSystemGroup    = "mariadb"
	DefaultMariaDBDataDir        = "data"
	DefaultMariaDBConfigDir      = "config"
	DefaultMariaDBConfigFile     = "my.cnf"
	DefaultMariaDBLogDir         = "logs"
	DefaultMariaDBLogFile        = "mariadb_error.log"
	DefaultMariaDBBinDir         = "bin"
	DefaultMariaDBBinFile        = "mariadbd"
	DefaultMariaDBBaseDir        = "/opt/mariadb%d"
	DefaultMariaDBPassLength     = 16
	DefaultMariaDBBakPassword    = "EJ10#6s4Y#oxKLhX"
	DefaultMariaDBRootPassword   = "e7ac7db829fb77df"
	DefaultMariaDBReplPassword   = "330nGj3uH6!3dMld"
	DefaultMariaDBGaleraUser     = "mariabackup"
	DefaultMariaDBGaleraPassword = "Bz01vXW!4!!oOF3O"
	DefaultMariaDBlocalhost      = "127.0.0.1"
	DefaultMariaDBtxisolation    = "READ-COMMITTED"
	MariaDBServiceTemplateFile   = "mariadb.service.template"
	MongodbURISpecialChar        = "[:/+@?&=]"

	MariaDBReplicationUser       = "dbuprepl"
	MariaDBReplicationPrivileges = "REPLICATION SLAVE, REPLICATION CLIENT, SLAVE MONITOR"

	DefaultMariaDBUPgradeBinFile = "mariadb-upgrade"
	MariaDBBackupUser            = "dbupbak"
	MariaDBBackupPrivileges      = "ALL PRIVILEGES"
)

const (
	ServiceFileName       = "mariadb%d.service"
	Kinds                 = "mariadb"
	PackageFile           = "mariadb%s-%s-%s.tar.gz"
	DefaultMariaDBVersion = "10.11.8"
	MariaDBModeMS         = "MS"
	MariaDBModeMM         = "MM"
	MariaDBMasterRole     = "master"
	MariaDBSlaveRole      = "slave"
)

// 正则规则
const (
	RegexpUsername     = "^[a-z_][a-z0-9_]{1,62}$" // 用户名规则, 2到63位小写字母,数字,下划线; 不能数字开头
	RegexpMemorySuffix = "[MGmg][Bb]{0,1}$"        // 内存后缀
)

const (
	DeployTmpDir = "/tmp/tmpmariadb"
)
