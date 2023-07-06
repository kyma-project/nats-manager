package retry

import (
	"fmt"
	"time"

	"go.uber.org/zap"
)

func Do(attempts int, interval time.Duration, logger *zap.Logger, fn func() error) error {
	ticker := time.NewTicker(interval)
	var err error
	for {
		select {
		case <-ticker.C:
			attempts--
			err = fn()
			if err != nil {
				logger.Warn(fmt.Sprintf("error while retrying: %s", err.Error()))
			}
			if err == nil || attempts == 0 {
				return err
			}
			logger.Warn(fmt.Sprintf("retrying with %v attempts left", attempts))
		}
	}
}

func Get[T any](attempts int, interval time.Duration, logger *zap.Logger, fn func() (*T, error)) (*T, error) {
	ticker := time.NewTicker(interval)
	var err error
	var obj *T
	for {
		select {
		case <-ticker.C:
			attempts--
			obj, err = fn()
			if err != nil {
				logger.Warn(fmt.Sprintf("error while retrying: %s", err.Error()))
			}
			if err == nil || attempts == 0 {
				return obj, err
			}
			logger.Warn(fmt.Sprintf("retrying with %v attempts left", attempts))
		}
	}
}
