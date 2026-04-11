package utils

import (
	"errors"
	"net/textproto"
	"testing"
)

func TestClassifySMTPProbeErrorReturnsRetryFor4xx(t *testing.T) {
	status, err := classifySMTPProbeError(&textproto.Error{Code: 451, Msg: "temporary local problem"})
	if status != CampusMailboxStatusRetry {
		t.Fatalf("expected retry status, got %v", status)
	}
	if err == nil {
		t.Fatal("expected original error to be returned")
	}
}

func TestClassifySMTPProbeErrorReturnsNotFoundFor550(t *testing.T) {
	status, err := classifySMTPProbeError(&textproto.Error{Code: 550, Msg: "mailbox unavailable"})
	if status != CampusMailboxStatusNotFound {
		t.Fatalf("expected not found status, got %v", status)
	}
	if err == nil {
		t.Fatal("expected original error to be returned")
	}
}

func TestClassifySMTPProbeErrorReturnsUnknownForNetworkFailure(t *testing.T) {
	status, err := classifySMTPProbeError(errors.New("i/o timeout"))
	if status != CampusMailboxStatusUnknown {
		t.Fatalf("expected unknown status, got %v", status)
	}
	if err == nil {
		t.Fatal("expected original error to be returned")
	}
}
