package main

import (
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3/constant"
	clients "github.com/cloudfoundry-community/go-cf-clients-helper/v2"
)

type Session struct {
	clients.Session
}

var large = ccv3.Query{
	Key:    ccv3.PerPage,
	Values: []string{"5000"},
}


var orderByTimestampDesc = ccv3.Query{
	Key:    ccv3.OrderBy,
	Values: []string{"-created_at"},
}

var session *Session

func getSession() (*Session, error) {
	if session != nil {
		return session, nil
	}
	c := clients.Config{
		Endpoint:          options.Api,
		CFClientID:        options.ClientID,
		CFClientSecret:    options.ClientSecret,
		User:              options.Username,
		Password:          options.Password,
		SkipSslValidation: options.SkipSSLValidation,
	}
	s, err := clients.NewSession(c)
	if err != nil {
		return nil, err
	}
	return &Session{*s}, nil
}

type Application struct {
	GUID      string                    `json:"guid,omitempty"`
	Name      string                    `json:"name,omitempty"`
	State     constant.ApplicationState `json:"state,omitempty"`
	CreatedAt string                    `json:"created_at,omitempty"`
	UpdatedAt string                    `json:"updated_at,omitempty"`
}


type Route struct {
	GUID      string `json:"guid,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}


type ServiceInstance struct {
	GUID      string `json:"guid,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

func (d *Session) ExtGetApplications(query ...ccv3.Query) ([]Application, error) {
	res := []Application{}
	_, _, err := d.V3().MakeListRequest(ccv3.RequestParams{
		RequestName:  "GetApplications",
		Query:        query,
		ResponseBody: Application{},
		AppendToList: func(item interface{}) error {
			res = append(res, item.(Application))
			return nil
		},
	})
	return res, err
}

func (d *Session) ExtGetRoutes(query ...ccv3.Query) ([]Route, error) {
	res := []Route{}
	_, _, err := d.V3().MakeListRequest(ccv3.RequestParams{
		RequestName:  "GetRoutes",
		Query:        query,
		ResponseBody: Route{},
		AppendToList: func(item interface{}) error {
			res = append(res, item.(Route))
			return nil
		},
	})
	return res, err
}


func (d *Session) ExtGetServiceInstances(query ...ccv3.Query) ([]ServiceInstance, error) {
	res := []ServiceInstance{}
	_, _, err := d.V3().MakeListRequest(ccv3.RequestParams{
		RequestName:  "GetServiceInstances",
		Query:        query,
		ResponseBody: ServiceInstance{},
		AppendToList: func(item interface{}) error {
			res = append(res, item.(ServiceInstance))
			return nil
		},
	})
	return res, err
}
