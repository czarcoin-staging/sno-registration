// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

package ratelimit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRateLimiter(t *testing.T) {
	numEvents := 3
	numLimits := 3
	duration := time.Millisecond
	rateLimiter := NewRateLimiter(duration, numEvents, numLimits)

	key := "key"
	for i := 0; i < numEvents; i++ {
		isAllowed := rateLimiter.IsAllowed(key, time.Now().UTC())
		assert.True(t, isAllowed)
	}

	isAllowed := rateLimiter.IsAllowed(key, time.Now().UTC())
	assert.False(t, isAllowed)
}
