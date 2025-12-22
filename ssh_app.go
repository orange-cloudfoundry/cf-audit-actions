package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"code.cloudfoundry.org/cli/v8/api/cloudcontroller/ccv3"
	"code.cloudfoundry.org/cli/v8/resources"
	"github.com/orange-cloudfoundry/cf-audit-actions/messages"
	"github.com/orcaman/concurrent-map"
	"github.com/thoas/go-funk"
)

type ValidateSSHApp struct {
	SshLimit     Duration `short:"l" long:"ssh-time-limit" description:"Define when an enabled ssh should be set as disabled" default:"24h"`
	Force        bool     `short:"f" long:"force" description:"Will apply directly deactivation of ssh without confirmation"`
	IgnoreSpaces []string `alias:"is" long:"ignore-space" description:"Ignore space ids or by name in format of <org name>/<space name or *>"`
}

var validateSSHApp ValidateSSHApp

type SSHAppMeta struct {
	app           resources.Application
	session       *Session
	deactivateMap cmap.ConcurrentMap
	limit         time.Duration
}

type SSHAppApplyResult struct {
	message string
}

func (c *ValidateSSHApp) Execute(_ []string) error {
	initParallel()
	sess, err := getSession()
	if err != nil {
		return err
	}

	ignoredSpaces, err := findSpaces(c.IgnoreSpaces)
	if err != nil {
		return err
	}
	apps, _, err := sess.V3().GetApplications()
	if err != nil {
		return err
	}
	deactivateMap := cmap.New()
	for _, app := range apps {
		enabled, _, err := sess.V3().GetAppFeature(app.GUID, "ssh")
		if err != nil {
			return err
		}
		if !enabled.Enabled {
			continue
		}
		if funk.ContainsString(ignoredSpaces, app.SpaceGUID) {
			continue
		}
		sshAppMeta := SSHAppMeta{
			app:           app,
			session:       sess,
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
			_, err := sess.V3().UpdateAppFeature(meta.(string), false, "ssh")
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
	fmt.Println("Modifications has been applied.")
	return nil
}

func findSshAppDeactivate(meta interface{}) error {
	sshSpaceMeta := meta.(SSHAppMeta)
	app := sshSpaceMeta.app
	sess := sshSpaceMeta.session
	deactivateMap := sshSpaceMeta.deactivateMap

	events, _, err := sess.V3().GetEvents(
		orderByTimestampDesc,
		ccv3.Query{
			Key:    "types",
			Values: []string{"audit.app.ssh-authorized"},
		},
		ccv3.Query{
			Key:    ccv3.TargetGUIDFilter,
			Values: []string{app.GUID},
		},
	)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		deactivateMap.Set(app.GUID, SSHAppApplyResult{
			message: messages.C.Sprintf(
				"App '%s' will %s because there is no connexion in ssh from a long time",
				messages.C.Cyan(app.Name),
				messages.C.BgRed("deactivate ssh"),
			),
		})
		return nil
	}
	event := events[0]
	originAt := event.CreatedAt
	at := originAt.In(time.Local).Add(sshSpaceMeta.limit)
	if !time.Now().After(at) {
		return nil
	}
	deactivateMap.Set(app.GUID, SSHAppApplyResult{
		message: messages.C.Sprintf("App '%s' -> Last connexion at %s by %s, %s",
			messages.C.Cyan(app.Name),
			messages.C.Red(originAt.Format(time.RFC850)),
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

type AppUpdateSSH struct {
	EnableSSH bool `json:"enable_ssh"`
}
