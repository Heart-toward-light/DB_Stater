// Created by LiuSainan on 2022-02-18 10:57:48

package s3ceph

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type S3Ceph struct {
	EndPoint  string
	AccessKey string
	SecretKey string
	Config    *aws.Config
	Session   *session.Session
}

func NewS3Ceph(endPoint, accessKey, secretKey, mode string) (*S3Ceph, error) {
	var err error
	c := &S3Ceph{
		EndPoint:  endPoint,
		AccessKey: accessKey,
		SecretKey: secretKey,
	}

	// if !strings.HasPrefix(bucket, "/") {
	// 	Bucket = "/" + bucket
	// }
	switch mode {
	case "normal":
		c.Config = &aws.Config{
			Credentials:      credentials.NewStaticCredentials(c.AccessKey, c.SecretKey, ""),
			Endpoint:         aws.String(c.EndPoint),
			Region:           aws.String("us-east-1"),
			DisableSSL:       aws.Bool(true),
			S3ForcePathStyle: aws.Bool(false), //virtual-host style方式，不要修改
		}
	case "SkipVerify":
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		httpclient := &http.Client{Transport: tr}

		c.Config = &aws.Config{
			Credentials: credentials.NewStaticCredentials(c.AccessKey, c.SecretKey, ""),
			Endpoint:    aws.String(c.EndPoint),
			Region:      aws.String("us-east-1"),
			// DisableSSL:       aws.Bool(true),
			S3ForcePathStyle: aws.Bool(false), //virtual-host style方式，不要修改
			HTTPClient:       httpclient,
		}
	}

	if c.Session, err = session.NewSession(c.Config); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *S3Ceph) CreateBuckets(bucket string) error {
	// 因为没有创建bucket权限, 所以不确定需要不需要加前缀, 暂时注释
	// if !strings.HasPrefix(bucket, "/") {
	// 	bucket = "/" + bucket
	// }

	params := &s3.CreateBucketInput{Bucket: aws.String(bucket)}

	svc := s3.New(c.Session)
	if _, err := svc.CreateBucket(params); err != nil {
		return err
	}

	// Wait until bucket is created before finishing
	// fmt.Printf("Waiting for bucket %q to be created...\n", bucket)

	if err := svc.WaitUntilBucketExists(&s3.HeadBucketInput{Bucket: aws.String(bucket)}); err != nil {
		return err
	}

	fmt.Printf("Bucket %q successfully created\n", bucket)
	return nil

}

func (c *S3Ceph) ListBuckets() (result []S3Bucket, err error) {
	svc := s3.New(c.Session)
	rst, err := svc.ListBuckets(nil)
	if err != nil {
		return result, fmt.Errorf("unable to list buckets, %v", err)
	}

	for _, b := range rst.Buckets {
		result = append(result, S3Bucket{
			Name:         aws.StringValue(b.Name),
			CreationDate: aws.TimeValue(b.CreationDate),
		})
	}

	return result, nil

}

func (c *S3Ceph) ListObjectFromBucket(bucket string, prefix string) (result []S3Object, err error) {
	if !strings.HasPrefix(bucket, "/") {
		bucket = "/" + bucket
	}

	params := &s3.ListObjectsInput{Bucket: aws.String(bucket), Prefix: aws.String(prefix)}

	svc := s3.New(c.Session)
	resp, err := svc.ListObjects(params)

	if err != nil {
		return result, err
	}

	for _, item := range resp.Contents {
		result = append(result, S3Object{
			Key:          aws.StringValue(item.Key),
			LastModified: aws.TimeValue(item.LastModified),
			Size:         aws.Int64Value(item.Size),
			StorageClass: aws.StringValue(item.StorageClass),
		})
	}

	return result, nil

}

func (c *S3Ceph) Upload(bucket string, localpath string, s3path string) error {
	if !strings.HasPrefix(bucket, "/") {
		bucket = "/" + bucket
	}

	if localpath == "" {
		return errors.New("请指定要上传的本地文件路径")
	}

	if s3path == "" {
		return errors.New("请指定要上传到S3上的存放路径")
	}

	filename := filepath.Base(localpath)
	if strings.HasSuffix(s3path, "/") {
		s3path = filepath.Join(s3path, filename)
	}

	file, err := os.Open(localpath)
	if err != nil {
		return err
	}
	defer file.Close()

	uploader := s3manager.NewUploader(c.Session)

	if _, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(s3path),
		Body:   file,
	}); err != nil {
		return err
	}
	return nil
}

func (c *S3Ceph) Download(bucket string, s3path string, localpath string) error {
	if !strings.HasPrefix(bucket, "/") {
		bucket = "/" + bucket
	}

	if s3path == "" {
		return errors.New("请指定要下载的S3上的文件路径")
	}

	if localpath == "" {
		return errors.New("请指定要下载到本地的路径")
	}

	filename := filepath.Base(s3path)
	if strings.HasSuffix(localpath, "/") {
		localpath = filepath.Join(localpath, filename)
	}

	file, err := os.Create(localpath)
	if err != nil {
		return err
	}
	defer file.Close()

	downloader := s3manager.NewDownloader(c.Session)

	if _, err := downloader.Download(file,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(s3path),
		}); err != nil {
		return err
	}

	return err
}

func (c *S3Ceph) DeleteObject(bucket string, s3path string) error {
	if !strings.HasPrefix(bucket, "/") {
		bucket = "/" + bucket
	}

	svc := s3.New(c.Session)

	if _, err := svc.DeleteObject(&s3.DeleteObjectInput{Bucket: aws.String(bucket), Key: aws.String(s3path)}); err != nil {
		return err
	}

	if err := svc.WaitUntilObjectNotExists(&s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(s3path),
	}); err != nil {
		return err
	}
	return nil
}
