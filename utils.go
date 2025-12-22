package main

import (
	"fmt"
	"strings"

	"code.cloudfoundry.org/cli/v8/api/cloudcontroller/ccv3"
	"github.com/orange-cloudfoundry/cf-audit-actions/messages"
)

func findSpaces(idOrPairs []string) ([]string, error) {
	ids := make([]string, 0)
	for _, idOrPair := range idOrPairs {
		id, err := findSpace(idOrPair)
		if err != nil {
			errorMessage := messages.C.Sprintf(
				"error found when finding space '%s'",
				messages.C.Cyan(idOrPair),
			)
			return []string{}, fmt.Errorf("%s", errorMessage)
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
	sess, err := getSession()
	if err != nil {
		return []string{idOrPair}, err
	}
	orgs, _, err := sess.V3().GetOrganizations(ccv3.Query{
		Key:    ccv3.NameFilter,
		Values: []string{pairSplit[0]},
	})
	if err != nil {
		return []string{idOrPair}, err
	}
	if len(orgs) != 1 {
		return []string{idOrPair}, fmt.Errorf("multiple match for org name %s", pairSplit[0])
	}

	if pairSplit[1] != "*" {
		spaces, _, _, err := sess.V3().GetSpaces(
			ccv3.Query{Key: ccv3.OrganizationGUIDFilter, Values: []string{orgs[0].GUID}},
			ccv3.Query{Key: ccv3.NameFilter, Values: []string{pairSplit[1]}},
		)
		if err != nil {
			return []string{idOrPair}, err
		}
		if len(spaces) != 1 {
			return []string{idOrPair}, fmt.Errorf("multiple match for space name %s in org %s", pairSplit[1], orgs[0].GUID)
		}
		return []string{spaces[0].GUID}, nil
	}
	spaces, _, _, err := sess.V3().GetSpaces(large, ccv3.Query{
		Key:    ccv3.OrganizationGUIDFilter,
		Values: []string{orgs[0].GUID},
	})
	if err != nil {
		return []string{idOrPair}, err
	}

	ids := make([]string, len(spaces))
	for i, space := range spaces {
		ids[i] = space.GUID
	}
	return ids, nil
}
