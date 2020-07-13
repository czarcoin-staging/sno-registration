// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

package ratelimit

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter allows to prevent multiple events in fixed period of time.
type RateLimiter struct {
	mu          sync.Mutex
	rateLimited map[string]*limit // list of limited entities.
	duration    time.Duration     // interval during which events are not limiting.
	numEvents   int               // maximum number of events allowed during duration.
	numLimits   int               // maximum number of limits that we store.
}

// limit holds fact of some event occurred.
type limit struct {
	limiter  *rate.Limiter
	occursAt time.Time
}

// NewRateLimiter is a constructor for NewRateLimiter.
func NewRateLimiter(duration time.Duration, numEvents, numLimits int) *RateLimiter {
	return &RateLimiter{
		rateLimited: make(map[string]*limit),
		duration:    duration,
		numEvents:   numEvents,
		numLimits:   numLimits,
	}
}

// Run periodically cleans up limits that are not rate limited now.
func (rateLimiter *RateLimiter) Run(ctx context.Context) {
	cleanupTicker := time.NewTicker(rateLimiter.duration)
	defer cleanupTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-cleanupTicker.C:
			rateLimiter.free()
		}
	}
}

// free is used to remove old limits by key.
func (rateLimiter *RateLimiter) free() {
	rateLimiter.mu.Lock()
	defer rateLimiter.mu.Unlock()
	for key, limit := range rateLimiter.rateLimited {
		if time.Since(limit.occursAt) > rateLimiter.duration {
			delete(rateLimiter.rateLimited, key)
		}
	}
}

// IsAllowed indicates if event is allowed to happen.
func (rateLimiter *RateLimiter) IsAllowed(key string, occursAt time.Time) bool {
	rateLimiter.mu.Lock()
	defer rateLimiter.mu.Unlock()

	v, exists := rateLimiter.rateLimited[key]
	if exists {
		v.occursAt = occursAt
		return v.limiter.AllowN(occursAt, 1)
	}

	if len(rateLimiter.rateLimited) >= rateLimiter.numLimits {
		rateLimiter.removeOldestLimit()
	}

	limiter := rate.NewLimiter(
		rate.Limit(time.Second)/rate.Limit(rateLimiter.duration),
		rateLimiter.numEvents,
	)

	rateLimiter.rateLimited[key] = &limit{limiter, occursAt}

	return limiter.AllowN(occursAt, 1)
}

// removeOldestLimit removes latest limit from the list of rate limited entities not to overflow our limit store.
func (rateLimiter *RateLimiter) removeOldestLimit() {
	var oldestKey string
	var oldestTime time.Time

	for key, limit := range rateLimiter.rateLimited {
		if oldestTime.IsZero() || limit.occursAt.Before(oldestTime) {
			oldestTime = limit.occursAt
			oldestKey = key
		}
	}

	if oldestKey != "" && len(rateLimiter.rateLimited) >= rateLimiter.numLimits {
		delete(rateLimiter.rateLimited, oldestKey)
	}
}
