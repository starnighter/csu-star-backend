package utils

import (
	"csu-star-backend/config"
	"net/http"
	"net/url"
	"strings"
	"testing"

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

func TestApplyCosCDNDomain(t *testing.T) {
	previousConfig := config.GlobalConfig
	t.Cleanup(func() {
		config.GlobalConfig = previousConfig
	})

	config.GlobalConfig = &config.Config{
		Tencent: config.TencentConfig{
			Cos: config.CosConfig{
				CDNDomain: "file.csustar.wiki",
			},
		},
	}

	rawURL := "https://bucket-123.cos.ap-guangzhou.myqcloud.com/resources/a.pdf?q-sign-algorithm=sha1&response-content-disposition=attachment"
	got := applyCosCDNDomain(rawURL)
	want := "https://file.csustar.wiki/resources/a.pdf?q-sign-algorithm=sha1&response-content-disposition=attachment"
	if got != want {
		t.Fatalf("expected CDN URL %q, got %q", want, got)
	}
}

func TestTencentCosDownloadTemporarilyUsesCDNWithoutSigningHost(t *testing.T) {
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
	if parsed.Host != "file.csustar.wiki" {
		t.Fatalf("expected CDN host, got %q", parsed.Host)
	}
	if parsed.Query().Get("q-header-list") != "" {
		t.Fatalf("expected host not to be signed, got q-header-list=%q", parsed.Query().Get("q-header-list"))
	}
	if parsed.Query().Get("response-content-disposition") == "" {
		t.Fatalf("expected response-content-disposition to be preserved")
	}
}
