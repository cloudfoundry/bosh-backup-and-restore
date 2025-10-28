package mockbosh

import (
	"github.com/cloudfoundry/bosh-backup-and-restore/internal/cf-webmock/mockhttp"
)

type getDeploymentMock struct {
	*mockhttp.MockHttp
}

type deployment struct {
	Name string `json:"name"`
}

func GetDeployments() *getDeploymentMock {
	return &getDeploymentMock{
		MockHttp: mockhttp.NewMockedHttpRequest("GET", "/deployments"),
	}
}

func (d *getDeploymentMock) RespondsWithListOfDeployments(deployments []string) *mockhttp.MockHttp {
	var deps []deployment

	for _, dep := range deployments {
		deps = append(deps, deployment{Name: dep})
	}
	return d.RespondsWithJson(deps)
}
