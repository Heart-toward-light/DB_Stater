/*
@Author : WuWeiJian
@Date : 2021-05-11 11:23
*/

package dao

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoClient struct {
	Host     string
	Port     int
	User     string
	Pass     string
	Database string
	URL      string
	Conn     *mongo.Client
}

func NewMongoClient(host string, port int, user string, password string, dbname string) (*MongoClient, error) {
	var err error
	client := &MongoClient{
		Host:     host,
		Port:     port,
		User:     user,
		Pass:     password,
		Database: dbname,
	}
	switch user {
	case "":
		client.URL = fmt.Sprintf("mongodb://%s:%d/?authSource=%s&connectTimeoutMS=3000&socketTimeoutMS=3000&serverSelectionTimeoutMS=3000&connect=direct", client.Host, client.Port, client.Database)
	default:
		client.URL = fmt.Sprintf("mongodb://%s:%s@%s:%d/?authSource=%s&connectTimeoutMS=3000&socketTimeoutMS=3000&serverSelectionTimeoutMS=3000&connect=direct", client.User, client.Pass, client.Host, client.Port, client.Database)
	}
	//logger.Warningf("添加日志 - 打印连接串: %s\n", client.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if client.Conn, err = mongo.Connect(ctx, options.Client().ApplyURI(client.URL)); err != nil {
		return client, fmt.Errorf("连接数据库 %s 失败: %v", client.URL, err)
	}
	return client, nil
}

// 运行command命令
func (m *MongoClient) RunCommand(dbname string, cmd bson.D) (bson.M, error) {
	//opts := options.RunCmd().SetReadPreference(readpref.Primary())
	var result bson.M
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := m.Conn.Database(dbname).RunCommand(ctx, cmd).Decode(&result); err != nil {
		return result, err
	}
	return result, nil
}

// GetReplSetName 获取副本集名称, 好像没用了。用 DBisMaster 也可以获取到
func (m *MongoClient) GetReplSetName() (string, error) {
	var result bson.M
	var err error
	cmd := bson.D{{"replSetGetStatus", 1}}
	if result, err = m.RunCommand("admin", cmd); err != nil {
		return "", fmt.Errorf("获取副本集名称失败: %v", err)
	}
	if _, ok := result["set"]; ok {
		return result["set"].(string), nil
	}
	return "", fmt.Errorf("获取副本集名称失败: %v", err)
}

// GetReplSetName 获取副本集名称, 好像没用了。用 DBisMaster 也可以获取到
func (m *MongoClient) GetReplStatus() (bson.M, error) {
	cmd := bson.D{{"replSetGetStatus", 1}}
	return m.RunCommand("admin", cmd)
}

// GetPrimaryIP 获取副本集名, 主库IP等信息
func (m *MongoClient) DBisMaster() (bson.M, error) {
	var result bson.M
	var err error
	cmd := bson.D{{"isMaster", 1}}
	if result, err = m.RunCommand("admin", cmd); err != nil {
		return result, fmt.Errorf("执行db.isMaster()失败: %v", err)
	}
	if result["ok"].(float64) != 1 {
		return result, fmt.Errorf("执行db.isMaster()失败\n")
	}
	return result, nil
}

// 初始化副本集
func (m *MongoClient) ReplSetInitiate() error {
	var result bson.M
	var err error
	cmd := bson.D{{"replSetInitiate", ""}}

	if result, err = m.RunCommand("admin", cmd); err != nil {
		//json, err1 := bson.MarshalExtJSON(result, true, true)
		//logger.Warningf("添加日志 - 打印初始化副本集转json错误: %v\n", err1)
		//logger.Warningf("添加日志 - 打印初始化副本集结果: %s\n", string(json))
		return fmt.Errorf("执行rs.initiate()失败: %v", err)
	}
	//json, err1 := bson.MarshalExtJSON(result, true, true)
	//logger.Warningf("添加日志 - 打印初始化副本集转json错误: %v\n", err1)
	//logger.Warningf("添加日志 - 打印初始化副本集结果: %s\n", string(json))
	if result["ok"].(float64) != 1 {
		return fmt.Errorf("执行rs.initiate()失败\n")
	}
	return nil
}

// GetPrimaryIP 获取副本集名, 主库IP等信息
func (m *MongoClient) GetReplConfig() (bson.M, error) {
	var result bson.M
	var err error
	cmd := bson.D{{"replSetGetConfig", 1}}
	if result, err = m.RunCommand("admin", cmd); err != nil {
		return result, fmt.Errorf("执行rs.conf()失败: %v", err)
	}
	if result["ok"].(float64) != 1 {
		return result, fmt.Errorf("执行rs.conf()失败\n")
	}
	return result["config"].(bson.M), nil
}

// 刷新副本集配置
func (m *MongoClient) ReplReConfig(config bson.M) error {
	var result bson.M
	var err error
	cmd := bson.D{{"replSetReconfig", config}}
	if result, err = m.RunCommand("admin", cmd); err != nil {
		return fmt.Errorf("执行rs.reconfig()失败: %v", err)
	}
	//json, err1 := bson.MarshalExtJSON(result, true, true)
	//logger.Warningf("添加日志 - 打印重置配置转json错误: %v\n", err1)
	//logger.Warningf("添加日志 - 打印重置配置结果: %s\n", string(json))
	if result["ok"].(float64) != 1 {
		return fmt.Errorf("执行rs.reconfig()失败\n")
	}
	return nil
}

// CreateUser 创建mongodb用户
func (m *MongoClient) CreateUser(username, password, dbname string) error {
	if username == "" || dbname == "" || password == "" {
		return fmt.Errorf("用户名,密码,库名都不能为空\n")
	}
	cmd := bson.D{{"createUser", username}, {"pwd", password}, {"roles", bson.A{"root"}}}
	//logger.Warningf("添加日志 - 打印创建用户的命令信息: %s\n", cmd.Map())
	if _, err := m.RunCommand(dbname, cmd); err != nil {
		//json, err1 := bson.MarshalExtJSON(create_user_info, true, true)
		//logger.Warningf("添加日志 - 打印创建用户转json错误: %v\n", err1)
		//logger.Warningf("添加日志 - 打印创建用户结果: %s\n", string(json))
		return fmt.Errorf("为db: %s 创建用户: %s 失败: %v", dbname, username, err)
	}
	return nil
}

// RSAdd 增加从节点
func (m *MongoClient) RSAdd(ip string, port int) error {
	return nil
}

// Mongos 增加分片Shard
func (m *MongoClient) ShardingAdd(shardDB string) error {
	var result bson.M
	var err error
	cmd := bson.D{{Key: "addShard", Value: shardDB}}

	result, err = m.RunCommand("admin", cmd)
	if err != nil {
		return fmt.Errorf("执行sh.addShard()失败: %v", err)
	}

	if result["ok"].(float64) != 1 {
		return fmt.Errorf("执行sh.addShard()失败\n")
	}
	return nil
}

// Mongos 查看分片列表
func (m *MongoClient) ShardingList() (bson.M, error) {
	var result bson.M
	var err error

	cmd := bson.D{{Key: "listShards", Value: 1}}
	if result, err = m.RunCommand("admin", cmd); err != nil {
		return result, fmt.Errorf("执行listShards失败了: %v", err)
	}
	if result["ok"].(float64) != 1 {
		return result, fmt.Errorf("执行listShards失败\n")
	}
	return result, nil
}
