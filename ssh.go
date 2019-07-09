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

type ValidateSSHSpace struct {
	SshLimit     SshLimit `short:"l" long:"ssh-time-limit" description:"Define when an enabled ssh should be set as disabled" default:"24h"`
	Force        bool     `short:"f" long:"force" description:"Will apply directly deactivation of ssh without confirmation"`
	IgnoreSpaces []string `alias:"is" long:"ignore-space" description:"Ignore space ids or by name in format of <org name>/<space name>"`
}

type SshLimit time.Duration

func (limit *SshLimit) UnmarshalFlag(value string) error {
	duration, err := time.ParseDuration(value)
	if err != nil {
		return err
	}
	*limit = SshLimit(duration)
	return nil
}

var validateSSHSpace ValidateSSHSpace

type SSHSpaceMeta struct {
	space         cfclient.Space
	client        *cfclient.Client
	deactivateMap cmap.ConcurrentMap
	limit         time.Duration
}

type SSHSpaceApplyResult struct {
	space   cfclient.Space
	message string
}

func (c *ValidateSSHSpace) Execute(_ []string) error {
	initParallel()
	client, err := retrieveClient()
	if err != nil {
		return err
	}

	ignoredSpaces, err := findSpaces(c.IgnoreSpaces)
	if err != nil {
		return err
	}

	spaces, err := client.ListSpaces()
	if err != nil {
		return err
	}
	deactivateMap := cmap.New()
	for _, space := range spaces {
		if !space.AllowSSH {
			continue
		}
		if funk.ContainsString(ignoredSpaces, space.Guid) {
			continue
		}
		sshSpaceMeta := SSHSpaceMeta{
			space:         space,
			client:        client,
			deactivateMap: deactivateMap,
			limit:         time.Duration(c.SshLimit),
		}
		RunParallel(sshSpaceMeta, findSshSpaceDeactivate)
	}
	errors := WaitParallel()
	if len(errors) > 0 {
		fmt.Println("There is error when finding events on spaces:")
		for _, err := range errors {
			fmt.Printf("\t- %s\n", err.Error())
		}
	}
	fmt.Println("\nWill apply:")
	for _, v := range deactivateMap.Items() {
		result := v.(SSHSpaceApplyResult)
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
	for k, v := range deactivateMap.Items() {
		result := v.(SSHSpaceApplyResult)
		RunParallel(result.space, func(meta interface{}) error {
			space := meta.(cfclient.Space)
			_, err := client.UpdateSpace(k, cfclient.SpaceRequest{
				Name:     space.Name,
				AllowSSH: false,
			})
			if err != nil {
				return err
			}
			return nil
		})

	}
	errors = WaitParallel()
	if len(errors) > 0 {
		fmt.Println("There is error when updating spaces:")
		for _, err := range errors {
			fmt.Printf("\t- %s\n", err.Error())
		}
	}
	fmt.Println("Modifications has been applied.")
	return nil
}

func findSshSpaceDeactivate(meta interface{}) error {
	sshSpaceMeta := meta.(SSHSpaceMeta)
	space := sshSpaceMeta.space
	client := sshSpaceMeta.client
	deactivateMap := sshSpaceMeta.deactivateMap
	events, err := client.ListEventsByQuery(url.Values{
		"q": []string{
			fmt.Sprintf("space_guid:%s", space.Guid),
			fmt.Sprintf("type:audit.app.ssh-authorized"),
		},
		"order-by":        []string{"timestamp"},
		"order-direction": []string{"desc"},
	})
	if err != nil {
		return err
	}
	if len(events) == 0 {
		deactivateMap.Set(space.Guid, SSHSpaceApplyResult{
			space: space,
			message: messages.C.Sprintf(
				"Space '%s' will %s because there is no connexion in ssh from a long time",
				messages.C.Cyan(space.Name),
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
	deactivateMap.Set(space.Guid, SSHSpaceApplyResult{
		space: space,
		message: messages.C.Sprintf("Space '%s' -> Last connexion at %s by %s with email %s, %s",
			messages.C.Cyan(space.Name),
			messages.C.Red(at.Format(time.RFC850)),
			messages.C.Green(event.ActorUsername),
			messages.C.Green(event.ActorName),
			messages.C.BgRed("ssh will be deactivate"),
		),
	})
	return nil
}

func init() {
	desc := `Check if ssh is enabled in spaces and deactivate it if it reach the time limit`
	_, err := parser.AddCommand(
		"ssh",
		desc,
		desc,
		&validateSSHSpace)
	if err != nil {
		panic(err)
	}
}
