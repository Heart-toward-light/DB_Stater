package config

import "path/filepath"

type Ssh_config struct {
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	KeyFile  string `yaml:"keyfile"`
	TmpDir   string `yaml:"tmp-dir"`
}

type Mongo_config struct {
	Config_memory  int    `yaml:"config_memory"`
	Shard_memory   int    `yaml:"shard_memory"`
	Username       string `yaml:"username"`
	Password       string `yaml:"password"`
	Ipv6           bool   `yaml:"ipv6"`
	Bind_ip        string `yaml:"bind-ip"`
	Resource_limit string `yaml:"resource-limit"`
	System_user    string `yaml:"system-user"`
	System_group   string `yaml:"system-group"`
}

type MongosNode struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	Dir  string `yaml:"dir"`
}

type MongoConfigNode struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	Dir  string `yaml:"dir"`
}

type MongoShardNode struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	Dir  string `yaml:"dir"`
}

type MongoShard struct {
	Shard []MongoShardNode `yaml:"shard"`
}

type Sharding struct {
	ClusterRole string `yaml:"clusterRole"`
}

// type Shardingswich struct {
// 	Sharding string
// }

type MongosSharding struct {
	ConfigDB string `yaml:"configDB"`
}

type MongosprocessManagement struct {
	Fork bool `yaml:"fork"`
}

type MongoSConfig struct {
	ProcessManagement MongosprocessManagement `yaml:"processManagement"`
	Net               Net                     `yaml:"net"`
	SystemLog         SystemLog               `yaml:"systemLog"`
	Security          Security                `yaml:"security"`
	SetParameter      SetParameter            `yaml:"setParameter"`
	Sharding          MongosSharding          `yaml:"sharding"`
}

func NewMongoSConfig(option *MongosOptions) *MongoSConfig {

	return &MongoSConfig{
		SystemLog: SystemLog{
			Destination: "file",
			Path:        filepath.Join(option.Dir, DefaultMongoDBLogDir, DefaultMongoSLogFile),
			LogAppend:   true,
		},
		Net: Net{
			Ipv6:                   option.Ipv6,
			BindIp:                 option.BindIP,
			Port:                   option.Port,
			MaxIncomingConnections: 5000,
			ServiceExecutor:        "adaptive",
		},
		ProcessManagement: MongosprocessManagement{
			Fork: true,
		},
		SetParameter: SetParameter{
			EnableLocalhostAuthBypass: true,
			HonorSystemUmask:          true,
		},
		Security: Security{
			KeyFile: filepath.Join(option.Dir, "data", "keyfile"),
		},
		Sharding: MongosSharding{
			ConfigDB: option.ConfigDB,
		},
	}

}
