package ratelimiter

type NoOpRateLimiter struct {
}

func NewNoOpRateLimiter() RateLimiter {
	return NoOpRateLimiter{}
}

func (n NoOpRateLimiter) RateLimit() {

}
