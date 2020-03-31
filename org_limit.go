package main

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/cloudfoundry-community/go-cfclient"
)

type OrgLimit struct {
	TimeLimit Duration `short:"l" long:"time-limit" required:"1" description:"Define when an app should be deleted" default:"168h"`
	Force     bool     `short:"f" long:"force" description:"Will apply directly deactivation of ssh without confirmation"`
	Org       string   `short:"o" long:"org" required:"1" description:"Set organization to check"`
}

var orgLimit OrgLimit

func (c *OrgLimit) Execute(_ []string) error {
	initParallel()
	client, err := retrieveClient()
	if err != nil {
		return err
	}

	org, err := client.GetOrgByName(c.Org)
	if err != nil {
		return err
	}

	apps, err := client.ListAppsByQuery(url.Values{
		"q": []string{"organization_guid:" + org.Guid},
	})
	if err != nil {
		return err
	}
	toDelete := make([]cfclient.App, 0)
	for _, app := range apps {
		createdAt, _ := time.Parse(time.RFC3339, app.CreatedAt)
		if createdAt.Add(time.Duration(c.TimeLimit)).Before(time.Now()) {
			toDelete = append(toDelete, app)
		}
	}
	if len(toDelete) == 0 {
		fmt.Println("Nothing to do.")
		return nil
	}
	fmt.Printf("\nWill delete app on org %s: \n", c.Org)
	for _, app := range toDelete {
		fmt.Printf("\t- %s ( cf curl /v2/apps/%s )\n", app.Name, app.Guid)
	}

	if !c.Force {
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("\nPlease confirm apply by typing 'yes': ")
		confirm, _ := reader.ReadString('\n')
		confirm = strings.TrimSpace(confirm)
		if confirm != "yes" {
			fmt.Println("Not apply !")
			return nil
		}
	}

	for _, app := range toDelete {
		RunParallel(app.Guid, func(meta interface{}) error {
			return client.DeleteApp(meta.(string))
		})
	}
	errors := WaitParallel()
	if len(errors) > 0 {
		fmt.Println("There is error when deleting apps:")
		for _, err := range errors {
			fmt.Printf("\t- %s\n", err.Error())
		}
	}
	fmt.Println("Modifications has been applied.")
	return nil
}

func init() {
	desc := `Delete all apps which has been created after a period of time in an org`
	_, err := parser.AddCommand(
		"org-limiter",
		desc,
		desc,
		&orgLimit)
	if err != nil {
		panic(err)
	}
}
