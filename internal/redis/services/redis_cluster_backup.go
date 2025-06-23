// Created by LiuSainan on 2022-02-10 15:13:25

package services

import (
	"dbup/internal/global/s3ceph"
	"dbup/internal/redis/dao"
	"dbup/internal/utils"
	"dbup/internal/utils/logger"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"
)

// redis 备份
type RedisClusterBackup struct {
	BackupCmd      string
	BackupBasePath string
	BackupFullPath string
	Host           string
	Port           int
	Password       string
	Expire         int
	ExpireTime     time.Time
	EndPoint       string
	AccessKey      string
	SecretKey      string
	Bucket         string
	Mode           string
	S3BasePath     string
	S3FullPath     string
	BackupToS3     bool
}

func NewRedisClusterBackup() *RedisClusterBackup {
	return &RedisClusterBackup{}
}

type BackupInfo struct {
	Host       string
	Port       int
	BackupFile string
}

func (b *RedisClusterBackup) Validator() error {
	logger.Infof("验证参数\n")
	if b.Mode != "SkipVerify" && b.Mode != "normal" {
		return fmt.Errorf("--mode 连接方式值只能是 normal 或 SkipVerify, 默认 normal")
	}

	if b.BackupBasePath == "" {
		return fmt.Errorf("请指定备份目录; 如果是备份到S3,也需要指定本地目录临时存放备份")
	}

	if b.Expire > 1000 || b.Expire < 0 {
		return fmt.Errorf("过期参数必须大于等于0, 小于1000")
	}
	if !b.BackupToS3 {
		return nil
	}

	if b.EndPoint == "" {
		return fmt.Errorf("请指定 S3 连接地址")
	}

	if b.AccessKey == "" {
		return fmt.Errorf("请指定 S3 accesskey")
	}

	if b.SecretKey == "" {
		return fmt.Errorf("请指定 S3 secretkey")
	}

	if b.Bucket == "" {
		return fmt.Errorf("请指定 S3 bucket")
	}

	return nil
}

func (b *RedisClusterBackup) InitArgs() {

	if b.S3BasePath == "" {
		b.S3BasePath = b.BackupBasePath
	}

	b.S3BasePath = strings.Trim(b.S3BasePath, "/")

	now := time.Now()
	b.ExpireTime = now.AddDate(0, 0, -b.Expire)
	b.BackupFullPath = path.Join(b.BackupBasePath, now.Format("20060102150405"))
	b.S3FullPath = path.Join(b.S3BasePath, now.Format("20060102150405"))
}

func (b *RedisClusterBackup) Run() error {
	if err := b.Validator(); err != nil {
		return err
	}

	b.InitArgs()

	if err := b.Mkdir(); err != nil {
		return err
	}

	masters, err := b.Masters()
	if err != nil {
		return err
	}

	if err := b.Backup(masters); err != nil {
		return err
	}

	if !b.BackupToS3 {
		if b.Expire != 0 {
			fmt.Println("删除本地过期备份")
			if err := b.RemoveLocalExpired(); err != nil {
				return err
			}
		}

		return nil
	}

	if err := b.BackupToS3Action(masters); err != nil {
		return err
	}

	if err := b.RemoveLocal(b.BackupFullPath); err != nil {
		return err
	}

	if b.Expire != 0 {
		fmt.Println("删除S3过期备份")
		if err := b.RemoveFromS3Action(); err != nil {
			return err
		}
	}

	logger.Infof("备份完成\n")
	return nil
}

func (b *RedisClusterBackup) Mkdir() error {
	// 判断目录是否可用
	if utils.IsExists(b.BackupFullPath) {
		if utils.IsDir(b.BackupFullPath) {
			if emp, err := utils.IsEmpty(b.BackupFullPath); err != nil {
				return fmt.Errorf("指定的备份目录: %s, 不为空", b.BackupFullPath)
			} else if !emp {
				return fmt.Errorf("指定的备份目录: %s, 不为空", b.BackupFullPath)
			}
		} else {
			return fmt.Errorf("指定的备份目录: %s, 是一个文件", b.BackupFullPath)
		}
	} else {
		if err := os.MkdirAll(b.BackupFullPath, 0755); err != nil {
			return err
		}
	}

	return nil
}

func (b *RedisClusterBackup) Masters() (backinfo []BackupInfo, err error) {
	logger.Infof("获取所有节点")
	client, err := dao.NewRedisConn(b.Host, b.Port, b.Password)
	if err != nil {
		return backinfo, err
	}
	defer client.Conn.Close()

	nodes, err := client.ClusterNodes()
	if err != nil {
		return backinfo, err
	}

	for _, node := range nodes {
		if node.Role != "master" {
			continue
		}
		backinfo = append(backinfo, BackupInfo{
			Host:       node.Host,
			Port:       node.Port,
			BackupFile: fmt.Sprintf("%s_%d.rdb", node.Host, node.Port),
		})
	}
	return backinfo, nil
}

func (b *RedisClusterBackup) Backup(masters []BackupInfo) error {

	logger.Infof("备份所有主节点开始\n")
	for _, master := range masters {
		bk := Backup{
			BackupCmd:  b.BackupCmd,
			BackupFile: path.Join(b.BackupFullPath, master.BackupFile),
			Host:       master.Host,
			Port:       master.Port,
			Password:   b.Password,
		}
		if err := bk.Run(); err != nil {
			return err
		}
	}
	return nil
}

func (b *RedisClusterBackup) RemoveLocalExpired() error {
	fs, err := ioutil.ReadDir(b.BackupBasePath)
	if err != nil {
		return err
	}
	for _, f := range fs {
		if strings.HasPrefix(f.Name(), "2") && f.ModTime().Before(b.ExpireTime) {
			if err := b.RemoveLocal(path.Join(b.BackupBasePath, f.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *RedisClusterBackup) RemoveLocal(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("删除本地备份目录失败: %v", err)
	} else {
		logger.Warningf("删除本地备份目录成功: %s\n", path)
	}
	return nil
}

func (b *RedisClusterBackup) BackupToS3Action(masters []BackupInfo) error {
	s3c, err := s3ceph.NewS3Ceph(b.EndPoint, b.AccessKey, b.SecretKey, b.Mode)
	if err != nil {
		return err
	}

	for _, master := range masters {
		if err := s3c.Upload(b.Bucket, path.Join(b.BackupFullPath, master.BackupFile), path.Join(b.S3FullPath, master.BackupFile)); err != nil {
			return err
		}
	}

	fmt.Println("上传到S3完成")
	return nil
}

func (b *RedisClusterBackup) RemoveFromS3Action() error {
	s3c, err := s3ceph.NewS3Ceph(b.EndPoint, b.AccessKey, b.SecretKey, b.Mode)
	if err != nil {
		return err
	}

	objs, err := s3c.ListObjectFromBucket(b.Bucket, b.S3BasePath)
	if err != nil {
		return err
	}

	for _, obj := range objs {
		if strings.HasPrefix(strings.TrimPrefix(strings.TrimPrefix(obj.Key, b.S3BasePath), "/"), "2") && obj.LastModified.Before(b.ExpireTime) {
			if err := s3c.DeleteObject(b.Bucket, obj.Key); err != nil {
				return err
			}
			logger.Warningf("删除S3备份文件成功: %s\n", obj.Key)
		}
	}

	fmt.Println("删除完成")
	return nil
}
