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
			err := cleanupAppRoutes(meta.(string))
			if err != nil {
				return err
			}
			err = cleanupAppServices(meta.(string))
			if err != nil {
				return err
			}
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

	err = c.cleanupOrphanRoutes(org.Guid)
	if err != nil {
		fmt.Println("Error while cleanup orphan's routes:" + err.Error())
	}
	err = c.cleanupOrphanServices(org.Guid)
	if err != nil {
		fmt.Println("Error while cleanup orphan's service instance:" + err.Error())
	}

	fmt.Println("Modifications has been applied.")
	return nil
}

func cleanupAppRoutes(appGUID string) error {
	client, err := retrieveClient()
	if err != nil {
		return err
	}
	routeMappings, err := client.ListRouteMappingsByQuery(url.Values{
		"q": []string{"app_guid:" + appGUID},
	})
	if err != nil {
		return err
	}

	for _, routeMapping := range routeMappings {
		err := client.DeleteRouteMapping(routeMapping.Guid)
		if err != nil {
			return err
		}
	}

	return nil
}

func cleanupAppServices(appGUID string) error {
	client, err := retrieveClient()
	if err != nil {
		return err
	}
	serviceBindings, err := client.ListServiceBindingsByQuery(url.Values{
		"q": []string{"app_guid:" + appGUID},
	})
	if err != nil {
		return err
	}

	for _, serviceBinding := range serviceBindings {
		err := client.DeleteServiceBinding(serviceBinding.Guid)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *OrgLimit) cleanupOrphanRoutes(orgGUID string) error {
	initParallel()
	client, err := retrieveClient()
	if err != nil {
		return err
	}
	routes, err := client.ListRoutesByQuery(url.Values{
		"q": []string{"organization_guid:" + orgGUID},
	})
	if err != nil {
		return err
	}
	for _, route := range routes {
		createdAt, _ := time.Parse(time.RFC3339, route.CreatedAt)
		if createdAt.Add(time.Duration(c.TimeLimit)).Before(time.Now()) {
			RunParallel(route.Guid, func(meta interface{}) error {
				result, err := client.ListAppsByRoute((meta.(string)))
				if err != nil {
					return err
				}
				if len(result) == 0 {
					return client.DeleteRoute(meta.(string))
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
	client, err := retrieveClient()
	if err != nil {
		return err
	}
	serviceInstances, err := client.ListServiceInstancesByQuery(url.Values{
		"q": []string{"organization_guid:" + orgGUID},
	})
	if err != nil {
		return err
	}
	for _, serviceInstance := range serviceInstances {
		createdAt, _ := time.Parse(time.RFC3339, serviceInstance.CreatedAt)
		if createdAt.Add(time.Duration(c.TimeLimit)).Before(time.Now()) {
			RunParallel(serviceInstance.Guid, func(meta interface{}) error {
				result, err := client.ListServiceBindingsByQuery(url.Values{
					"q": []string{"service_instance_guid:" + (meta.(string))},
				})
				if err != nil {
					return err
				}
				if len(result) == 0 {
					return client.DeleteServiceInstance(meta.(string), true, true)
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
