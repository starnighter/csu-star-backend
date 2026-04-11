package utils

import (
	"errors"
	"testing"

	emailverifier "github.com/AfterShip/email-verifier"
)

func TestCheckCampusMailboxStatusReturnsExists(t *testing.T) {
	originalVerify := verifyCampusMailboxWithAfterShip
	verifyCampusMailboxWithAfterShip = func(email string) (*emailverifier.Result, error) {
		return &emailverifier.Result{
			Email:        email,
			Reachable:    "yes",
			HasMxRecords: true,
			Syntax:       emailverifier.Syntax{Valid: true},
			SMTP:         &emailverifier.SMTP{Deliverable: true},
		}, nil
	}
	t.Cleanup(func() {
		verifyCampusMailboxWithAfterShip = originalVerify
	})

	status, err := CheckCampusMailboxStatus("test@csu.edu.cn")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != CampusMailboxStatusExists {
		t.Fatalf("expected exists status, got %v", status)
	}
}

func TestCheckCampusMailboxStatusReturnsNotFound(t *testing.T) {
	originalVerify := verifyCampusMailboxWithAfterShip
	verifyCampusMailboxWithAfterShip = func(email string) (*emailverifier.Result, error) {
		return &emailverifier.Result{
			Email:        email,
			Reachable:    "no",
			HasMxRecords: true,
			Syntax:       emailverifier.Syntax{Valid: true},
			SMTP:         &emailverifier.SMTP{Deliverable: false, CatchAll: false},
		}, nil
	}
	t.Cleanup(func() {
		verifyCampusMailboxWithAfterShip = originalVerify
	})

	status, err := CheckCampusMailboxStatus("test@csu.edu.cn")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != CampusMailboxStatusNotFound {
		t.Fatalf("expected not found status, got %v", status)
	}
}

func TestCheckCampusMailboxStatusReturnsRetryForTemporaryLookupError(t *testing.T) {
	originalVerify := verifyCampusMailboxWithAfterShip
	verifyCampusMailboxWithAfterShip = func(email string) (*emailverifier.Result, error) {
		return nil, &emailverifier.LookupError{
			Message: emailverifier.ErrTryAgainLater,
			Details: "421 try again later",
		}
	}
	t.Cleanup(func() {
		verifyCampusMailboxWithAfterShip = originalVerify
	})

	status, err := CheckCampusMailboxStatus("test@csu.edu.cn")
	if err == nil {
		t.Fatal("expected lookup error")
	}
	if status != CampusMailboxStatusRetry {
		t.Fatalf("expected retry status, got %v", status)
	}
}

func TestCheckCampusMailboxStatusReturnsUnknownForTimeout(t *testing.T) {
	originalVerify := verifyCampusMailboxWithAfterShip
	verifyCampusMailboxWithAfterShip = func(email string) (*emailverifier.Result, error) {
		return nil, &emailverifier.LookupError{
			Message: emailverifier.ErrTimeout,
			Details: "dial tcp timeout",
		}
	}
	t.Cleanup(func() {
		verifyCampusMailboxWithAfterShip = originalVerify
	})

	status, err := CheckCampusMailboxStatus("test@csu.edu.cn")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if status != CampusMailboxStatusUnknown {
		t.Fatalf("expected unknown status, got %v", status)
	}
}

func TestClassifyAfterShipVerifyErrorFallsBackToUnknown(t *testing.T) {
	status, err := classifyAfterShipVerifyError(errors.New("network error"))
	if err == nil {
		t.Fatal("expected original error")
	}
	if status != CampusMailboxStatusUnknown {
		t.Fatalf("expected unknown status, got %v", status)
	}
}
