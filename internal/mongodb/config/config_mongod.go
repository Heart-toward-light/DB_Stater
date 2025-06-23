/*
@Author : WuWeiJian
@Date : 2021-05-10 16:11
*/

package config

import (
	"path/filepath"
	"strings"
)

type ProcessManagement struct {
	Fork        bool   `yaml:"fork"`
	PidFilePath string `yaml:"pidFilePath"`
}

type Net struct {
	Ipv6                   bool   `yaml:"ipv6"`
	BindIp                 string `yaml:"bindIp"`
	Port                   int    `yaml:"port"`
	MaxIncomingConnections int    `yaml:"maxIncomingConnections"`
	ServiceExecutor        string `yaml:"serviceExecutor"`
}

type SystemLog struct {
	Destination string `yaml:"destination"`
	Path        string `yaml:"path"`
	LogAppend   bool   `yaml:"logAppend"`
}

type Journal struct {
	Enabled          bool `yaml:"enabled"`
	CommitIntervalMs int  `yaml:"commitIntervalMs"`
}

type EngineConfig struct {
	CacheSizeGB         int    `yaml:"cacheSizeGB"`
	JournalCompressor   string `yaml:"journalCompressor"`
	DirectoryForIndexes bool   `yaml:"directoryForIndexes"`
}

type CollectionConfig struct {
	BlockCompressor string `yaml:"blockCompressor"`
}

type WiredTiger struct {
	EngineConfig     EngineConfig     `yaml:"engineConfig"`
	CollectionConfig CollectionConfig `yaml:"collectionConfig"`
}

type Storage struct {
	DbPath         string     `yaml:"dbPath"`
	Journal        Journal    `yaml:"journal"`
	DirectoryPerDB bool       `yaml:"directoryPerDB"`
	WiredTiger     WiredTiger `yaml:"wiredTiger"`
}

type OperationProfiling struct {
	Mode              string `yaml:"mode"`
	SlowOpThresholdMs int64  `yaml:"slowOpThresholdMs"`
}

type Replication struct {
	OplogSizeMB int64  `yaml:"oplogSizeMB"`
	ReplSetName string `yaml:"replSetName"`
}

type Security struct {
	KeyFile string `yaml:"keyFile"`
}

type SetParameter struct {
	EnableLocalhostAuthBypass bool `yaml:"enableLocalhostAuthBypass"`
	HonorSystemUmask          bool `yaml:"honorSystemUmask"`
}

type Free struct {
	State string `yaml:"state"`
}

type Monitoring struct {
	Free Free `yaml:"free"`
}

type Cloud struct {
	Monitoring Monitoring `yaml:"monitoring"`
}

type MongoDBConfig struct {
	ProcessManagement  ProcessManagement  `yaml:"processManagement"`
	Net                Net                `yaml:"net"`
	SystemLog          SystemLog          `yaml:"systemLog"`
	Storage            Storage            `yaml:"storage"`
	OperationProfiling OperationProfiling `yaml:"operationProfiling"`
	Replication        Replication        `yaml:"replication"`
	Security           Security           `yaml:"security"`
	SetParameter       SetParameter       `yaml:"setParameter"`
	Cloud              Cloud              `yaml:"cloud"`
}

func NewMongoDBConfig(option *MongodbOptions, replSetName string) *MongoDBConfig {

	return &MongoDBConfig{
		ProcessManagement: ProcessManagement{
			Fork:        true,
			PidFilePath: filepath.Join(option.Dir, "mongod.pid"),
		},
		Net: Net{
			Ipv6:                   option.Ipv6,
			BindIp:                 option.BindIP,
			Port:                   option.Port,
			MaxIncomingConnections: 5000,
			ServiceExecutor:        "adaptive",
		},
		SystemLog: SystemLog{
			Destination: "file",
			Path:        filepath.Join(option.Dir, DefaultMongoDBLogDir, DefaultMongoDBLogFile),
			LogAppend:   true,
		},
		Storage: Storage{
			DbPath: filepath.Join(option.Dir, "data"),
			Journal: Journal{
				Enabled:          true,
				CommitIntervalMs: 100,
			},
			DirectoryPerDB: true,
			WiredTiger: WiredTiger{
				EngineConfig: EngineConfig{
					CacheSizeGB:         option.Memory,
					JournalCompressor:   "snappy",
					DirectoryForIndexes: true,
				},
				CollectionConfig: CollectionConfig{
					BlockCompressor: "snappy",
				},
			},
		},
		OperationProfiling: OperationProfiling{
			Mode:              "slowOp",
			SlowOpThresholdMs: 10000,
		},
		Replication: Replication{
			OplogSizeMB: 51200,
			ReplSetName: replSetName,
		},
		// Sharding: Sharding{
		// 	ClusterRole: clusterRole,
		// },
		Security: Security{
			KeyFile: filepath.Join(option.Dir, "data", "keyfile"),
		},
		SetParameter: SetParameter{
			EnableLocalhostAuthBypass: true,
			HonorSystemUmask:          true,
		},
		Cloud: Cloud{
			Monitoring: Monitoring{
				Free: Free{
					State: "off",
				},
			},
		},
	}
}

type MongoDBShardConfig struct {
	ProcessManagement  ProcessManagement  `yaml:"processManagement"`
	Net                Net                `yaml:"net"`
	SystemLog          SystemLog          `yaml:"systemLog"`
	Storage            Storage            `yaml:"storage"`
	OperationProfiling OperationProfiling `yaml:"operationProfiling"`
	Replication        Replication        `yaml:"replication"`
	Security           Security           `yaml:"security"`
	SetParameter       SetParameter       `yaml:"setParameter"`
	Cloud              Cloud              `yaml:"cloud"`
	Sharding           Sharding           `yaml:"sharding"`
}

func NewMongoDBShardConfig(option *MongodbOptions, replSetName string) *MongoDBShardConfig {

	clusterRole := "shardsvr"
	if find := strings.Contains(replSetName, "Config-"); find {
		clusterRole = "configsvr"
	}
	return &MongoDBShardConfig{
		ProcessManagement: ProcessManagement{
			Fork:        true,
			PidFilePath: filepath.Join(option.Dir, "mongod.pid"),
		},
		Net: Net{
			Ipv6:                   option.Ipv6,
			BindIp:                 option.BindIP,
			Port:                   option.Port,
			MaxIncomingConnections: 5000,
			ServiceExecutor:        "adaptive",
		},
		SystemLog: SystemLog{
			Destination: "file",
			Path:        filepath.Join(option.Dir, DefaultMongoDBLogDir, DefaultMongoDBLogFile),
			LogAppend:   true,
		},
		Storage: Storage{
			DbPath: filepath.Join(option.Dir, "data"),
			Journal: Journal{
				Enabled:          true,
				CommitIntervalMs: 100,
			},
			DirectoryPerDB: true,
			WiredTiger: WiredTiger{
				EngineConfig: EngineConfig{
					CacheSizeGB:         option.Memory,
					JournalCompressor:   "snappy",
					DirectoryForIndexes: true,
				},
				CollectionConfig: CollectionConfig{
					BlockCompressor: "snappy",
				},
			},
		},
		OperationProfiling: OperationProfiling{
			Mode:              "slowOp",
			SlowOpThresholdMs: 10000,
		},
		Replication: Replication{
			OplogSizeMB: 51200,
			ReplSetName: replSetName,
		},
		Sharding: Sharding{
			ClusterRole: clusterRole,
		},
		Security: Security{
			KeyFile: filepath.Join(option.Dir, "data", "keyfile"),
		},
		SetParameter: SetParameter{
			EnableLocalhostAuthBypass: true,
			HonorSystemUmask:          true,
		},
		Cloud: Cloud{
			Monitoring: Monitoring{
				Free: Free{
					State: "off",
				},
			},
		},
	}
}
