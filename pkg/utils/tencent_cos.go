package utils

import (
	"context"
	"csu-star-backend/config"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tencentyun/cos-go-sdk-v5"
)

var (
	cosClient *cos.Client
)

type FileInfo struct {
	Filename string `json:"filename"`
	FileKey  string `json:"file_key"`
	FileUrl  string `json:"file_url"`
	FileSize int64  `json:"file_size"`
	FileHash string `json:"file_hash"`
	MimeType string `json:"mime_type"`
}

func InitTencentCos() error {
	u, err := url.Parse(fmt.Sprintf(
		"https://%s-%s.cos.%s.myqcloud.com",
		config.GlobalConfig.Tencent.Cos.Bucket,
		config.GlobalConfig.Tencent.Cos.AppID,
		config.GlobalConfig.Tencent.Cos.Region,
	))
	if err != nil {
		return err
	}

	b := &cos.BaseURL{BucketURL: u}
	cosClient = cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  config.GlobalConfig.Tencent.SecretID,
			SecretKey: config.GlobalConfig.Tencent.SecretKey,
		},
	})
	return nil
}

func TencentCosUpload(file *multipart.FileHeader, cosKeyPrefix string) (FileInfo, error) {
	var fileInfo FileInfo

	f, err := file.Open()
	if err != nil {
		return fileInfo, err
	}
	defer f.Close()

	fileHash, err := CalculateFileHash(f)
	if err != nil {
		return fileInfo, err
	}

	mimeType, err := DetectMimeType(f)
	if err != nil {
		return fileInfo, err
	}

	uniqueFileName := GenerateUniqueFileName(file.Filename)
	cosKey := cosKeyPrefix + uniqueFileName

	_, err = cosClient.Object.Put(context.Background(), cosKey, f, nil)
	if err != nil {
		return fileInfo, err
	}

	fileURL := cosClient.Object.GetObjectURL(cosKey).String()

	fileInfo.FileKey = cosKey
	fileInfo.FileHash = fileHash
	fileInfo.FileUrl = fileURL
	fileInfo.MimeType = mimeType
	fileInfo.FileSize = file.Size
	fileInfo.Filename = uniqueFileName

	return fileInfo, nil
}

func TencentCosUploadByStream(stream io.Reader, cosKeyPrefix, fileExtension string) (string, error) {
	cosKey := cosKeyPrefix + strings.ReplaceAll(uuid.New().String(), "-", "") + fileExtension
	_, err := cosClient.Object.Put(context.Background(), cosKey, stream, nil)
	if err != nil {
		return "", err
	}

	return cosClient.Object.GetObjectURL(cosKey).String(), nil
}

func TencentCosDownloadTemporarily(cosKey string) (fileUrl string, err error) {
	presignedURL, err := cosClient.Object.GetPresignedURL(
		context.Background(),
		http.MethodGet,
		cosKey,
		config.GlobalConfig.Tencent.SecretID,
		config.GlobalConfig.Tencent.SecretKey,
		2*time.Hour,
		nil,
	)
	if err != nil {
		return "", err
	}

	return presignedURL.String(), nil
}

func GenerateUniqueFileName(rawFileName string) string {
	ext := filepath.Ext(rawFileName)
	return strings.ReplaceAll(uuid.New().String(), "-", "") + ext
}
