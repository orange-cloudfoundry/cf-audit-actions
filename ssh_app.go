package main

import (
	"bufio"
	"fmt"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/orange-cloudfoundry/cf-audit-actions/messages"
	"github.com/orcaman/concurrent-map"
	"github.com/thoas/go-funk"
	"net/url"
	"os"
	"strings"
	"time"
)

type ValidateSSHApp struct {
	SshLimit     SshLimit `short:"l" long:"ssh-time-limit" description:"Define when an enabled ssh should be set as disabled" default:"24h"`
	Force        bool     `short:"f" long:"force" description:"Will apply directly deactivation of ssh without confirmation"`
	IgnoreSpaces []string `alias:"is" long:"ignore-space" description:"Ignore space ids or by name in format of <org name>/<space name or *>"`
}

var validateSSHApp ValidateSSHApp

type SSHAppMeta struct {
	app           cfclient.App
	client        *cfclient.Client
	deactivateMap cmap.ConcurrentMap
	limit         time.Duration
}

type SSHAppApplyResult struct {
	message string
}

func (c *ValidateSSHApp) Execute(_ []string) error {
	initParallel()
	client, err := retrieveClient()
	if err != nil {
		return err
	}

	ignoredSpaces, err := findSpaces(c.IgnoreSpaces)
	if err != nil {
		return err
	}
	apps, err := client.ListApps()
	if err != nil {
		return err
	}
	deactivateMap := cmap.New()
	for _, app := range apps {
		if !app.EnableSSH {
			continue
		}
		if funk.ContainsString(ignoredSpaces, app.SpaceGuid) {
			continue
		}
		sshAppMeta := SSHAppMeta{
			app:           app,
			client:        client,
			deactivateMap: deactivateMap,
			limit:         time.Duration(c.SshLimit),
		}
		RunParallel(sshAppMeta, findSshAppDeactivate)
	}
	errors := WaitParallel()
	if len(errors) > 0 {
		fmt.Println("There is error when finding events on apps:")
		for _, err := range errors {
			fmt.Printf("\t- %s\n", err.Error())
		}
	}
	fmt.Println("\nWill apply:")
	for _, v := range deactivateMap.Items() {
		result := v.(SSHAppApplyResult)
		fmt.Printf("\t- %s\n", result.message)
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
	for k := range deactivateMap.Items() {
		RunParallel(k, func(meta interface{}) error {
			_, err := client.UpdateApp(meta.(string), cfclient.AppUpdateResource{
				EnableSSH: false,
			})
			if err != nil {
				return err
			}
			return nil
		})
	}
	errors = WaitParallel()
	if len(errors) > 0 {
		fmt.Println("There is error when updating apps:")
		for _, err := range errors {
			fmt.Printf("\t- %s\n", err.Error())
		}
	}
	return nil
}

func findSshAppDeactivate(meta interface{}) error {
	sshSpaceMeta := meta.(SSHAppMeta)
	app := sshSpaceMeta.app
	client := sshSpaceMeta.client
	deactivateMap := sshSpaceMeta.deactivateMap
	events, err := client.ListEventsByQuery(url.Values{
		"q": []string{
			fmt.Sprintf("actee:%s", app.Guid),
			fmt.Sprintf("type:audit.app.ssh-authorized"),
		},
		"order-by":        []string{"timestamp"},
		"order-direction": []string{"desc"},
	})
	if err != nil {
		return err
	}
	if len(events) == 0 {
		deactivateMap.Set(app.Guid, SSHAppApplyResult{
			message: messages.C.Sprintf(
				"App '%s' will %s because there is no connexion in ssh from a long time",
				messages.C.Cyan(app.Name),
				messages.C.BgRed("deactivate ssh"),
			),
		})
		return nil
	}
	event := events[0]
	at, _ := time.Parse(time.RFC3339, events[0].CreatedAt)
	at = at.In(time.Local).Add(sshSpaceMeta.limit)
	if !time.Now().After(at) {
		return nil
	}
	deactivateMap.Set(app.Guid, SSHAppApplyResult{
		message: messages.C.Sprintf("App '%s' -> Last connexion at %s by %s with email %s, %s",
			messages.C.Cyan(app.Name),
			messages.C.Red(at.Format(time.RFC850)),
			messages.C.Green(event.ActorUsername),
			messages.C.Green(event.ActorName),
			messages.C.BgRed("ssh will be deactivate"),
		),
	})
	return nil
}

func init() {
	desc := `Check if ssh is enabled in apps and deactivate it if it reach the time limit`
	_, err := parser.AddCommand(
		"ssh-app",
		desc,
		desc,
		&validateSSHApp)
	if err != nil {
		panic(err)
	}
}
