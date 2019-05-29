package main

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/orange-cloudfoundry/cf-audit-actions/messages"
	"os"
)

type Options struct {
	Api               string `short:"a" long:"api" description:"cf api endpoint" required:"true"`
	ClientID          string `short:"i" long:"client-id" description:"cf client id"`
	ClientSecret      string `short:"s" long:"client-secret" description:"cf client id"`
	Username          string `short:"u" long:"username" description:"cf username (if client-id can't bet set)'"`
	Password          string `short:"p" long:"password" description:"cf password (if client-id can't bet set)"`
	Parallel          int    `          long:"parallel" description:"how many parallel request can be made"`
	SkipSSLValidation bool   `short:"k" long:"skip-ssl-validation" description:"Skip ssl validation"`
	Version           func() `short:"v" long:"version" description:"Show version"`
}

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)
var options Options
var parser = flags.NewParser(&options, flags.HelpFlag|flags.PassDoubleDash|flags.IgnoreUnknown)

func Parse(args []string) error {

	askVersion := false
	options.Version = func() {
		askVersion = true
		fmt.Printf("cf-audit-actions %v, commit %v, built at %v", version, commit, date)
	}
	_, err := parser.ParseArgs(args[1:])
	if err != nil {
		if errFlag, ok := err.(*flags.Error); ok && askVersion && errFlag.Type == flags.ErrCommandRequired {
			return nil
		}
		if errFlag, ok := err.(*flags.Error); ok && errFlag.Type == flags.ErrHelp {
			messages.Println(err.Error())
			return nil
		}
		return err
	}

	return nil
}

func main() {
	var err error
	err = Parse(os.Args)
	if err != nil {
		messages.Fatal(err.Error())
	}
}
