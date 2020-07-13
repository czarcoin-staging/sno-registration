// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

package analytics

import (
	"time"

	"github.com/zeebo/errs"
	"gopkg.in/segmentio/analytics-go.v3"
)

// TODO: should context be passed to the analytics methods?

// Error is an error class that indicates internal analytics error.
var Error = errs.Class("analytics error")

// Client is a custom client for Segment service.
type Client struct {
	client analytics.Client
}

// NewClient is a constructor for Client client.
func NewClient(pubkey string) *Client {
	segment := Client{
		client: analytics.New(pubkey),
	}

	return &segment
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
