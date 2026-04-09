package utils

import (
	"net/http"
	"time"
)

func NewHttpClient(timeout time.Duration, maxIdle int) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:        maxIdle,
			MaxIdleConnsPerHost: maxIdle / 2,
			IdleConnTimeout:     90 * time.Second,
		},
	}
}
