package common

import (
	"time"
)

func Retry(attempts int, interval time.Duration, fn func() error) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	var err error

	for attempts > 0 {
		<-ticker.C // Wait for the ticker interval.
		attempts--
		err = fn()
		if err == nil {
			return nil
		}
	}
	// Return the err if all attempts fail.
	return err
}

func RetryGet[T any](attempts int, interval time.Duration, fn func() (*T, error)) (*T, error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	var err error
	var obj *T

	for attempts > 0 {
		<-ticker.C // Wait for the ticker interval.
		attempts--
		obj, err = fn()
		if err == nil {
			return obj, nil
		}
	}
	// Return nil object if all attempts fail.
	return nil, err
}
