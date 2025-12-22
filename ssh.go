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

type ValidateSSHSpace struct {
	SshLimit     Duration `short:"l" long:"ssh-time-limit" description:"Define when an enabled ssh should be set as disabled" default:"24h"`
	Force        bool     `short:"f" long:"force" description:"Will apply directly deactivation of ssh without confirmation"`
	IgnoreSpaces []string `alias:"is" long:"ignore-space" description:"Ignore space ids or by name in format of <org name>/<space name>"`
}

var validateSSHSpace ValidateSSHSpace

type SSHSpaceMeta struct {
	space         resources.Space
	session       *Session
	deactivateMap cmap.ConcurrentMap
	limit         time.Duration
}

type SSHSpaceApplyResult struct {
	space   resources.Space
	message string
}

func (c *ValidateSSHSpace) Execute(_ []string) error {
	initParallel()
	sess, err := getSession()
	if err != nil {
		return err
	}

	ignoredSpaces, err := findSpaces(c.IgnoreSpaces)
	if err != nil {
		return err
	}

	spaces, _, _, err := sess.V3().GetSpaces(large)
	if err != nil {
		return err
	}
	deactivateMap := cmap.New()
	for _, space := range spaces {
		enabled, _, err := sess.V3().GetSpaceFeature(space.GUID, "ssh")
		if err != nil {
			return err
		}
		if !enabled {
			continue
		}
		if funk.ContainsString(ignoredSpaces, space.GUID) {
			continue
		}
		sshSpaceMeta := SSHSpaceMeta{
			space:         space,
			session:       sess,
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
	for _, v := range deactivateMap.Items() {
		result := v.(SSHSpaceApplyResult)
		RunParallel(result.space, func(meta interface{}) error {
			space := meta.(resources.Space)
			_, err := sess.V3().UpdateSpaceFeature(space.GUID, false, "ssh")
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
	sess := sshSpaceMeta.session
	deactivateMap := sshSpaceMeta.deactivateMap

	events, _, err := sess.V3().GetEvents(
		orderByTimestampDesc,
		ccv3.Query{
			Key:    "types",
			Values: []string{"audit.app.ssh-authorized"},
		},
		ccv3.Query{
			Key:    ccv3.SpaceGUIDFilter,
			Values: []string{space.GUID},
		},
	)

	if err != nil {
		return err
	}
	if len(events) == 0 {
		deactivateMap.Set(space.GUID, SSHSpaceApplyResult{
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
	originAt := events[0].CreatedAt
	at := originAt.In(time.Local).Add(sshSpaceMeta.limit)
	if !time.Now().After(at) {
		return nil
	}
	deactivateMap.Set(space.GUID, SSHSpaceApplyResult{
		space: space,
		message: messages.C.Sprintf("Space '%s' -> Last connexion at %s by %s, %s",
			messages.C.Cyan(space.Name),
			messages.C.Red(originAt.Format(time.RFC850)),
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
