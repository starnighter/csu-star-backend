package utils

import (
	"crypto/tls"
	"csu-star-backend/config"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"net/textproto"
	"sort"
	"strings"
	"time"
)

type CampusMailboxStatus int

const (
	CampusMailboxStatusExists CampusMailboxStatus = iota
	CampusMailboxStatusNotFound
	CampusMailboxStatusRetry
	CampusMailboxStatusUnknown
)

const campusMailboxProbeTimeout = 10 * time.Second

func CheckCampusMailboxStatus(email string) (CampusMailboxStatus, error) {
	domain, err := campusEmailDomain(email)
	if err != nil {
		return CampusMailboxStatusUnknown, err
	}

	sender, err := verificationProbeSender()
	if err != nil {
		return CampusMailboxStatusUnknown, err
	}

	mxRecords, err := net.LookupMX(domain)
	if err != nil {
		return CampusMailboxStatusUnknown, err
	}
	if len(mxRecords) == 0 {
		return CampusMailboxStatusUnknown, fmt.Errorf("no mx records found for %s", domain)
	}

	sort.Slice(mxRecords, func(i, j int) bool {
		if mxRecords[i].Pref == mxRecords[j].Pref {
			return mxRecords[i].Host < mxRecords[j].Host
		}
		return mxRecords[i].Pref < mxRecords[j].Pref
	})

	var probeErrs []error
	hasExplicitRetry := false
	for _, mx := range mxRecords {
		host := strings.TrimSuffix(strings.TrimSpace(mx.Host), ".")
		if host == "" {
			continue
		}

		status, err := probeCampusMailboxViaMX(host, sender, email)
		if err == nil {
			return status, nil
		}
		if status == CampusMailboxStatusNotFound {
			return status, nil
		}
		if status == CampusMailboxStatusRetry {
			hasExplicitRetry = true
		}
		probeErrs = append(probeErrs, fmt.Errorf("%s: %w", host, err))
	}

	if len(probeErrs) == 0 {
		return CampusMailboxStatusUnknown, errors.New("no valid mx host available")
	}
	if hasExplicitRetry {
		return CampusMailboxStatusRetry, errors.Join(probeErrs...)
	}
	return CampusMailboxStatusUnknown, errors.Join(probeErrs...)
}

func campusEmailDomain(email string) (string, error) {
	at := strings.LastIndex(email, "@")
	if at <= 0 || at == len(email)-1 {
		return "", fmt.Errorf("invalid email address: %s", email)
	}
	return strings.ToLower(strings.TrimSpace(email[at+1:])), nil
}

func verificationProbeSender() (string, error) {
	providers := verificationSMTPProviders()
	if len(providers) == 0 {
		return "", errors.New("no smtp providers configured for mailbox probe")
	}

	sender := strings.TrimSpace(providers[0].FromEmailAddr)
	if sender == "" {
		return "", errors.New("mailbox probe sender is empty")
	}
	return sender, nil
}

func probeCampusMailboxViaMX(mxHost, sender, recipient string) (CampusMailboxStatus, error) {
	conn, err := (&net.Dialer{Timeout: campusMailboxProbeTimeout}).Dial("tcp", net.JoinHostPort(mxHost, "25"))
	if err != nil {
		return CampusMailboxStatusUnknown, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(campusMailboxProbeTimeout))

	client, err := smtp.NewClient(conn, mxHost)
	if err != nil {
		return CampusMailboxStatusUnknown, err
	}
	defer client.Close()

	if err = client.Hello(mailboxProbeHelloName()); err != nil {
		return classifySMTPProbeError(err)
	}

	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{
			ServerName: mxHost,
			MinVersion: tls.VersionTLS12,
		}
		if err = client.StartTLS(tlsConfig); err != nil {
			return classifySMTPProbeError(err)
		}
		if err = client.Hello(mailboxProbeHelloName()); err != nil {
			return classifySMTPProbeError(err)
		}
	}

	if err = client.Mail(sender); err != nil {
		return classifySMTPProbeError(err)
	}
	if err = client.Rcpt(recipient); err != nil {
		return classifySMTPProbeError(err)
	}
	if err = client.Quit(); err != nil {
		return CampusMailboxStatusUnknown, err
	}
	return CampusMailboxStatusExists, nil
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

func classifySMTPProbeError(err error) (CampusMailboxStatus, error) {
	var smtpErr *textproto.Error
	if errors.As(err, &smtpErr) {
		switch {
		case smtpErr.Code >= 200 && smtpErr.Code < 300:
			return CampusMailboxStatusExists, nil
		case smtpErr.Code == 550:
			return CampusMailboxStatusNotFound, err
		case smtpErr.Code >= 400 && smtpErr.Code < 500:
			return CampusMailboxStatusRetry, err
		default:
			return CampusMailboxStatusUnknown, err
		}
	}
	return CampusMailboxStatusUnknown, err
}
