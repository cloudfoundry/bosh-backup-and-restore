package ratelimiter

import (
	"errors"
	"fmt"
	"time"
)

type RateLimiter interface {
	RateLimit()
}

type ConnectionRateLimiter struct {
	guard    chan bool
	duration time.Duration
}

func NewConnectionRateLimiter(maxConnections int, durationString string) (RateLimiter, error) {

	if maxConnections < 1 || maxConnections > 100 {
		errorMessage := "max connections cannot be less than 1 or greater than 100"
		fmt.Println(errorMessage)
		return nil, errors.New(errorMessage)
	}

	duration, err := time.ParseDuration(durationString)

	if err != nil {
		fmt.Printf("unable to parse rating limit duration: %s\n", err.Error())
		return nil, err
	}

	if duration <= 0 || duration > (3600*time.Second) {
		errorMessage := "duration cannot be 0 or greater than 3600 seconds"
		fmt.Println(errorMessage)
		return nil, errors.New(errorMessage)
	}

	return &ConnectionRateLimiter{
		guard:    make(chan bool, maxConnections),
		duration: duration,
	}, nil
}

func (t *ConnectionRateLimiter) RateLimit() {

	t.guard <- true

	go func() {
		time.Sleep(t.duration)
		<-t.guard
	}()
}
