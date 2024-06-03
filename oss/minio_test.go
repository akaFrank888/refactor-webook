package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/suite"
	"io"
	"log"
	"strconv"
	"testing"
)

type MinioTestSuite struct {
	suite.Suite
	minioClient *minio.Client
	bucketName  string
}

func (s *MinioTestSuite) SetupSuite() {
	// MinIO服务器的访问信息
	endpoint := "localhost:9000"
	accessKeyID := "QVSWvxnymDaQ2XsVIAWS"
	secretAccessKey := "syTzQgrcqQydr1aXln7astdRTYtrrme9jaGCEbF9"
	useSSL := false // 根据实际情况设置是否使用SSL，默认为true

	// 创建MinIO客户端
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalln("创建MinIO客户端失败:", err)
	}

	s.minioClient = minioClient
	s.bucketName = "my-bucket"
}

func (s *MinioTestSuite) TestMinioUploadImg() {
	// 上传文件
	objectName := `远程loopy.jpg`
	filePath := `D:\minIO\data\loopy\loopy.jpg`
	uploadInfo, err := s.minioClient.FPutObject(context.Background(), s.bucketName, objectName, filePath, minio.PutObjectOptions{})

	log.Println(uploadInfo)

	if err != nil {
		log.Fatalln("上传文件失败:", err)
	}
	log.Println("文件上传成功")
}

func (s *MinioTestSuite) TestMinioDownloadImg() {
	// 下载文件
	downloadPath := `D:\minIO\data\本地课表.jpg`
	objectName := `课表.png`
	err := s.minioClient.FGetObject(context.Background(), s.bucketName, objectName, downloadPath, minio.GetObjectOptions{})
	if err != nil {
		log.Fatalln("下载文件失败:", err)
	}
	log.Println("文件下载成功")
}

func (s *MinioTestSuite) TestMinioUploadArticle() {
	// 上传text/plain类型的字符串
	objectName := strconv.FormatInt(803, 10)
	content := "this is a article，是我的文章"
	uploadInfo, err := s.minioClient.PutObject(context.Background(), s.bucketName, objectName, bytes.NewReader([]byte(content)), -1, minio.PutObjectOptions{
		ContentType: "text/plain;charset=utf-8",
	})

	log.Println("key：", uploadInfo.Key)

	if err != nil {
		log.Fatalln("上传文章失败:", err)
	}
	log.Println("上传文章成功")
}

func (s *MinioTestSuite) TestMinioDownloadArticle() {
	objectName := `802`
	reader, err := s.minioClient.GetObject(context.Background(), s.bucketName, objectName, minio.GetObjectOptions{})
	content, err := io.ReadAll(reader)
	if err != nil {
		log.Fatalln("下载文章失败:", err)
	}
	log.Println("文章内容：", string(content))
}

func (s *MinioTestSuite) TestMinioDelete() {
	// 删除文件
	objectName := "803"
	err := s.minioClient.RemoveObject(context.Background(), s.bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		log.Fatalln("删除文件失败:", err)
	}
	log.Println("文件删除成功")
}

func (s *MinioTestSuite) TestMinioList() {
	// 列出桶内对象
	objectCh := s.minioClient.ListObjects(context.Background(), s.bucketName, minio.ListObjectsOptions{Recursive: true})
	for object := range objectCh {
		if object.Err != nil {
			log.Println(object.Err)
			return
		}
		fmt.Println("桶文件名称：" + object.Key)
	}
}

func TestMinio(t *testing.T) {
	suite.Run(t, &MinioTestSuite{})
}
