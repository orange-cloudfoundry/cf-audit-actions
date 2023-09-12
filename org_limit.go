package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3"
)

type OrgLimit struct {
	TimeLimit Duration `short:"l" long:"time-limit" required:"1" description:"Define when an app should be deleted" default:"168h"`
	Force     bool     `short:"f" long:"force" description:"Will apply directly deactivation of ssh without confirmation"`
	Org       string   `short:"o" long:"org" required:"1" description:"Set organization to check"`
}

var orgLimit OrgLimit

func (c *OrgLimit) Execute(_ []string) error {
	initParallel()
	sess, err := getSession()
	if err != nil {
		return err
	}

	orgs, _, err := sess.V3().GetOrganizations(ccv3.Query{
		Key:    ccv3.NameFilter,
		Values: []string{c.Org},
	})
	if err != nil || len(orgs) == 0 {
		return err
	}

	apps, err := sess.ExtGetApplications(large, ccv3.Query{
		Key:    ccv3.OrganizationGUIDFilter,
		Values: []string{orgs[0].GUID},
	})
	if err != nil {
		return err
	}

	toDelete := make([]Application, 0)
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
		fmt.Printf("\t- %s ( cf curl /v2/apps/%s )\n", app.Name, app.GUID)
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
		RunParallel(app.GUID, func(meta interface{}) error {
			err := cleanupAppRoutes(meta.(string))
			if err != nil {
				return err
			}
			err = cleanupAppServices(meta.(string))
			if err != nil {
				return err
			}
			_, _, err = sess.V3().DeleteApplication(meta.(string))
			return err
		})
	}

	errors := WaitParallel()
	var lastError error
	if len(errors) > 0 {
		fmt.Println("There is error when deleting apps:")
		for _, err := range errors {
			fmt.Printf("\t- %s\n", err.Error())
			lastError = err
		}
	}

	err = c.cleanupOrphanRoutes(orgs[0].GUID)
	if err != nil {
		fmt.Println("Error while cleanup orphan's routes:" + err.Error())
		lastError = err
	}
	err = c.cleanupOrphanServices(orgs[0].GUID)
	if err != nil {
		fmt.Println("Error while cleanup orphan's service instance:" + err.Error())
		lastError = err
	}

	fmt.Println("Modifications has been applied.")
	return lastError
}

func cleanupAppRoutes(appGUID string) error {
	sess, err := getSession()
	if err != nil {
		return err
	}
	routeMappings, _, err := sess.V3().GetRoutes(large, ccv3.Query{
		Key:    ccv3.AppGUIDFilter,
		Values: []string{appGUID},
	})
	if err != nil {
		return err
	}

	for _, routeMapping := range routeMappings {
		_, _, err := sess.V3().DeleteRoute(routeMapping.GUID)
		if err != nil {
			return err
		}
	}

	return nil
}

func cleanupAppServices(appGUID string) error {
	sess, err := getSession()
	if err != nil {
		return err
	}
	serviceBindings, _, err := sess.V3().GetServiceCredentialBindings(large, ccv3.Query{
		Key:    ccv3.AppGUIDFilter,
		Values: []string{appGUID},
	})
	if err != nil {
		return err
	}

	for _, serviceBinding := range serviceBindings {
		_, _, err := sess.V3().DeleteServiceCredentialBinding(serviceBinding.GUID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *OrgLimit) cleanupOrphanRoutes(orgGUID string) error {
	initParallel()

	sess, err := getSession()
	if err != nil {
		return err
	}
	routes, err := sess.ExtGetRoutes(large, ccv3.Query{
		Key:    ccv3.OrganizationGUIDFilter,
		Values: []string{orgGUID},
	})
	if err != nil {
		return err
	}

	for _, route := range routes {
		createdAt, _ := time.Parse(time.RFC3339, route.CreatedAt)
		if createdAt.Add(time.Duration(c.TimeLimit)).Before(time.Now()) {
			RunParallel(route.GUID, func(meta interface{}) error {
				dests, _, err := sess.V3().GetRouteDestinations(meta.(string))
				if err != nil {
					return err
				}
				if len(dests) == 0 {
					_, _, err := sess.V3().DeleteRoute(meta.(string))
					return err
				}
				return nil
			})
		}
	}

	errors := WaitParallel()
	if len(errors) > 0 {
		fmt.Println("There is error when deleting route :")
		for _, err := range errors {
			fmt.Printf("\t- %s\n", err.Error())
		}
	}

	return nil
}

func (c *OrgLimit) cleanupOrphanServices(orgGUID string) error {
	initParallel()
	sess, err := getSession()
	if err != nil {
		return err
	}
	serviceInstances, err := sess.ExtGetServiceInstances(large, ccv3.Query{
		Key:    ccv3.OrganizationGUIDFilter,
		Values: []string{orgGUID},
	})
	if err != nil {
		return err
	}

	for _, serviceInstance := range serviceInstances {
		createdAt, _ := time.Parse(time.RFC3339, serviceInstance.CreatedAt)
		if createdAt.Add(time.Duration(c.TimeLimit)).Before(time.Now()) {
			RunParallel(serviceInstance.GUID, func(meta interface{}) error {
				result, _, err := sess.V3().GetServiceCredentialBindings(large, ccv3.Query{
					Key:    ccv3.ServiceInstanceGUIDFilter,
					Values: []string{meta.(string)},
				})
				if err != nil {
					return err
				}
				if len(result) == 0 {
					_, _, err = sess.V3().DeleteServiceInstance(meta.(string))
					return err
				}
				return nil
			})
		}
	}

	errors := WaitParallel()
	if len(errors) > 0 {
		fmt.Println("There is error when deleting service instance :")
		for _, err := range errors {
			fmt.Printf("\t- %s\n", err.Error())
		}
	}

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
