// Created by LiuSainan on 2022-06-09 11:52:43

package config

import (
	"fmt"
	"path/filepath"
	"time"

	"gopkg.in/ini.v1"
)

type MariadbConfigClient struct {
	Port   int    `ini:"port"`
	Socket string `ini:"socket"`
}

type MariadbConfigMysql struct {
	MaxAllowedPacket    string `ini:"max_allowed_packet"`
	Prompt              string `ini:"prompt"`
	DefaultCharacterSet string `ini:"default_character_set"`
}

type MariadbConfigMysqldump struct {
	MaxAllowedPacket string `ini:"max_allowed_packet"`
}

type MariadbConfigMysqld struct {
	ServerID                     int64  `ini:"server_id" comment:"# global"`
	User                         string `ini:"user"`
	Port                         int    `ini:"port"`
	Socket                       string `ini:"socket"`
	PidFile                      string `ini:"pid_file"`
	SecureFilePriv               string `ini:"secure_file_priv"`
	Datadir                      string `ini:"datadir"`
	Tmpdir                       string `ini:"tmpdir"`
	InnodbTmpdir                 string `ini:"innodb_tmpdir"`
	CharacterSetServer           string `ini:"character_set_server"`
	CollationServer              string `ini:"collation_server"`
	RelayLogRecovery             string `ini:"relay_log_recovery"`
	LocalInfile                  string `ini:"local_infile"`
	ExplicitDefaultsForTimestamp string `ini:"explicit_defaults_for_timestamp"`
	MaxHeapTableSize             string `ini:"max_heap_table_size"`
	MaxConnections               string `ini:"max_connections"`
	MaxUserConnections           string `ini:"max_user_connections"`
	ThreadCacheSize              string `ini:"thread_cache_size"`
	MaxConnectErrors             string `ini:"max_connect_errors"`
	WaitTimeout                  string `ini:"wait_timeout"`
	InteractiveTimeout           string `ini:"interactive_timeout"`
	Netreadtimeout               string `ini:"net_read_timeout"`
	Netwritetimeout              string `ini:"net_write_timeout"`
	BackLog                      string `ini:"back_log"`
	SkipNameResolve              string `ini:"skip-name-resolve"`
	SkipSlaveStart               string `ini:"skip-slave-start"`
	ReadOnly                     string `ini:"read_only"`
	EventScheduler               string `ini:"event_scheduler" comment:"# superReadOnly =  #mariadb 里没有这个参数"`
	LowerCaseTableNames          string `ini:"lower_case_table_names"`
	SqlMode                      string `ini:"sql_mode" comment:"# defaultAuthenticationPlugin = "`
	GtidDomainId                 string `ini:"gtid_domain_id"`
	Autoincrement                string `ini:"auto_increment_increment"`
	Autoffset                    string `ini:"auto_increment_offset"`

	ReportHost string `ini:"report-host" comment:"# slavereport"`
	ReportPort int    `ini:"report-port"`

	LogError                string `ini:"log_error" comment:"# MySQLerrorlog&&GeneralQueryLog&&PerformanceSchema"`
	InnodbPrintAllDeadlocks string `ini:"innodb_print_all_deadlocks"`

	GeneralLogFile string `ini:"general_log_file"`
	GeneralLog     string `ini:"general_log"`

	PerformanceSchema                                    string `ini:"performance_schema"`
	PerformanceSchemaConsumerEventsStatementsHistoryLong string `ini:"performance_schema_consumer_events_statements_history_long"`

	OpenFilesLimit          string `ini:"open_files_limit" comment:"# performanceSchemaInstrument = \r\n \r\n# Tablebuffersandcaches = "`
	TableDefinitionCache    string `ini:"table_definition_cache"`
	TableOpenCache          string `ini:"table_open_cache"`
	TableOpenCacheInstances string `ini:"table_open_cache_instances"`

	MaxAllowedPacket  string `ini:"max_allowed_packet" comment:"# sessionbuffer"`
	JoinBufferSize    string `ini:"join_buffer_size"`
	SortBufferSize    string `ini:"sort_buffer_size"`
	TmpTableSize      string `ini:"tmp_table_size"`
	ReadBufferSize    string `ini:"read_buffer_size"`
	ReadRndBufferSize string `ini:"read_rnd_buffer_size"`

	LogQueriesNotUsingIndexes string `ini:"log_queries_not_using_indexes" comment:"# slowlog"`
	SlowQueryLog              string `ini:"slow_query_log"`
	SlowQueryLogFile          string `ini:"slow_query_log_file"`
	LongQueryTime             string `ini:"long_query_time"`
	MinExaminedRowLimit       string `ini:"min_examined_row_limit"`

	LogBin                  string `ini:"log_bin" comment:"# binlog"`
	RelayLog                string `ini:"relay-log"`
	BinlogFormat            string `ini:"binlog-format"`
	SyncBinlog              string `ini:"sync_binlog"`
	BinlogCacheSize         string `ini:"binlog_cache_size"`
	BinlogStmtCacheSize     string `ini:"binlog_stmt_cache_size"`
	MaxBinlogSize           string `ini:"max_binlog_size"`
	BinlogExpireLogsSeconds string `ini:"binlog_expire_logs_seconds"`
	LogSlaveUpdates         string `ini:"log-slave-updates"`
	SlaveTransactionRetries string `ini:"slave_transaction_retries"`
	SlaveParallelThreads    string `ini:"slave_parallel_threads"`

	InnodbBufferPoolSize  string `ini:"innodb_buffer_pool_size" comment:"# engineInnoDB"`
	DefaultStorageEngine  string `ini:"default_storage_engine"`
	InnodbFlushMethod     string `ini:"innodb_flush_method" comment:"# disabledStorageEngines = "`
	InnodbDataHomeDir     string `ini:"innodb_data_home_dir"`
	InnodbDataFilePath    string `ini:"innodb_data_file_path"`
	InnodbAutoincLockMode string `ini:"innodb_autoinc_lock_mode"`
	InnodbMonitorEnable   string `ini:"innodb_monitor_enable"`
	InnodbLogBufferSize   string `ini:"innodb_log_buffer_size"`
	InnodbDoublewrite     string `ini:"innodb_doublewrite"`
	InnodbStrictMode      string `ini:"innodb_strict_mode"`

	InnodbLogGroupHomeDir string `ini:"innodb_log_group_home_dir" comment:"# redolog"`
	InnodbLogFileSize     string `ini:"innodb_log_file_size" comment:"# innodb_log_files_in_group"`

	InnodbFlushLogAtTrxCommit string `ini:"innodb_flush_log_at_trx_commit" comment:"# innodbperformance"`
	InnodbFilePerTable        string `ini:"innodb_file_per_table"`

	InnodbFlushSync                string `ini:"innodb_flush_sync" comment:"# innodb_flush_sync&&innodb_io_capacity"`
	InnodbIoCapacity               string `ini:"innodb_io_capacity"`
	InnodbIoCapacityMax            string `ini:"innodb_io_capacity_max"`
	InnodbLockWaitTimeout          string `ini:"innodb_lock_wait_timeout"`
	InnodbMaxDirtyPagesPct         string `ini:"innodb_max_dirty_pages_pct"`
	InnodbDefaultRowFormat         string `ini:"innodb_default_row_format"`
	InnodbBufferPoolDumpAtShutdown string `ini:"innodb_buffer_pool_dump_at_shutdown"`
	InnodbBufferPoolLoadAtStartup  string `ini:"innodb_buffer_pool_load_at_startup"`
	InnodbBufferPoolDumpPct        string `ini:"innodb_buffer_pool_dump_pct"`
	TransactionIsolation           string `ini:"transaction_isolation"`
}

type MariaDBConfig struct {
	Client    MariadbConfigClient    `ini:"client"`
	Mysql     MariadbConfigMysql     `ini:"mysql"`
	Mysqldump MariadbConfigMysqldump `ini:"mysqldump"`
	Mysqld    MariadbConfigMysqld    `ini:"mysqld"`
}

func NewMariaDBConfig() *MariaDBConfig {
	return &MariaDBConfig{
		Mysql: MariadbConfigMysql{
			MaxAllowedPacket:    "64M",
			Prompt:              "'\\u@\\h [\\d]> '",
			DefaultCharacterSet: "utf8mb4",
		},
		Mysqldump: MariadbConfigMysqldump{
			MaxAllowedPacket: "1024M",
		},
		Mysqld: MariadbConfigMysqld{
			CharacterSetServer:           "utf8mb4",
			CollationServer:              "utf8mb4_general_ci",
			RelayLogRecovery:             "ON",
			LocalInfile:                  "ON",
			ExplicitDefaultsForTimestamp: "OFF",
			MaxHeapTableSize:             "64M",
			MaxConnections:               "5000",
			MaxUserConnections:           "5000",
			ThreadCacheSize:              "100",
			MaxConnectErrors:             "1000000",
			WaitTimeout:                  "3600",
			InteractiveTimeout:           "3600",
			Netreadtimeout:               "900",
			Netwritetimeout:              "900",
			BackLog:                      "1024",
			SkipNameResolve:              "ON",
			SkipSlaveStart:               "OFF",
			ReadOnly:                     "OFF",
			EventScheduler:               "ON",
			LowerCaseTableNames:          "1",
			SqlMode:                      "'STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION'",
			GtidDomainId:                 "0",
			Autoincrement:                "1",
			Autoffset:                    "1",
			InnodbPrintAllDeadlocks:      "ON",
			GeneralLog:                   "OFF",
			PerformanceSchema:            "ON",
			PerformanceSchemaConsumerEventsStatementsHistoryLong: "ON",
			OpenFilesLimit:                 "65535",
			TableDefinitionCache:           "1400",
			TableOpenCache:                 "2000",
			TableOpenCacheInstances:        "16",
			MaxAllowedPacket:               "64M",
			JoinBufferSize:                 "1M",
			SortBufferSize:                 "2M",
			TmpTableSize:                   "32M",
			ReadBufferSize:                 "128k",
			ReadRndBufferSize:              "256k",
			LogQueriesNotUsingIndexes:      "OFF",
			SlowQueryLog:                   "ON",
			LongQueryTime:                  "1",
			MinExaminedRowLimit:            "100",
			BinlogFormat:                   "row",
			SyncBinlog:                     "1",
			BinlogCacheSize:                "2M",
			BinlogStmtCacheSize:            "2M",
			MaxBinlogSize:                  "1G",
			BinlogExpireLogsSeconds:        "604800",
			LogSlaveUpdates:                "ON",
			SlaveTransactionRetries:        "100",
			SlaveParallelThreads:           "4",
			DefaultStorageEngine:           "InnoDB",
			InnodbFlushMethod:              "O_DIRECT",
			InnodbDataFilePath:             "ibdata1:100M:autoextend",
			InnodbAutoincLockMode:          "2",
			InnodbMonitorEnable:            "all",
			InnodbLogBufferSize:            "16M",
			InnodbDoublewrite:              "1",
			InnodbStrictMode:               "ON",
			InnodbLogFileSize:              "2G",
			InnodbFlushLogAtTrxCommit:      "1",
			InnodbFilePerTable:             "1",
			InnodbFlushSync:                "OFF",
			InnodbIoCapacity:               "1000",
			InnodbIoCapacityMax:            "2000",
			InnodbLockWaitTimeout:          "20",
			InnodbMaxDirtyPagesPct:         "90",
			InnodbDefaultRowFormat:         "DYNAMIC",
			InnodbBufferPoolDumpAtShutdown: "1",
			InnodbBufferPoolLoadAtStartup:  "1",
			InnodbBufferPoolDumpPct:        "50",
		},
	}
}

// HandleConfig 调整配置
func (c *MariaDBConfig) HandleConfig(option *MariaDBOptions) {
	c.Client.Port = option.Port
	c.Client.Socket = fmt.Sprintf("/tmp/.mariadb%d.sock", option.Port)

	c.Mysqld.ServerID = time.Now().Unix()
	c.Mysqld.User = option.SystemUser
	c.Mysqld.Port = option.Port
	c.Mysqld.Socket = fmt.Sprintf("/tmp/.mariadb%d.sock", option.Port)
	c.Mysqld.PidFile = filepath.Join(option.Dir, "data", "mariadb.pid")
	c.Mysqld.SecureFilePriv = filepath.Join(option.Dir, "loadfiles")
	c.Mysqld.Datadir = filepath.Join(option.Dir, "data")
	c.Mysqld.Tmpdir = filepath.Join(option.Dir, "tmp")
	c.Mysqld.InnodbTmpdir = filepath.Join(option.Dir, "tmp")

	c.Mysqld.ReportHost = option.OwnerIP
	c.Mysqld.ReportPort = option.Port

	c.Mysqld.LogError = filepath.Join(option.Dir, "logs", "error.log")
	c.Mysqld.GeneralLogFile = filepath.Join(option.Dir, "logs", "general.log")
	c.Mysqld.SlowQueryLogFile = filepath.Join(option.Dir, "logs", "slow.log")

	c.Mysqld.LogBin = filepath.Join(option.Dir, "data", "binlog")
	c.Mysqld.RelayLog = filepath.Join(option.Dir, "data", "relaylog")

	c.Mysqld.InnodbBufferPoolSize = option.Memory

	c.Mysqld.InnodbDataHomeDir = filepath.Join(option.Dir, "data")
	c.Mysqld.InnodbLogGroupHomeDir = filepath.Join(option.Dir, "data")
	c.Mysqld.TransactionIsolation = option.TxIsolation

	if option.AutoIncrement == 2 {
		c.Mysqld.Autoincrement = "2"
		c.Mysqld.Autoffset = "1"
	} else if option.AutoIncrement == 3 {
		c.Mysqld.Autoincrement = "2"
		c.Mysqld.Autoffset = "2"
	}
	// && option.Clustermode == MariaDBModeMS
	if option.Role == MariaDBSlaveRole && option.AddSlave {
		c.Mysqld.ReadOnly = "ON"
		c.Mysqld.EventScheduler = "OFF"
	}
}

// SaveTo 将Prepare实例数据写入配置文件
func (c *MariaDBConfig) SaveTo(filename string) error {
	cfg := ini.Empty(ini.LoadOptions{IgnoreInlineComment: true})
	if err := ini.ReflectFrom(cfg, c); err != nil {
		return fmt.Errorf("mariadb 配置文件 映射到(%s)文件错误: %v", filename, err)
	}
	if err := cfg.SaveTo(filename); err != nil {
		return fmt.Errorf("mariadb 配置文件 保存到(%s)文件错误: %v", filename, err)
	}
	return nil
}

type MariadbConfigGalera struct {
	Wsrep_on               string `ini:"wsrep_on"`
	Wsrep_provider         string `ini:"wsrep_provider"`
	Wsrep_node_name        string `ini:"wsrep_node_name"`
	Wsrep_node_address     string `ini:"wsrep_node_address"`
	Wsrep_cluster_name     string `ini:"wsrep_cluster_name"`
	Wsrep_cluster_address  string `ini:"wsrep_cluster_address"`
	Wsrep_provider_options string `ini:"wsrep_provider_options"`
	Wsrep_slave_threads    int    `ini:"wsrep_slave_threads"`
	Wsrep_sst_method       string `ini:"wsrep_sst_method"`
	Wsrep_sst_auth         string `ini:"wsrep_sst_auth"`
}

// HandleConfig 调整配置
func (c *MariaDBGaleraConfig) HandleGaleraConfig(goption *MariaDBOptions) {

	c.Client.Port = goption.Port
	c.Client.Socket = fmt.Sprintf("/tmp/.mariadb%d.sock", goption.Port)

	c.Mysqld.ServerID = time.Now().Unix()
	c.Mysqld.User = goption.SystemUser
	c.Mysqld.Port = goption.Port
	c.Mysqld.Socket = fmt.Sprintf("/tmp/.mariadb%d.sock", goption.Port)
	c.Mysqld.PidFile = filepath.Join(goption.Dir, "data", "mariadb.pid")
	c.Mysqld.SecureFilePriv = filepath.Join(goption.Dir, "loadfiles")
	c.Mysqld.Datadir = filepath.Join(goption.Dir, "data")
	c.Mysqld.Tmpdir = filepath.Join(goption.Dir, "tmp")
	c.Mysqld.InnodbTmpdir = filepath.Join(goption.Dir, "tmp")

	c.Mysqld.ReportHost = goption.OwnerIP
	c.Mysqld.ReportPort = goption.Port

	c.Mysqld.LogError = filepath.Join(goption.Dir, "logs", "error.log")
	c.Mysqld.GeneralLogFile = filepath.Join(goption.Dir, "logs", "general.log")
	c.Mysqld.SlowQueryLogFile = filepath.Join(goption.Dir, "logs", "slow.log")

	c.Mysqld.LogBin = filepath.Join(goption.Dir, "data", "binlog")
	c.Mysqld.RelayLog = filepath.Join(goption.Dir, "data", "relaylog")

	c.Mysqld.InnodbBufferPoolSize = goption.Memory

	c.Mysqld.InnodbDataHomeDir = filepath.Join(goption.Dir, "data")
	c.Mysqld.InnodbLogGroupHomeDir = filepath.Join(goption.Dir, "data")

	c.Galera.Wsrep_on = "ON"
	c.Galera.Wsrep_provider = filepath.Join(goption.Dir, "/lib/galera", "libgalera_smm.so")
	c.Galera.Wsrep_node_name = fmt.Sprintf("'%s'", goption.OwnerIP)
	c.Galera.Wsrep_node_address = fmt.Sprintf("'%s'", goption.OwnerIP)
	c.Galera.Wsrep_cluster_name = "'galera-cluster'"
	c.Galera.Wsrep_cluster_address = fmt.Sprintf("'gcomm://%s'", goption.Wsrepclusteraddress)
	c.Galera.Wsrep_provider_options = "gcache.size=1G"
	c.Galera.Wsrep_slave_threads = 8
	c.Galera.Wsrep_sst_method = DefaultMariaDBGaleraUser
	c.Galera.Wsrep_sst_auth = fmt.Sprintf("%s:%s", DefaultMariaDBGaleraUser, DefaultMariaDBGaleraPassword)

}

type MariaDBGaleraConfig struct {
	Client    MariadbConfigClient    `ini:"client"`
	Mysql     MariadbConfigMysql     `ini:"mysql"`
	Mysqldump MariadbConfigMysqldump `ini:"mysqldump"`
	Mysqld    MariadbConfigMysqld    `ini:"mysqld"`
	Galera    MariadbConfigGalera    `ini:"galera"`
}

func NewMariaDBGaleraConfig() *MariaDBGaleraConfig {
	return &MariaDBGaleraConfig{
		Mysql: MariadbConfigMysql{
			MaxAllowedPacket:    "64M",
			Prompt:              "'\\u@\\h [\\d]> '",
			DefaultCharacterSet: "utf8mb4",
		},
		Mysqldump: MariadbConfigMysqldump{
			MaxAllowedPacket: "1024M",
		},
		Mysqld: MariadbConfigMysqld{
			CharacterSetServer:           "utf8mb4",
			CollationServer:              "utf8mb4_general_ci",
			RelayLogRecovery:             "ON",
			LocalInfile:                  "ON",
			ExplicitDefaultsForTimestamp: "OFF",
			MaxHeapTableSize:             "64M",
			MaxConnections:               "5000",
			MaxUserConnections:           "5000",
			ThreadCacheSize:              "100",
			MaxConnectErrors:             "1000000",
			WaitTimeout:                  "3600",
			InteractiveTimeout:           "3600",
			Netreadtimeout:               "900",
			Netwritetimeout:              "900",
			BackLog:                      "1024",
			SkipNameResolve:              "ON",
			SkipSlaveStart:               "OFF",
			ReadOnly:                     "OFF",
			EventScheduler:               "ON",
			LowerCaseTableNames:          "1",
			SqlMode:                      "'STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION'",
			GtidDomainId:                 "0",
			InnodbPrintAllDeadlocks:      "ON",
			GeneralLog:                   "OFF",
			PerformanceSchema:            "ON",
			PerformanceSchemaConsumerEventsStatementsHistoryLong: "ON",
			OpenFilesLimit:                 "65535",
			TableDefinitionCache:           "1400",
			TableOpenCache:                 "2000",
			TableOpenCacheInstances:        "16",
			MaxAllowedPacket:               "64M",
			JoinBufferSize:                 "1M",
			SortBufferSize:                 "2M",
			TmpTableSize:                   "32M",
			ReadBufferSize:                 "128k",
			ReadRndBufferSize:              "256k",
			LogQueriesNotUsingIndexes:      "OFF",
			SlowQueryLog:                   "ON",
			LongQueryTime:                  "1",
			MinExaminedRowLimit:            "100",
			BinlogFormat:                   "row",
			SyncBinlog:                     "1",
			BinlogCacheSize:                "2M",
			BinlogStmtCacheSize:            "2M",
			MaxBinlogSize:                  "1G",
			BinlogExpireLogsSeconds:        "604800",
			LogSlaveUpdates:                "ON",
			SlaveTransactionRetries:        "100",
			DefaultStorageEngine:           "InnoDB",
			InnodbFlushMethod:              "O_DIRECT",
			InnodbDataFilePath:             "ibdata1:100M:autoextend",
			InnodbAutoincLockMode:          "2",
			InnodbMonitorEnable:            "all",
			InnodbLogBufferSize:            "16M",
			InnodbDoublewrite:              "1",
			InnodbStrictMode:               "ON",
			InnodbLogFileSize:              "2G",
			InnodbFlushLogAtTrxCommit:      "1",
			InnodbFilePerTable:             "1",
			InnodbFlushSync:                "OFF",
			InnodbIoCapacity:               "1000",
			InnodbIoCapacityMax:            "2000",
			InnodbLockWaitTimeout:          "20",
			InnodbMaxDirtyPagesPct:         "90",
			InnodbDefaultRowFormat:         "DYNAMIC",
			InnodbBufferPoolDumpAtShutdown: "1",
			InnodbBufferPoolLoadAtStartup:  "1",
			InnodbBufferPoolDumpPct:        "50",
		},
	}
}

// SaveTo 将Prepare实例数据写入 Galera 配置文件
func (c *MariaDBGaleraConfig) GaleraSaveTo(filename string) error {
	cfg := ini.Empty(ini.LoadOptions{IgnoreInlineComment: true})
	if err := ini.ReflectFrom(cfg, c); err != nil {
		return fmt.Errorf("mariadb 配置文件 映射到(%s)文件错误: %v", filename, err)
	}
	if err := cfg.SaveTo(filename); err != nil {
		return fmt.Errorf("mariadb 配置文件 保存到(%s)文件错误: %v", filename, err)
	}
	return nil
}
