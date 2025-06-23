/*
@Author : WuWeiJian
@Date : 2021-05-10 16:12
*/

package config

// 一些默认配置
const (
	DefaultMongoDBPort        = 27017
	DefaultMongoSPort         = 29000
	DefaultMongoDBSystemUser  = "mongod"
	DefaultMongoDBSystemGroup = "mongod"
	DefaultMongoDBDataDir     = "data"
	DefaultMongoDBConfigDir   = "config"
	DefaultMongoDBConfigFile  = "mongod.conf"
	DefaultMongoSConfigFile   = "mongos.conf"
	DefaultMongoDBLogDir      = "logs"
	DefaultMongoDBLogFile     = "mongod.log"
	DefaultMongoSLogFile      = "mongos.log"
	DefaultMongoDBBinDir      = "bin"
	DefaultMongoDBLibDir      = "lib"
	DefaultMongoDBBinFile     = "mongod"
	DefaultMongoSBinFile      = "mongos"
	DefaultMongoDBBaseDir     = "/opt/mongodb%d"
	DefaultMongoDBReplSet     = "md%d-%s"
	// DefaultMongoDBIpv6         = "false"
	DefaultMongoDBBindIP       = "0.0.0.0"
	DefaultMongoDBIpv6BindIP   = "0.0.0.0,::"
	MongoDBServiceTemplateFile = "mongodb.service.template"

	MongodbURISpecialChar = "[:/+@?&=]"
)

const (
	ServiceFileName       = "mongodb%d.service"
	Kinds                 = "mongodb"
	MongoDBPrimary        = "PRIMARY"
	MongoDBSecondary      = "SECONDARY"
	MongoDBArbiter        = "ARBITER"
	PackageFile           = "mongodb%s-%s-%s.tar.gz"
	DefaultMongoDBVersion = "4.2.21"
)

const (
	DeployTmpDir = "/tmp/tmpmongodb"
)

const (
	Mongos              = "mongos"
	MongoConfig         = "config"
	MongoShards         = "Shard"
	Mongoclusterinstall = "install"
	Mongoclusterdelete  = "delete"
)
