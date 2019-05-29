package main

import (
	"github.com/cloudfoundry-community/go-cfclient"
)

var bufClient *cfclient.Client

func retrieveClient() (*cfclient.Client, error) {
	if bufClient != nil {
		return bufClient, nil
	}
	c := &cfclient.Config{
		ApiAddress:        options.Api,
		ClientID:          options.ClientID,
		ClientSecret:      options.ClientSecret,
		Username:          options.Username,
		Password:          options.Password,
		SkipSslValidation: options.SkipSSLValidation,
	}

	bufClient, err := cfclient.NewClient(c)
	if err != nil {
		return bufClient, err
	}
	return bufClient, nil
}
