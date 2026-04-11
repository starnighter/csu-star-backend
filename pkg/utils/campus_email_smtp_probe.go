package utils

import (
	"csu-star-backend/config"
	"strings"
	"time"

	emailverifier "github.com/AfterShip/email-verifier"
)

type CampusMailboxStatus int

const (
	CampusMailboxStatusExists CampusMailboxStatus = iota
	CampusMailboxStatusNotFound
	CampusMailboxStatusRetry
	CampusMailboxStatusUnknown
)

const campusMailboxProbeTimeout = 5 * time.Second

var verifyCampusMailboxWithAfterShip = func(email string) (*emailverifier.Result, error) {
	verifier := emailverifier.NewVerifier().
		EnableSMTPCheck().
		HelloName(mailboxProbeHelloName()).
		FromEmail(verificationProbeSender()).
		ConnectTimeout(campusMailboxProbeTimeout).
		OperationTimeout(campusMailboxProbeTimeout)

	return verifier.Verify(email)
}

func CheckCampusMailboxStatus(email string) (CampusMailboxStatus, error) {
	result, err := verifyCampusMailboxWithAfterShip(email)
	if err != nil {
		return classifyAfterShipVerifyError(err)
	}
	if result == nil {
		return CampusMailboxStatusUnknown, nil
	}
	if !result.Syntax.Valid || !result.HasMxRecords {
		return CampusMailboxStatusUnknown, nil
	}
	if result.SMTP == nil {
		return CampusMailboxStatusUnknown, nil
	}
	if result.SMTP.Deliverable {
		return CampusMailboxStatusExists, nil
	}
	if strings.EqualFold(result.Reachable, "no") {
		return CampusMailboxStatusNotFound, nil
	}
	return CampusMailboxStatusUnknown, nil
}

func verificationProbeSender() string {
	providers := verificationSMTPProviders()
	if len(providers) == 0 {
		return "user@example.org"
	}
	if sender := strings.TrimSpace(providers[0].FromEmailAddr); sender != "" {
		return sender
	}
	return "user@example.org"
}

func mailboxProbeHelloName() string {
	if config.GlobalConfig == nil {
		return "localhost"
	}

	if bindHost := strings.TrimSpace(config.GlobalConfig.Server.BindHost); bindHost != "" {
		return bindHost
	}
	return "localhost"
}

func classifyAfterShipVerifyError(err error) (CampusMailboxStatus, error) {
	lookupErr, ok := err.(*emailverifier.LookupError)
	if !ok {
		return CampusMailboxStatusUnknown, err
	}

	switch lookupErr.Message {
	case emailverifier.ErrTryAgainLater,
		emailverifier.ErrMailboxBusy,
		emailverifier.ErrExceededMessagingLimits,
		emailverifier.ErrTooManyRCPT,
		emailverifier.ErrFullInbox:
		return CampusMailboxStatusRetry, err
	default:
		return CampusMailboxStatusUnknown, err
	}
}
