// Created by LiuSainan on 2022-02-18 10:57:17

package s3ceph

import "time"

type S3Bucket struct {
	Name         string
	CreationDate time.Time
}

type S3Object struct {
	Key          string
	LastModified time.Time
	Size         int64
	StorageClass string
}
