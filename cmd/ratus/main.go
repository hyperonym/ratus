package main

import (
	"fmt"
	"log"
	"runtime/debug"

	"github.com/alexflint/go-arg"

	"github.com/hyperonym/ratus/internal/config"
)

// version contains the version string set by -ldflags.
var version string

// args contains the command line arguments.
type args struct {
	Engine string `arg:"--engine,env:ENGINE" placeholder:"NAME" help:"name of the storage engine to be used" default:"mongodb"`
	config.ServerConfig
	config.ChoreConfig
	config.PaginationConfig
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

	// TODO
	fmt.Printf("%#v\n", a)

	return nil
}
