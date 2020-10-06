// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

package server

import (
	"context"
	"encoding/json"
	"mime"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/gorilla/mux"
	"github.com/zeebo/errs"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"storj.io/snoregistration/internal/ratelimit"
	"storj.io/snoregistration/service"
)

var (
	// Error is an error class that indicates internal admin http server error.
	Error = errs.Class("web server error")
)

// Config contains configuration for sno registration server.
type Config struct {
	Address   string `help:"url sno registration web server" default:"127.0.0.1:8081"`
	StaticDir string `help:"path to the folder with static files for web interface" default:"web"`

	RateLimitDuration  time.Duration `help:"the rate at which request are allowed" default:"5m"`
	RateLimitNumEvents int           `help:"number of events available during duration" default:"5"`
	RateLimitNumLimits int           `help:"number of IPs whose rate limits we store" default:"1000"`

	SEO            string `help:"used to communicate with web crawlers and other web robots" default:"User-agent: *\nDisallow: \nDisallow: /cgi-bin/"`
	CaptchaSiteKey string `help:"captcha site key" default:""`
}

// Server represents main admin portal http server with all endpoints.
//
// architecture: Endpoint
type Server struct {
	log    *zap.Logger
	config *Config

	service     *service.Service
	rateLimiter *ratelimit.RateLimiter

	server   http.Server
	listener net.Listener

	indexTemplate *template.Template
}

// NewServer returns new instance of SNO registration HTTP Server.
func NewServer(log *zap.Logger, service *service.Service, config *Config, listener net.Listener) (*Server, error) {
	server := Server{
		log:      log,
		service:  service,
		config:   config,
		listener: listener,
	}

	server.rateLimiter = ratelimit.NewRateLimiter(server.config.RateLimitDuration, server.config.RateLimitNumEvents, server.config.RateLimitNumLimits)

	fs := http.FileServer(http.Dir(server.config.StaticDir))
	router := mux.NewRouter()
	router.StrictSlash(true)

	err := server.initializeTemplates()
	if err != nil {
		return nil, Error.Wrap(err)
	}

	router.PathPrefix("/static/").Handler(server.compressionHandler(http.StripPrefix("/static", fs)))
	router.Handle("/", http.HandlerFunc(server.Index)).Methods(http.MethodGet)
	router.Handle("/{email}", server.rateLimit(http.HandlerFunc(server.RegistrationToken))).Methods(http.MethodPut)
	router.Handle("/{email}", server.rateLimit(http.HandlerFunc(server.Subscribe))).Methods(http.MethodPost)
	router.HandleFunc("/robots.txt", server.seoHandler)

	server.server = http.Server{
		Handler: router,
	}

	return &server, nil
}

// Index is a web handler for the sno registration web page.
func (server *Server) Index(w http.ResponseWriter, r *http.Request) {
	header := w.Header()

	cspValues := []string{
		"base-uri 'self'",
		"default-src 'self'",
		"connect-src 'self'",
		"frame-ancestors 'self'",
		"frame-src 'self' https://www.google.com/recaptcha/",
		"img-src 'self' data:",
		"font-src 'self'",
		"style-src 'self'",
		"script-src 'sha256-w2NsSS57Cf4nDYGNAK9XFx8sx3ArDhES/M2FtWtz3jk=' 'self' https://www.google.com/recaptcha/ https://www.gstatic.com/recaptcha/",
	}

	header.Set("Content-Type", "text/html; charset=UTF-8")
	header.Set("Content-Security-Policy", strings.Join(cspValues, "; "))
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("Referrer-Policy", "same-origin")

	var data struct {
		CaptchaSiteKey string
	}

	data.CaptchaSiteKey = server.config.CaptchaSiteKey

	err := server.indexTemplate.Execute(w, data)
	if err != nil {
		server.log.Error("can not execute index template", zap.Error(Error.Wrap(err)))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

// RegistrationToken is a web handler for obtaining sno registration token.
func (server *Server) RegistrationToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var err error

	params := mux.Vars(r)
	email, ok := params["email"]
	if !ok {
		server.log.Error("failed to parse email parameter", zap.Error(Error.Wrap(err)))
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	server.log.Debug("new SNO registration attempt", zap.String("email", email))

	token, err := server.service.GetAuthToken(ctx, email)
	if err != nil {
		server.log.Error("failed to get auth token", zap.Error(Error.Wrap(err)))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	server.log.Debug("registration token issued successfully", zap.String("token", token))

	err = json.NewEncoder(w).Encode(token)
	if err != nil {
		server.log.Error("failed to write json error response", zap.Error(Error.Wrap(err)))
		return
	}
}

// Subscribe is a web handler that adds an email to the Storj mailing.
func (server *Server) Subscribe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var err error

	params := mux.Vars(r)
	email, ok := params["email"]
	if !ok {
		server.log.Error("failed to parse email parameter", zap.Error(Error.Wrap(err)))
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	server.log.Debug("new subscription on newsletter", zap.String("email", email))

	err = server.service.Subscribe(ctx, email)
	if err != nil {
		server.log.Error("failed to subscribe to storj newsletter", zap.Error(Error.Wrap(err)))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

// Run starts the server that host webapp and api endpoints.
func (server *Server) Run(ctx context.Context) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	var group errgroup.Group
	group.Go(func() error {
		<-ctx.Done()
		return Error.Wrap(server.server.Shutdown(context.Background()))
	})
	group.Go(func() error {
		server.rateLimiter.Run(ctx)
		return nil
	})
	group.Go(func() error {
		defer cancel()
		return Error.Wrap(server.server.Serve(server.listener))
	})

	return Error.Wrap(group.Wait())
}

// Close closes server and underlying listener.
func (server *Server) Close() error {
	return Error.Wrap(server.server.Close())
}

// rateLimit is an handler that prevents from multiple requests from single ip address.
func (server *Server) rateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			server.log.Error("could not split host to add ip to rate limiter", zap.Error(Error.Wrap(err)))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		isAllowed := server.rateLimiter.IsAllowed(ip, time.Now().UTC())
		if !isAllowed {
			server.log.Debug("rate limit exceeded", zap.String("ip:", ip))
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// compressionHandler is used to compress static content with gzip if browser supports such decoding.
func (server *Server) compressionHandler(fn http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isGzipSupported := strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
		isBrotliSupported := strings.Contains(r.Header.Get("Accept-Encoding"), "br")
		if !isGzipSupported && !isBrotliSupported {
			fn.ServeHTTP(w, r)
			return
		}
		extension := filepath.Ext(r.RequestURI)
		// we compress only fonts, js and css bundles
		formats := map[string]bool{
			".js":  true,
			".css": true,
		}
		isNeededFormatToCompress := formats[extension]

		w.Header().Set("Cache-Control", "public,max-age=31536000,immutable")
		w.Header().Set("Content-Type", mime.TypeByExtension(extension))
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// in case if old browser doesn't support compressing or if file extension is not recommended to compressing
		// just return original file.
		if !isGzipSupported || !isNeededFormatToCompress {
			fn.ServeHTTP(w, r)
			return
		}

		w.Header().Add("Vary", "Accept-Encoding")

		newRequest := r.Clone(r.Context())
		*newRequest.URL = *r.URL

		if isBrotliSupported {
			w.Header().Set("Content-Encoding", "br")
			newRequest.URL.Path += ".br"
		} else {
			w.Header().Set("Content-Encoding", "gzip")
			newRequest.URL.Path += ".gz"
		}

		fn.ServeHTTP(w, newRequest)
	})
}

// seoHandler used to communicate with web crawlers and other web robots.
func (server *Server) seoHandler(w http.ResponseWriter, req *http.Request) {
	header := w.Header()

	header.Set("Content-Type", mime.TypeByExtension(".txt"))
	header.Set("X-Content-Type-Options", "nosniff")

	_, err := w.Write([]byte(server.config.SEO))
	if err != nil {
		server.log.Error(err.Error())
	}
}

// initializeTemplates initializes and caches templates for sno registration server.
func (server *Server) initializeTemplates() (err error) {
	server.indexTemplate, err = template.ParseFiles(filepath.Join(server.config.StaticDir, "dist", "index.html"))
	if err != nil {
		return err
	}

	return nil
}
