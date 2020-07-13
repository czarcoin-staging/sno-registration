// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

package service

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/zeebo/errs"

	"storj.io/snoregistration/internal/analytics"
)

// Error is the default sno registration error.
var Error = errs.Class("sno registration service error")

// Service exposes all sno registration related logic.
type Service struct {
	baseCaServerURL string
	segment         *analytics.Client
}

// NewService is a constructor for service.
func NewService(segment *analytics.Client, baseCaServerURL string) *Service {
	return &Service{
		baseCaServerURL: baseCaServerURL,
		segment:         segment,
	}
}

// GetAuthToken sends an https request to CA server to receive sno registration token.
func (service *Service) GetAuthToken(ctx context.Context, email string) (string, error) {
	client := &http.Client{}

	baseCaURL, err := url.Parse(service.baseCaServerURL)
	if err != nil {
		return "", Error.Wrap(err)
	}

	// TODO: we should url.QueryEscape(email) but in this case CA server will answer with bad token, like this:
	// TODO: qwe%40ukr.net:1Nf8Wm6kG5VpkhH8WR6BkbJtgdyjrdpmFf5ZxQqBWLLBiT4rHsu1SVLsYqp6yCRQy5PrzqE1iTFEtyffXA7ek78ih1mH45
	// TODO: where qwe%40ukr.net is urlencoded email, but it should be url decoded
	// TODO: instead of this we add this if statement because of lack of time
	if strings.ContainsAny(email, "\n\t/ <>&") {
		return "", Error.Wrap(errs.New("invalid email"))
	}

	baseCaURL.Path = path.Join(baseCaURL.Path, "v1/authorizations/", email)
	req, err := http.NewRequest(http.MethodPut, baseCaURL.String(), nil)
	if err != nil {
		return "", Error.Wrap(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", Error.Wrap(err)
	}
	defer func() {
		err = errs.Combine(err, resp.Body.Close())
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", Error.Wrap(err)
	}

	token := string(body)

	err = service.segment.Identify(email, token, time.Now().UTC())
	if err != nil {
		return "", Error.Wrap(err)
	}

	return string(body), nil
}

// Subscribe adds an email to the Storj mailing.
func (service *Service) Subscribe(ctx context.Context, email string) error {
	return Error.Wrap(service.segment.Subscribe(email))
}
