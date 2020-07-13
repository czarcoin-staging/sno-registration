This is a web service that allow us to receive SNO auth token.

It has only 1 web page in which we are able to ask an auth token using and 
subscribe to storj newslatter.

This service also contains some CLI that will allow usto generate config and run service.

## Project structure

`cmd` - contains CLI application.

`internal` - contains some internal programming modules.

`server` - contains web server with all endpoint.

`service` - layer in which we place all business related logic.

`web` - contains all static resources, such as html pages, images, styles.

### cmd package

Our CLI has only 2 commands - `setup` and `run`.

`setup` command should be used to create config file - `snoregistration setup`.

`run` command will run web server - `snoregistration run`.

### internal package

This package contains two programming modules - rate limiter and segment client.

#### rate limiter

rate limiter is used to track `n` events during `m` duration. 

If num of events during `m` was fired more that `n` times - event source will be banned for some time.
  
Usage of rate limiter is quite straightforward:

`isAllowed := rateLimiter.IsAllowed(your_string_key)`

#### segment client

Just a custom segment client with 2 functions - Subscribe, that is used to subscribe to Storj newsletter and
Identify, to start email sending flow in customer.io

### server package

Sno registration web server has 3 endpoints with appropriate handlers

```
router.Handle("/", http.HandlerFunc(server.Index)).Methods(http.MethodGet)
router.Handle("/{email}", server.rateLimit(http.HandlerFunc(server.RegistrationToken))).Methods(http.MethodPut)
router.Handle("/{email}", server.rateLimit(http.HandlerFunc(server.Subscribe))).Methods(http.MethodPost)
```

`Index` - will return html page with needed static resources and appropriate http headers.

`RegistrationToken` - is an api endpoint that is used to get SNO registration token from CA server. 

`Subscribe` -  - is an api endpoint that is used to subscribe on Storj newsletter.

Also it has some middleware:

`rateLimit` - checks ip of each request to prevent bruteforce.

`compressionHandler` - checks if browser is able to work with compression. If yes - it will return `gzipped` 
version of static file.

### Configuration

Here is all possible configurations for SNO registration service:

```
// Config contains configurable values for sno registration Peer.
type Config struct {
	CAServerUrl  string `help:"url to the CA server" default:""`
	SegmentioKey string `help:"write key for segment.io service" default:""`
	Server       server.Config
}

// Config contains configuration for sno registration server.
type Config struct {
	Address   string `help:"url sno registration web server" default:"127.0.0.1:8081"`
	StaticDir string `help:"path to the folder with static files for web interface" default:"web"`

	RateLimitDuration  time.Duration `help:"the rate at which request are allowed" default:"5m"`
	RateLimitNumEvents int           `help:"number of events available during duration" default:"5"`
	RateLimitNumLimits int           `help:"number of IPs whose rate limits we store" default:"1000"`
}
```

`CAServerUrl` - is an url of the CA server.
`SegmentioKey` - is an segment public key, source - Storj.io.
`Address` - address of SNO registration web server.
`StaticDir` - path to the static files.