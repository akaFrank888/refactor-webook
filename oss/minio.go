package main

import (
	"context"
	"fmt"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func main() {
	// MinIO服务器的访问信息
	endpoint := "localhost:9000"
	accessKeyID := "QVSWvxnymDaQ2XsVIAWS"
	secretAccessKey := "syTzQgrcqQydr1aXln7astdRTYtrrme9jaGCEbF9"
	useSSL := true // 根据实际情况设置是否使用SSL，默认为true

	// 创建MinIO客户端
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalln("创建MinIO客户端失败:", err)
	}

	bucketName := "my-bucket"

	// 上传文件
	objectName := `loopy.jpg`
	filePath := `C:\Users\Frank Jone\Desktop`
	_, err = minioClient.FPutObject(context.Background(), bucketName, objectName, filePath, minio.PutObjectOptions{})
	if err != nil {
		log.Fatalln("上传文件失败:", err)
	}
	log.Println("文件上传成功")

	// 下载文件
	downloadPath := "/data"
	err = minioClient.FGetObject(context.Background(), bucketName, objectName, downloadPath, minio.GetObjectOptions{})
	if err != nil {
		log.Fatalln("下载文件失败:", err)
	}
	log.Println("文件下载成功")

	// 删除文件
	err = minioClient.RemoveObject(context.Background(), bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		log.Fatalln("删除文件失败:", err)
	}
	log.Println("文件删除成功")

	// 列出桶内对象
	objectCh := minioClient.ListObjects(context.Background(), bucketName, minio.ListObjectsOptions{Recursive: true})
	for object := range objectCh {
		if object.Err != nil {
			log.Println(object.Err)
			return
		}
		fmt.Println("桶文件名称：" + object.Key)
	}
}
