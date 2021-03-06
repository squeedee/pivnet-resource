package main

import (
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/pivotal-cf-experimental/go-pivnet/cmd/pivnet/commands"
	"github.com/pivotal-cf-experimental/go-pivnet/cmd/pivnet/errors"
	"github.com/pivotal-cf-experimental/go-pivnet/cmd/pivnet/version"
)

var (
	// buildVersion is deliberately left uninitialized so it can be set at compile-time
	buildVersion string
)

func main() {
	if buildVersion == "" {
		version.Version = "dev"
	} else {
		version.Version = buildVersion
	}

	parser := flags.NewParser(&commands.Pivnet, flags.HelpFlag)

	_, err := parser.Parse()
	if err != nil {
		if err == commands.ErrShowHelpMessage {
			helpParser := flags.NewParser(&commands.Pivnet, flags.HelpFlag)
			helpParser.NamespaceDelimiter = "-"
			helpParser.ParseArgs([]string{"-h"})
			helpParser.WriteHelp(os.Stderr)
			os.Exit(0)
		}

		// Do not consider the built-in help an error
		if e, ok := err.(*flags.Error); ok {
			if e.Type == flags.ErrHelp {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(0)
			}
		}

		if err == errors.ErrAlreadyHandled {
			os.Exit(1)
		}

		coloredMessage := fmt.Sprintf(errors.RedFunc(err.Error()))
		fmt.Fprintln(os.Stderr, coloredMessage)
		os.Exit(1)
	}
}
