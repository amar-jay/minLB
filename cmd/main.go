package main

import (
	"errors"
    "time"
	"net/http"
    "context"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"

	"github.com/urfave/cli/v2"
)

type Retries int
const (
	Attempts Retries = iota
	Retry
)

var backendList cli.StringSlice
var serverPool ServerPool
func main() {

    app := &cli.App{
        Name:        "minLB",
        Description: "This is a minimum Load balancer",
        Version: "1.0.0",
        Action:  run,
        Flags: []cli.Flag{
            &cli.IntFlag{
                Name: "port",
                Aliases: []string{"p"},
                Usage: "Port to serve",
                Value: 3000,
            },
            &cli.StringSliceFlag{
                Name: "backends",
                Aliases: []string{"b"},
                Required: true,
                Destination: &backendList,
                Usage: "all the backends to create a load balancer for",
            },
        },
    }

    cli.VersionPrinter = func(c *cli.Context) {
            println("v" + c.App.Version)

    }
    
    if err := app.Run(os.Args); err != nil {
//            println(err)
            os.Exit(1)
    }
}

func run(c *cli.Context) error {
    backendList := backendList.Value()

    if len(backendList) == 0 {
    	return errors.New("Please provide one or more backends to load balance")
    }

    println("Backends running")
    for _, url := range backendList {
        println("\t- " + url)
    }

    for _, u := range backendList {
        serverUrl, err := url.Parse(u)
        if err != nil {
                return err
        }

        proxy := httputil.NewSingleHostReverseProxy(serverUrl)
        proxy.ErrorHandler = NewErrorHandler(serverUrl, proxy) 

        b := Backend{
            URL:          serverUrl,
            ReverseProxy: proxy,
        }
        b.SetAlive(true)
        serverPool.AddBackend(&b)
        println("Configured server: " + serverUrl.String())
    }

    // ln, err := net.Listen("tcp", "0.0.0.0")
    port := strconv.Itoa(c.Int("port"))

    server := http.Server{
        Addr:    ":" + port,
        Handler: http.HandlerFunc(lbHandler),
    }

    // start health checking
    go healthCheck()

    println("Load Balancer started at :" + port)
	if err := server.ListenAndServe(); err != nil {
		return err
    }
    return nil
}



// GetFromContext returns the retries for request[Retry | Attempt]
func GetFromContext(r *http.Request, item Retries) int {
    if retry, ok := r.Context().Value(item).(int); ok {
            return retry
    }
    return 0
}

// healthCheck runs a routine for check status of the backends every 2 mins
func healthCheck() {
	t := time.NewTicker(time.Minute * 2)
	for {
		select {
		case <-t.C:
			println("Starting health check...")
			serverPool.HealthCheck()
			println("Health check completed")
		}
	}
}

func NewErrorHandler(serverUrl *url.URL, proxy *httputil.ReverseProxy) func(w http.ResponseWriter, r *http.Request, e error) {

    return func(w http.ResponseWriter, r *http.Request, e error) {
        println("[url: " + serverUrl.Host + "]" + e.Error())
        retries := GetFromContext(r, Retry)
        if retries < 3 {
                <-time.After(10 * time.Millisecond)
                ctx := context.WithValue(r.Context(), Retry, retries+1)
                proxy.ServeHTTP(w, r.WithContext(ctx))
                return
        }

        			// after 3 retries, mark this backend as down
        serverPool.MarkBackendStatus(serverUrl, false)

        // if the same request routing for few attempts with different backends, increase the count
        attempts := GetFromContext(r, Attempts)
        println(r.RemoteAddr + "(" + r.URL.Path +") Attempting retry %d\n" + string(attempts))
        ctx := context.WithValue(r.Context(), Attempts, attempts+1)
        lbHandler(w, r.WithContext(ctx))
    }
}

func lbHandler(w http.ResponseWriter,r *http.Request) {
    attempts := GetFromContext(r, Attempts)
    if attempts > 3 {
        println( r.RemoteAddr + "(" + r.URL.Path + ") Max attempts reached, terminating")
        http.Error(w, "Service not available", http.StatusServiceUnavailable)
        return
    }

    peer := serverPool.GetNextPeer()
    if peer != nil {
        println("changed service")
        peer.ReverseProxy.ServeHTTP(w, r)
        return
    }

    http.Error(w, "Service not available", http.StatusServiceUnavailable)
}
