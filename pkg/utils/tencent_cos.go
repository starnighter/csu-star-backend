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

func TencentCosDownloadTemporarily(cosKey, downloadName string) (fileUrl string, err error) {
	var opt *cos.PresignedURLOptions
	if disposition := buildDownloadContentDisposition(downloadName); disposition != "" {
		query := url.Values{}
		query.Set("response-content-disposition", disposition)
		opt = &cos.PresignedURLOptions{Query: &query}
	}

	presignedURL, err := cosClient.Object.GetPresignedURL(
		context.Background(),
		http.MethodGet,
		cosKey,
		config.GlobalConfig.Tencent.SecretID,
		config.GlobalConfig.Tencent.SecretKey,
		2*time.Hour,
		opt,
	)
	if err != nil {
		return "", err
	}

	return presignedURL.String(), nil
}

func buildDownloadContentDisposition(downloadName string) string {
	downloadName = strings.TrimSpace(downloadName)
	if downloadName == "" {
		return ""
	}

	fallback := buildASCIIFallbackFilename(downloadName)
	encoded := encodeRFC5987Value(downloadName)
	return fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, fallback, encoded)
}

func buildASCIIFallbackFilename(downloadName string) string {
	downloadName = strings.TrimSpace(downloadName)
	if downloadName == "" {
		return "download"
	}

	ext := filepath.Ext(downloadName)
	base := strings.TrimSuffix(downloadName, ext)
	sanitizedBase := sanitizeASCIIFilenamePart(base)
	sanitizedExt := sanitizeASCIIFilenamePart(ext)

	if sanitizedBase == "" || !containsASCIILetterOrDigit(sanitizedBase) {
		sanitizedBase = "download"
	}
	if sanitizedExt != "" && !strings.HasPrefix(sanitizedExt, ".") {
		sanitizedExt = "." + sanitizedExt
	}

	return sanitizedBase + sanitizedExt
}

func sanitizeASCIIFilenamePart(value string) string {
	var builder strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case strings.ContainsRune(" ._-()[]", r):
			builder.WriteRune(r)
		default:
			builder.WriteByte('_')
		}
	}

	return strings.Trim(strings.Join(strings.Fields(builder.String()), " "), " .")
}

func containsASCIILetterOrDigit(value string) bool {
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return true
		}
	}
	return false
}

func encodeRFC5987Value(value string) string {
	return strings.ReplaceAll(url.QueryEscape(value), "+", "%20")
}

func TencentCosUploadTemporarily(cosKey string) (string, error) {
	presignedURL, err := cosClient.Object.GetPresignedURL(
		context.Background(),
		http.MethodPut,
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

func TencentCosObjectURL(cosKey string) string {
	if cosClient == nil || cosKey == "" {
		return ""
	}
	return cosClient.Object.GetObjectURL(cosKey).String()
}

func TencentCosDeleteObject(cosKey string) error {
	if cosClient == nil || cosKey == "" {
		return nil
	}
	_, err := cosClient.Object.Delete(context.Background(), cosKey)
	return err
}

func TencentCosObjectExists(cosKey string) (bool, error) {
	if cosClient == nil || cosKey == "" {
		return false, nil
	}
	_, err := cosClient.Object.Head(context.Background(), cosKey, nil)
	if err == nil {
		return true, nil
	}

	if responseError := cos.IsNotFoundError(err); responseError {
		return false, nil
	}

	return false, err
}

func GenerateUniqueFileName(rawFileName string) string {
	ext := filepath.Ext(rawFileName)
	return strings.ReplaceAll(uuid.New().String(), "-", "") + ext
}
