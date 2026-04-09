package utils

import (
	"strings"
	"testing"
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
