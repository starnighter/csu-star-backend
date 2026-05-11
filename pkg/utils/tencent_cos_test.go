package utils

import (
	"crypto/md5"
	"csu-star-backend/config"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"
)

func TestBuildDownloadContentDisposition(t *testing.T) {
	disposition := buildDownloadContentDisposition("实验报告 final.py")
	if !strings.Contains(disposition, `attachment; filename="`) {
		t.Fatalf("expected attachment filename fallback, got %q", disposition)
	}
	if !strings.Contains(disposition, `final.py"`) {
		t.Fatalf("expected fallback filename to preserve extension and ASCII suffix, got %q", disposition)
	}
	if !strings.Contains(disposition, "filename*=UTF-8''%E5%AE%9E%E9%AA%8C%E6%8A%A5%E5%91%8A%20final.py") {
		t.Fatalf("expected UTF-8 filename*, got %q", disposition)
	}
}

func TestBuildASCIIFallbackFilename(t *testing.T) {
	if got := buildASCIIFallbackFilename("实验报告.py"); got != "download.py" {
		t.Fatalf("expected fallback extension to be preserved, got %q", got)
	}
	if got := buildASCIIFallbackFilename("week 1 (final).py"); got != "week 1 (final).py" {
		t.Fatalf("expected ASCII filename to stay readable, got %q", got)
	}
	if got := buildASCIIFallbackFilename("   "); got != "download" {
		t.Fatalf("expected empty filename fallback, got %q", got)
	}
}

func TestBuildTencentCDNDownloadURL(t *testing.T) {
	previousConfig := config.GlobalConfig
	t.Cleanup(func() {
		config.GlobalConfig = previousConfig
	})

	config.GlobalConfig = &config.Config{
		Tencent: config.TencentConfig{
			Cos: config.CosConfig{
				CDNDomain:         "file.csustar.wiki",
				CDNAuthEnabled:    true,
				CDNAuthType:       "A",
				CDNAuthKey:        "test-key",
				CDNAuthTTLSeconds: 300,
			},
		},
	}

	query := url.Values{}
	query.Set("response-content-disposition", "attachment")
	got, err := buildTencentCDNDownloadURL("resources/a.pdf", query, time.Unix(1000, 0))
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Scheme != "https" || parsed.Host != "file.csustar.wiki" || parsed.Path != "/resources/a.pdf" {
		t.Fatalf("unexpected CDN URL: %q", got)
	}
	if parsed.Query().Get("response-content-disposition") != "attachment" {
		t.Fatalf("expected response-content-disposition to be preserved")
	}

	signParts := strings.Split(parsed.Query().Get("sign"), "-")
	if len(signParts) != 4 {
		t.Fatalf("expected TypeA sign format, got %q", parsed.Query().Get("sign"))
	}
	if signParts[0] != "1300" || signParts[2] != "0" {
		t.Fatalf("unexpected TypeA sign metadata: %q", parsed.Query().Get("sign"))
	}
	hash := md5.Sum([]byte(fmt.Sprintf("%s-%s-%s-0-%s", parsed.EscapedPath(), signParts[0], signParts[1], "test-key")))
	if signParts[3] != hex.EncodeToString(hash[:]) {
		t.Fatalf("unexpected TypeA signature hash")
	}
}

func TestTencentCosDownloadTemporarilyUsesCDNAuth(t *testing.T) {
	previousConfig := config.GlobalConfig
	previousClient := cosClient
	t.Cleanup(func() {
		config.GlobalConfig = previousConfig
		cosClient = previousClient
	})

	config.GlobalConfig = &config.Config{
		Tencent: config.TencentConfig{
			SecretID:  "test-ak",
			SecretKey: "test-sk",
			Cos: config.CosConfig{
				CDNDomain:         "file.csustar.wiki",
				CDNAuthEnabled:    true,
				CDNAuthType:       "A",
				CDNAuthKey:        "test-key",
				CDNAuthTTLSeconds: 300,
			},
		},
	}

	bucketURL, err := url.Parse("https://bucket-123.cos.ap-guangzhou.myqcloud.com")
	if err != nil {
		t.Fatal(err)
	}
	cosClient = cos.NewClient(&cos.BaseURL{BucketURL: bucketURL}, &http.Client{})

	got, err := TencentCosDownloadTemporarily("resources/a.pdf", "a.pdf")
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Host != "file.csustar.wiki" {
		t.Fatalf("expected CDN host, got %q", parsed.Host)
	}
	if parsed.Query().Get("sign") == "" {
		t.Fatalf("expected CDN TypeA sign parameter")
	}
	if parsed.Query().Get("q-ak") != "" || parsed.Query().Get("q-signature") != "" {
		t.Fatalf("expected CDN URL not to include COS signature params, got %q", got)
	}
	if parsed.Query().Get("response-content-disposition") != "" {
		t.Fatalf("expected CDN URL not to include response-content-disposition, got %q", got)
	}
}

func TestTencentCosDownloadTemporarilyFallsBackToCOSPresignedURL(t *testing.T) {
	previousConfig := config.GlobalConfig
	previousClient := cosClient
	t.Cleanup(func() {
		config.GlobalConfig = previousConfig
		cosClient = previousClient
	})

	config.GlobalConfig = &config.Config{
		Tencent: config.TencentConfig{
			SecretID:  "test-ak",
			SecretKey: "test-sk",
			Cos: config.CosConfig{
				CDNDomain: "file.csustar.wiki",
			},
		},
	}

	bucketURL, err := url.Parse("https://bucket-123.cos.ap-guangzhou.myqcloud.com")
	if err != nil {
		t.Fatal(err)
	}
	cosClient = cos.NewClient(&cos.BaseURL{BucketURL: bucketURL}, &http.Client{})

	got, err := TencentCosDownloadTemporarily("resources/a.pdf", "a.pdf")
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Host != "bucket-123.cos.ap-guangzhou.myqcloud.com" {
		t.Fatalf("expected COS host fallback, got %q", parsed.Host)
	}
	if parsed.Query().Get("q-ak") == "" || parsed.Query().Get("q-signature") == "" {
		t.Fatalf("expected COS signature params, got %q", got)
	}
}
