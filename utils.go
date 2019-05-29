package main

import (
	"fmt"
	"github.com/orange-cloudfoundry/cf-audit-actions/messages"
	"strings"
)

func findSpaces(idOrPairs []string) ([]string, error) {
	ids := make([]string, 0)
	for _, idOrPair := range idOrPairs {
		id, err := findSpace(idOrPair)
		if err != nil {
			return []string{}, fmt.Errorf(messages.C.Sprintf(
				"Error found when finding space '%s'",
				messages.C.Cyan(idOrPair),
			))
		}
		ids = append(ids, id...)
	}
	return ids, nil
}

func findSpace(idOrPair string) ([]string, error) {
	pairSplit := strings.SplitN(idOrPair, "/", 2)
	if len(pairSplit) == 1 {
		return []string{idOrPair}, nil
	}
	client, err := retrieveClient()
	if err != nil {
		return []string{idOrPair}, err
	}
	org, err := client.GetOrgByName(pairSplit[0])
	if err != nil {
		return []string{idOrPair}, err
	}
	if pairSplit[1] != "*" {
		space, err := client.GetSpaceByName(pairSplit[1], org.Guid)
		if err != nil {
			return []string{idOrPair}, err
		}
		return []string{space.Guid}, nil
	}
	spaces, err := client.OrgSpaces(org.Guid)
	if err != nil {
		return []string{idOrPair}, err
	}
	ids := make([]string, len(spaces))
	for i, space := range spaces {
		ids[i] = space.Guid
	}
	return ids, nil
}
