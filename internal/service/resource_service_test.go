package service

import (
	"strings"
	"testing"
)

func TestNewUploadedResourceFilePreservesExtensionInObjectKey(t *testing.T) {
	item := NewUploadedResourceFile("脚本.py", 128, "text/x-python")
	if !strings.HasSuffix(item.FileKey, ".py") {
		t.Fatalf("expected file key to preserve extension, got %q", item.FileKey)
	}

	noExt := NewUploadedResourceFile("README", 128, "text/plain")
	if strings.Contains(strings.TrimPrefix(noExt.FileKey, "resources/pending/"), ".") {
		t.Fatalf("expected file key without extension for extensionless file, got %q", noExt.FileKey)
	}
}
