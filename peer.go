// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

package snoregistration

import (
	"context"
	"errors"
	"net"

	"github.com/zeebo/errs"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"storj.io/snoregistration/internal/analytics"
	"storj.io/snoregistration/server"
	"storj.io/snoregistration/service"
)

// Config contains configurable values for sno registration Peer.
type Config struct {
	CAServerURL string `help:"url to the CA server" default:""`
	Analytics   analytics.Config
	Server      server.Config
}

// Peer is the representation of a SNO registration service.
type Peer struct {
	Log      *zap.Logger
	Listener net.Listener
	Service  *service.Service
	Endpoint *server.Server
	Segment  *analytics.Client
}

// New is a constructor for sno registration Peer.
func New(log *zap.Logger, config Config) (peer *Peer, err error) {
	segment, err := analytics.NewClient(log, config.Analytics)
	if err != nil {
		return nil, err
	}
	peer = &Peer{
		Log:     log,
		Segment: segment,
	}

	peer.Listener, err = net.Listen("tcp", config.Server.Address)
	if err != nil {
		return nil, err
	}

	regservice := service.NewService(segment, config.CAServerURL)
	peer.Service = regservice

	peer.Endpoint, err = server.NewServer(peer.Log.Named("SNO registration"), regservice, &config.Server, peer.Listener)
	if err != nil {
		return nil, errs.Combine(
			err,
			peer.Listener.Close(),
		)
	}

	return peer, nil
}

// Run runs SNO registration service until it's either closed or it errors.
func (peer *Peer) Run(ctx context.Context) error {
	group, ctx := errgroup.WithContext(ctx)

	// start SNO registration service as a separate goroutine.
	group.Go(func() error {
		return ignoreCancel(peer.Endpoint.Run(ctx))
	})

	return group.Wait()
}

// Close closes all the resources.
func (peer *Peer) Close() error {
	errlist := errs.Group{}

	if peer.Endpoint != nil {
		errlist.Add(peer.Endpoint.Close())
	}

	if peer.Listener != nil {
		errlist.Add(peer.Listener.Close())
	}

	if peer.Segment != nil {
		errlist.Add(peer.Segment.Close())
	}

	return errlist.Err()
}

// we ignore cancellation and stopping errors since they are expected.
func ignoreCancel(err error) error {
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}
