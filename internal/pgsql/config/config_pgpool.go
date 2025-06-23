/*
@Author : WuWeiJian
@Date : 2021-04-16 16:54
*/

package config

import (
	"dbup/internal/utils"
	"dbup/internal/utils/arrlib"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/ini.v1"
)

type PgPoolConfig struct {
	ListenAddresses                       string `ini:"listen_addresses"`
	PcpListenAddresses                    string `ini:"pcp_listen_addresses"`
	Port                                  int    `ini:"port"`
	PcpPort                               int    `ini:"pcp_port"`
	SocketDir                             string `ini:"socket_dir"`
	PcpSocketDir                          string `ini:"pcp_socket_dir"`
	BackendClusteringMode                 string `ini:"backend_clustering_mode"`
	ListenBacklogMultiplier               string `ini:"listen_backlog_multiplier"`
	SerializeAccept                       string `ini:"serialize_accept"`
	ReservedConnections                   string `ini:"reserved_connections"`
	EnablePoolHba                         string `ini:"enable_pool_hba"`
	PoolPasswd                            string `ini:"pool_passwd"`
	AuthenticationTimeout                 string `ini:"authentication_timeout"`
	AllowClearTextFrontendAuth            string `ini:"allow_clear_text_frontend_auth"`
	NumInitChildren                       string `ini:"num_init_children"`
	MaxPool                               string `ini:"max_pool"`
	ChildLifeTime                         string `ini:"child_life_time"`
	ChildMaxConnections                   string `ini:"child_max_connections"`
	ConnectionLifeTime                    string `ini:"connection_life_time"`
	ClientIdleLimit                       string `ini:"client_idle_limit"`
	Hostname0                             string `ini:"hostname0" comment:"# PGPool WatchDog 的配置信息"`
	WdPort0                               int    `ini:"wd_port0"`
	PgpoolPort0                           int    `ini:"pgpool_port0"`
	HeartbeatHostname0                    string `ini:"heartbeat_hostname0"`
	HeartbeatPort0                        int    `ini:"heartbeat_port0"`
	Hostname1                             string `ini:"hostname1"`
	WdPort1                               int    `ini:"wd_port1"`
	PgpoolPort1                           int    `ini:"pgpool_port1"`
	HeartbeatHostname1                    string `ini:"heartbeat_hostname1"`
	HeartbeatPort1                        int    `ini:"heartbeat_port1"`
	Hostname2                             string `ini:"hostname2"`
	WdPort2                               int    `ini:"wd_port2"`
	PgpoolPort2                           int    `ini:"pgpool_port2"`
	HeartbeatHostname2                    string `ini:"heartbeat_hostname2"`
	HeartbeatPort2                        int    `ini:"heartbeat_port2"`
	BackendHostname0                      string `ini:"backend_hostname0" comment:"# 配置PGSQL主库信息"`
	BackendPort0                          int    `ini:"backend_port0"`
	BackendWeight0                        int    `ini:"backend_weight0"`
	BackendDataDirectory0                 string `ini:"backend_data_directory0"`
	BackendFlag0                          string `ini:"backend_flag0"`
	BackendHostname1                      string `ini:"backend_hostname1" comment:"# 配置PGSQL从库信息"`
	BackendPort1                          int    `ini:"backend_port1"`
	BackendWeight1                        int    `ini:"backend_weight1"`
	BackendDataDirectory1                 string `ini:"backend_data_directory1"`
	BackendFlag1                          string `ini:"backend_flag1"`
	OtherPgpoolHostname0                  string `ini:"other_pgpool_hostname0" comment:"# 别的 PGPool WatchDog 信息"`
	OtherPgpoolPort0                      int    `ini:"other_pgpool_port0"`
	OtherWdPort0                          int    `ini:"other_wd_port0"`
	OtherPgpoolHostname1                  string `ini:"other_pgpool_hostname1"`
	OtherPgpoolPort1                      int    `ini:"other_pgpool_port1"`
	OtherWdPort1                          int    `ini:"other_wd_port1"`
	LogDestination                        string `ini:"log_destination"`
	LogLinePrefix                         string `ini:"log_line_prefix"`
	LogConnections                        string `ini:"log_connections"`
	LogDisconnections                     string `ini:"log_disconnections"`
	LogHostname                           string `ini:"log_hostname"`
	LogStatement                          string `ini:"log_statement"`
	LogPerNodeStatement                   string `ini:"log_per_node_statement"`
	LogClientMessages                     string `ini:"log_client_messages"`
	LogStandbyDelay                       string `ini:"log_standby_delay"`
	SyslogFacility                        string `ini:"syslog_facility"`
	SyslogIdent                           string `ini:"syslog_ident"`
	LogErrorVerbosity                     string `ini:"log_error_verbosity"`
	ClientMinMessages                     string `ini:"client_min_messages"`
	LogMinMessages                        string `ini:"log_min_messages"`
	LoggingCollector                      string `ini:"logging_collector"`
	DebugLevel                            string `ini:"debug_level"`
	LogDirectory                          string `ini:"log_directory"`
	LogFilename                           string `ini:"log_filename"`
	LogFileMode                           string `ini:"log_file_mode"`
	LogTruncateOnRotation                 string `ini:"log_truncate_on_rotation"`
	LogRotationAge                        string `ini:"log_rotation_age"`
	LogRotationSize                       string `ini:"log_rotation_size"`
	PidFileName                           string `ini:"pid_file_name"`
	Logdir                                string `ini:"logdir"`
	ConnectionCache                       string `ini:"connection_cache"`
	ResetQueryList                        string `ini:"reset_query_list"`
	ReplicateSelect                       string `ini:"replicate_select"`
	InsertLock                            string `ini:"insert_lock"`
	ReplicationStopOnMismatch             string `ini:"replication_stop_on_mismatch"`
	FailoverIfAffectedTuplesMismatch      string `ini:"failover_if_affected_tuples_mismatch"`
	LoadBalanceMode                       string `ini:"load_balance_mode"`
	IgnoreLeadingWhiteSpace               string `ini:"ignore_leading_white_space"`
	AllowSqlComments                      string `ini:"allow_sql_comments"`
	DisableLoadBalanceOnWrite             string `ini:"disable_load_balance_on_write"`
	DmlAdaptiveObjectRelationshipList     string `ini:"dml_adaptive_object_relationship_list"`
	StatementLevelLoadBalance             string `ini:"statement_level_load_balance"`
	SrCheckPeriod                         string `ini:"sr_check_period"`
	SrCheckUser                           string `ini:"sr_check_user"`
	SrCheckPassword                       string `ini:"sr_check_password"`
	SrCheckDatabase                       string `ini:"sr_check_database"`
	DelayThreshold                        string `ini:"delay_threshold"`
	FollowPrimaryCommand                  string `ini:"follow_primary_command"`
	HealthCheckPeriod                     string `ini:"health_check_period"`
	HealthCheckTimeout                    string `ini:"health_check_timeout"`
	HealthCheckUser                       string `ini:"health_check_user"`
	HealthCheckPassword                   string `ini:"health_check_password"`
	HealthCheckDatabase                   string `ini:"health_check_database"`
	HealthCheckMaxRetries                 string `ini:"health_check_max_retries"`
	HealthCheckRetryDelay                 string `ini:"health_check_retry_delay"`
	ConnectTimeout                        string `ini:"connect_timeout"`
	FailoverCommand                       string `ini:"failover_command"`
	FailoverOnBackendError                string `ini:"failover_on_backend_error"`
	DetachFalsePrimary                    string `ini:"detach_false_primary"`
	SearchPrimaryNodeTimeout              string `ini:"search_primary_node_timeout"`
	AutoFailback                          string `ini:"auto_failback"`
	AutoFailbackInterval                  string `ini:"auto_failback_interval"`
	UseWatchdog                           string `ini:"use_watchdog"`
	TrustedServers                        string `ini:"trusted_servers"`
	PingPath                              string `ini:"ping_path"`
	WdPriority                            string `ini:"wd_priority"`
	WdIpcSocketDir                        string `ini:"wd_ipc_socket_dir"`
	ClearMemqcacheOnEscalation            string `ini:"clear_memqcache_on_escalation"`
	FailoverWhenQuorumExists              string `ini:"failover_when_quorum_exists"`
	FailoverRequireConsensus              string `ini:"failover_require_consensus"`
	AllowMultipleFailoverRequestsFromNode string `ini:"allow_multiple_failover_requests_from_node"`
	EnableConsensusWithHalfVotes          string `ini:"enable_consensus_with_half_votes"`
	WdMonitoringInterfacesList            string `ini:"wd_monitoring_interfaces_list"`
	WdLifecheckMethod                     string `ini:"wd_lifecheck_method"`
	WdInterval                            string `ini:"wd_interval"`
	WdHeartbeatKeepalive                  string `ini:"wd_heartbeat_keepalive"`
	WdHeartbeatDeadtime                   string `ini:"wd_heartbeat_deadtime"`
	WdLifePoint                           string `ini:"wd_life_point"`
	WdLifecheckQuery                      string `ini:"wd_lifecheck_query"`
	WdLifecheckDbname                     string `ini:"wd_lifecheck_dbname"`
	WdLifecheckUser                       string `ini:"wd_lifecheck_user"`
	WdLifecheckPassword                   string `ini:"wd_lifecheck_password"`
	RelcacheExpire                        string `ini:"relcache_expire"`
	RelcacheSize                          string `ini:"relcache_size"`
	CheckTempTable                        string `ini:"check_temp_table"`
	CheckUnloggedTable                    string `ini:"check_unlogged_table"`
	EnableSharedRelcache                  string `ini:"enable_shared_relcache"`
	RelcacheQueryTarget                   string `ini:"relcache_query_target"`
}

func NewPgPoolConfig() *PgPoolConfig {
	return &PgPoolConfig{
		SocketDir:                             "'/tmp/'",
		PcpSocketDir:                          "'/tmp/'",
		BackendClusteringMode:                 "'streaming_replication'",
		ListenBacklogMultiplier:               "2",
		SerializeAccept:                       "off",
		ReservedConnections:                   "2",
		EnablePoolHba:                         "on",
		PoolPasswd:                            "'pool_passwd'",
		AuthenticationTimeout:                 "1min",
		AllowClearTextFrontendAuth:            "off",
		NumInitChildren:                       "64",
		MaxPool:                               "8",
		ChildLifeTime:                         "5min",
		ChildMaxConnections:                   "0",
		ConnectionLifeTime:                    "0",
		ClientIdleLimit:                       "0",
		LogDestination:                        "'stderr'",
		LogLinePrefix:                         "'%t: pid %p: '",
		LogConnections:                        "off",
		LogDisconnections:                     "off",
		LogHostname:                           "off",
		LogStatement:                          "off",
		LogPerNodeStatement:                   "off",
		LogClientMessages:                     "off",
		LogStandbyDelay:                       "'none'",
		SyslogFacility:                        "'LOCAL0'",
		SyslogIdent:                           "'pgpool'",
		LogErrorVerbosity:                     "default",
		ClientMinMessages:                     "error",
		LogMinMessages:                        "error",
		LoggingCollector:                      "on",
		DebugLevel:                            "0",
		LogFilename:                           "'pgpool-%a.log'",
		LogFileMode:                           "0600",
		LogTruncateOnRotation:                 "on",
		LogRotationAge:                        "1d",
		LogRotationSize:                       "1GB",
		Logdir:                                "'/tmp'", // TODO 有什么用？
		ConnectionCache:                       "on",
		ResetQueryList:                        "'ABORT; DISCARD ALL'",
		ReplicateSelect:                       "off",
		InsertLock:                            "on",
		ReplicationStopOnMismatch:             "off",
		FailoverIfAffectedTuplesMismatch:      "off",
		LoadBalanceMode:                       "off",
		IgnoreLeadingWhiteSpace:               "on",
		AllowSqlComments:                      "off",
		DisableLoadBalanceOnWrite:             "'transaction'",
		DmlAdaptiveObjectRelationshipList:     "''",
		StatementLevelLoadBalance:             "off",
		SrCheckPeriod:                         "10",
		DelayThreshold:                        "100000",
		FollowPrimaryCommand:                  "''",
		HealthCheckPeriod:                     "10",
		HealthCheckTimeout:                    "20",
		HealthCheckMaxRetries:                 "100",
		HealthCheckRetryDelay:                 "3",
		ConnectTimeout:                        "5000",
		FailoverOnBackendError:                "on",
		DetachFalsePrimary:                    "on",
		SearchPrimaryNodeTimeout:              "5min",
		AutoFailback:                          "on",
		AutoFailbackInterval:                  "1min",
		UseWatchdog:                           "on",
		TrustedServers:                        "''",
		PingPath:                              "'/bin'",
		WdPriority:                            "1",
		ClearMemqcacheOnEscalation:            "off",
		FailoverWhenQuorumExists:              "on",
		FailoverRequireConsensus:              "on",
		AllowMultipleFailoverRequestsFromNode: "off",
		EnableConsensusWithHalfVotes:          "on",
		WdMonitoringInterfacesList:            "''",
		WdLifecheckMethod:                     "'heartbeat'",
		WdInterval:                            "30",
		WdHeartbeatKeepalive:                  "2",
		WdHeartbeatDeadtime:                   "30",
		WdLifePoint:                           "3",
		WdLifecheckQuery:                      "'SELECT 1'",
		WdLifecheckDbname:                     "'template1'",
		RelcacheExpire:                        "0",
		RelcacheSize:                          "256",
		CheckTempTable:                        "catalog",
		CheckUnloggedTable:                    "on",
		EnableSharedRelcache:                  "on",
		RelcacheQueryTarget:                   "primary",
	}
}

// HandleConfig 调整配置
func (c *PgPoolConfig) HandleConfig(parameter *PgPoolParameter, basePath string) error {
	c.Port = parameter.Port
	c.PcpPort = parameter.PcpPort
	c.ListenAddresses = fmt.Sprintf("'%s'", parameter.BindIP)
	c.PcpListenAddresses = fmt.Sprintf("'%s'", parameter.PcpBindIP)

	if len(strings.Split(parameter.PGPoolIP, ",")) != 3 {
		return fmt.Errorf("pgpool的IP必须有三个")
	}

	c.Hostname0 = fmt.Sprintf("'%s'", strings.Split(parameter.PGPoolIP, ",")[0])
	c.WdPort0 = parameter.WDPort
	c.PgpoolPort0 = parameter.Port
	c.HeartbeatHostname0 = fmt.Sprintf("'%s'", strings.Split(parameter.PGPoolIP, ",")[0])
	c.HeartbeatPort0 = parameter.HeartPort

	c.Hostname1 = fmt.Sprintf("'%s'", strings.Split(parameter.PGPoolIP, ",")[1])
	c.WdPort1 = parameter.WDPort
	c.PgpoolPort1 = parameter.Port
	c.HeartbeatHostname1 = fmt.Sprintf("'%s'", strings.Split(parameter.PGPoolIP, ",")[1])
	c.HeartbeatPort1 = parameter.HeartPort

	c.Hostname2 = fmt.Sprintf("'%s'", strings.Split(parameter.PGPoolIP, ",")[2])
	c.WdPort2 = parameter.WDPort
	c.PgpoolPort2 = parameter.Port
	c.HeartbeatHostname2 = fmt.Sprintf("'%s'", strings.Split(parameter.PGPoolIP, ",")[2])
	c.HeartbeatPort2 = parameter.HeartPort

	h, e := os.Hostname()
	if e != nil {
		return e
	}

	localIPs, err := utils.LocalIP()
	if err != nil {
		return err
	}

	var localIP string
	var otherPGPool []string
	for _, ip := range strings.Split(parameter.PGPoolIP, ",") {
		if arrlib.InArray(ip, localIPs) || ip == h {
			localIP = ip
		} else {
			otherPGPool = append(otherPGPool, ip)
		}
	}

	if localIP == "" || len(otherPGPool) != 2 {
		return fmt.Errorf("pgpool的IP必须有三个, 而且其中一个IP必须是本机IP")
	}

	for i, ip := range otherPGPool {
		switch i {
		case 0:
			c.OtherPgpoolHostname0 = fmt.Sprintf("'%s'", ip)
			c.OtherPgpoolPort0 = parameter.Port
			c.OtherWdPort0 = parameter.WDPort
		case 1:
			c.OtherPgpoolHostname1 = fmt.Sprintf("'%s'", ip)
			c.OtherPgpoolPort1 = parameter.Port
			c.OtherWdPort1 = parameter.WDPort
		}
	}

	c.BackendHostname0 = fmt.Sprintf("'%s'", parameter.PGMaster)
	c.BackendPort0 = parameter.PGPort
	c.BackendWeight0 = 1
	c.BackendDataDirectory0 = fmt.Sprintf("'%s'", parameter.PGDir)
	c.BackendFlag0 = "'ALLOW_TO_FAILOVER'"

	c.BackendHostname1 = fmt.Sprintf("'%s'", parameter.PGSlave)
	c.BackendPort1 = parameter.PGPort
	c.BackendWeight1 = 1
	c.BackendDataDirectory1 = fmt.Sprintf("'%s'", parameter.PGDir)
	c.BackendFlag1 = "'ALLOW_TO_FAILOVER'"

	c.LogDirectory = fmt.Sprintf("'%s'", filepath.Join(basePath, "logs"))
	c.PidFileName = fmt.Sprintf("'%s'", filepath.Join(basePath, "pgpool.pid"))
	c.SrCheckUser = fmt.Sprintf("'%s'", DefaultPGReplUser)
	c.SrCheckPassword = fmt.Sprintf("'%s'", DefaultPGReplPass)
	c.SrCheckDatabase = fmt.Sprintf("'%s'", DefaultPGAdminUser)
	c.HealthCheckUser = fmt.Sprintf("'%s'", DefaultPGReplUser)
	c.HealthCheckPassword = fmt.Sprintf("'%s'", DefaultPGReplPass)
	c.HealthCheckDatabase = fmt.Sprintf("'%s'", DefaultPGAdminUser)
	c.FailoverCommand = fmt.Sprintf("'%s %s'", filepath.Join(basePath, "etc", FailOverScript), "%d %h %p %D %m %H %M %P %r %R %N %S")
	c.WdIpcSocketDir = "'/tmp'"
	c.WdLifecheckUser = fmt.Sprintf("'%s'", DefaultPGReplUser)
	c.WdLifecheckPassword = fmt.Sprintf("'%s'", DefaultPGReplPass)
	return nil
}

// SaveTo 将Prepare实例数据写入配置文件
func (c *PgPoolConfig) SaveTo(filename string) error {
	cfg := ini.Empty(ini.LoadOptions{IgnoreInlineComment: true})
	if err := ini.ReflectFrom(cfg, c); err != nil {
		return fmt.Errorf("pgpool 配置文件 映射到(%s)文件错误: %v", filename, err)
	}
	if err := cfg.SaveTo(filename); err != nil {
		return fmt.Errorf("pgpool 配置文件 保存到(%s)文件错误: %v", filename, err)
	}
	return nil
}
