package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/pivotal-cf-experimental/go-pivnet"
	"github.com/pivotal-cf-experimental/pivnet-resource/check"
	"github.com/pivotal-cf-experimental/pivnet-resource/concourse"
	"github.com/pivotal-cf-experimental/pivnet-resource/filter"
	"github.com/pivotal-cf-experimental/pivnet-resource/gp"
	"github.com/pivotal-cf-experimental/pivnet-resource/gp/lagershim"
	"github.com/pivotal-cf-experimental/pivnet-resource/useragent"
	"github.com/pivotal-cf-experimental/pivnet-resource/validator"
	"github.com/pivotal-golang/lager"
	"github.com/robdimsdale/sanitizer"
)

var (
	// version is deliberately left uninitialized so it can be set at compile-time
	version string

	l lager.Logger
)

func main() {
	if version == "" {
		version = "dev"
	}

	var input concourse.CheckRequest

	fmt.Fprintf(os.Stderr, "PivNet Resource version: %s\n", version)

	err := json.NewDecoder(os.Stdin).Decode(&input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Exiting with error: %v\n", err)
		log.Fatalln(err)
	}

	sanitized := concourse.SanitizedSource(input.Source)
	sanitizer := sanitizer.NewSanitizer(sanitized, os.Stderr)

	l = lager.NewLogger("pivnet-resource")
	l.RegisterSink(lager.NewWriterSink(sanitizer, lager.DEBUG))

	err = validator.NewCheckValidator(input).Validate()
	if err != nil {
		l.Error("Exiting with error", err)
		log.Fatalln(err)
	}

	var endpoint string
	if input.Source.Endpoint != "" {
		endpoint = input.Source.Endpoint
	} else {
		endpoint = pivnet.DefaultHost
	}

	specialLogger := lager.NewLogger("pivnet client")
	specialLogger.RegisterSink(lager.NewWriterSink(ioutil.Discard, lager.DEBUG))
	sp := lagershim.NewLagerShim(specialLogger)

	clientConfig := pivnet.ClientConfig{
		Host:      endpoint,
		Token:     input.Source.APIToken,
		UserAgent: useragent.UserAgent(version, "check", input.Source.ProductSlug),
	}
	client := gp.NewClient(
		clientConfig,
		sp,
	)

	f := filter.NewFilter()

	extendedClient := gp.NewExtendedClient(client, sp)

	response, err := check.NewCheckCommand(
		version,
		l,
		f,
		client,
		extendedClient,
	).Run(input)
	if err != nil {
		l.Error("Exiting with error", err)
		log.Fatalln(err)
	}

	err = json.NewEncoder(os.Stdout).Encode(response)
	if err != nil {
		l.Error("Exiting with error", err)
		log.Fatalln(err)
	}
}
