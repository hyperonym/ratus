package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/alexflint/go-arg"
	"golang.org/x/sync/errgroup"

	"github.com/hyperonym/ratus/docs"
	"github.com/hyperonym/ratus/internal/config"
	"github.com/hyperonym/ratus/internal/controller"
	"github.com/hyperonym/ratus/internal/engine"
	"github.com/hyperonym/ratus/internal/engine/mongodb"
	"github.com/hyperonym/ratus/internal/metrics"
	"github.com/hyperonym/ratus/internal/middleware"
	"github.com/hyperonym/ratus/internal/router"
)

// version contains the version string set by -ldflags.
var version string

// Create type aliases for embedding engine-specific configurations.
type (
	mongodbConfig = mongodb.Config
)

// args contains the command line arguments.
type args struct {
	Engine string `arg:"--engine,env:ENGINE" placeholder:"NAME" help:"name of the storage engine to be used" default:"mongodb"`
	config.ServerConfig
	config.ChoreConfig
	config.PaginationConfig
	mongodbConfig
}

// Version returns a version string based on how the binary was compiled.
// For binaries compiled with "make", the version set by -ldflags is returned.
// For binaries compiled with "go install", the version and commit hash from
// the embedded build information is returned if available.
func (args) Version() string {
	if info, ok := debug.ReadBuildInfo(); ok && version == "" {
		version = info.Main.Version
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" {
				version += "-" + s.Value
				break
			}
		}
	}
	return version
}

func main() {

	// Wrap the real main function to allow exiting with an error code without
	// affecting deferred functions. https://stackoverflow.com/a/18969976
	if err := run(); err != nil {
		log.Fatal(err)
	}
	log.Println("shut down gracefully")
}

func run() error {

	// Parse command line arguments.
	var a args
	arg.MustParse(&a)

	// Create a context without timeout for the initialization phase.
	ctx := context.Background()

	// Create a storage engine instance of the specified type.
	var (
		g   engine.Engine
		err error
	)
	switch strings.ToLower(a.Engine) {
	case "mongodb":
		g, err = mongodb.New(&a.mongodbConfig)
	default:
		err = fmt.Errorf("unknown storage engine: %s", a.Engine)
	}
	if err != nil {
		return err
	}

	// Initialize the storage engine instance and defer the close method for
	// graceful shutdown.
	if err := g.Open(ctx); err != nil {
		return err
	}
	defer g.Close(ctx)

	// Create router and mount API endpoints.
	r := router.New(&controller.V1{
		Pagination: middleware.Pagination(&a.PaginationConfig),
		Topic:      controller.NewTopicController(g),
		Task:       controller.NewTaskController(g),
		Promise:    controller.NewPromiseController(g),
		Health:     controller.NewHealthController(g),
		Metrics:    controller.NewMetricsController(g),
	}, &docs.Swagger{})

	// Start API server and background jobs.
	e, ctx := errgroup.WithContext(ctx)
	e.Go(func() error {
		return serve(ctx, r.Handler(), &a.ServerConfig)
	})
	e.Go(func() error {
		return chore(ctx, g, &a.ChoreConfig)
	})

	return e.Wait()
}

func serve(ctx context.Context, h http.Handler, c *config.ServerConfig) error {

	// A port number of zero will not start the API server.
	// This allows the instance to be responsible for running background jobs only.
	if c.Port <= 0 {
		return nil
	}

	// Create HTTP server using the provided handler.
	a := fmt.Sprintf("%s:%d", c.Bind, c.Port)
	s := &http.Server{
		Addr:    a,
		Handler: h,
	}

	// Listen for termination signals.
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)

	// Gracefully shut down the server when an interrupt or SIGTERM signal
	// is received, or close the server immediately if the context has been
	// canceled or another goroutine in the error group has return an error.
	go func() {
		select {
		case <-ch:
			log.Printf("stop listening on %s\n", a)
			s.Shutdown(context.Background())
		case <-ctx.Done():
			s.Close()
		}
	}()

	// Return errors other than those caused by closing the server.
	log.Printf("start listening on %s\n", a)
	if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

func chore(ctx context.Context, g engine.Engine, c *config.ChoreConfig) error {

	// An interval of zero will not start the background jobs.
	// This allows the instance to be responsible for handling requests only.
	if c.Interval <= 0 {
		return nil
	}

	// Calculate initial delay for the ticker.
	m := 1.0
	if c.InitialRandom {
		s := rand.NewSource(time.Now().UnixNano())
		m = rand.New(s).Float64()
	}
	d := time.Duration(c.InitialDelay.Seconds()*m*float64(time.Second) + 1)

	// Listen for termination signals.
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)

	// Start ticker for background jobs. The ticker will adjust the time
	// interval or drop ticks to make up for slow receivers.
	var n bool
	r := time.NewTicker(d)
	for {
		select {
		case <-ch:
			log.Println("stop running background jobs")
			r.Stop()
			return nil
		case <-ctx.Done():
			r.Stop()
			return ctx.Err()
		case <-r.C:

			// Reset the timer to use the normal interval after the initial delay.
			if !n {
				log.Println("start running background jobs")
				n = true
				r.Reset(c.Interval)
			}

			// Run background jobs and collect the elapsed time.
			t := time.Now()
			if err := g.Chore(ctx); err != nil {
				log.Println(err)
			}
			metrics.ChoreHistogram.Observe(time.Since(t).Seconds())
		}
	}
}
