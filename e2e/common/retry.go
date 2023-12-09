package common

import (
	"time"
)

func Retry(attempts int, interval time.Duration, fn func() error) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	var err error

	for attempts > 0 {
		<-ticker.C
		attempts--
		err = fn()
		if err == nil {
			return nil
		}
	}
	return err
}

func RetryGet[T any](attempts int, interval time.Duration, fn func() (*T, error)) (*T, error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	var err error
	var obj *T

	for attempts > 0 {
		<-ticker.C
		attempts--
		obj, err = fn()
		if err == nil {
			return obj, nil
		}
	}
	return obj, err
}
