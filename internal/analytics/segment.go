// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

package analytics

import (
	"time"

	"github.com/zeebo/errs"
	"go.uber.org/zap"
	"gopkg.in/segmentio/analytics-go.v3"
)

// Config contains configurable values for segment analytics.
type Config struct {
	PublicKey string `help:"write key for segment.io service" default:""`
	BatchSize int    `help:"number of messages queued before sending" default:"1"`
	Verbose   bool   `help:"when set to true it will log debug lvl logs, not only errors" default:"false"`
}

// Error is an error class that indicates internal analytics error.
var Error = errs.Class("analytics error")

// Client is a custom client for Segment service.
type Client struct {
	client analytics.Client
}

// NewClient is a constructor for Client client.
func NewClient(log *zap.Logger, config Config) (*Client, error) {
	client, err := analytics.NewWithConfig(config.PublicKey, analytics.Config{
		BatchSize: config.BatchSize,
		Logger:    analytics.StdLogger(zap.NewStdLog(log)),
		Verbose:   config.Verbose,
	})
	if err != nil {
		return nil, Error.Wrap(err)
	}

	return &Client{
		client: client,
	}, nil
}

// Identify sends user information like email and auth token to the Client service.
func (segment Client) Identify(email, token string, identifiedAt time.Time) error {
	return Error.Wrap(segment.client.Enqueue(analytics.Identify{
		UserId:    email,
		Timestamp: identifiedAt,
		Traits:    analytics.NewTraits().Set("email", email).Set("authToken", token),
	}))
}

// Subscribe adds an email to the Storj mailing.
func (segment Client) Subscribe(email string) error {
	err := segment.client.Enqueue(analytics.Identify{
		UserId:    email,
		Timestamp: time.Now().UTC(),
		Traits:    analytics.NewTraits().Set("email", email).Set("storj_newsletter", true),
	})
	if err != nil {
		return Error.Wrap(err)
	}

	return Error.Wrap(segment.client.Enqueue(analytics.Track{
		UserId:    email,
		Event:     "storj_newsletter",
		Timestamp: time.Now().UTC(),
	}))
}

// Close and flush segment client metrics metrics.
func (segment Client) Close() error {
	return Error.Wrap(segment.client.Close())
}
