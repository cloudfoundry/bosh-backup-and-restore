package mockbosh

import (
	"fmt"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/internal/cf-webmock/mockhttp"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"
)

type manifestMock struct {
	*mockhttp.MockHttp
}

func GetDeployment(deploymentName string) *manifestMock {
	mock := &manifestMock{MockHttp: mockhttp.NewMockedHttpRequest("GET", fmt.Sprintf("/deployments/%s", deploymentName))}
	return mock
}

func Manifest(deploymentName string) *manifestMock {
	return GetDeployment(deploymentName)
}

func (t *manifestMock) RespondsWith(manifest []byte) *mockhttp.MockHttp {
	data := map[string]string{"manifest": string(manifest)}
	return t.RespondsWithJson(data)
}

func (t *manifestMock) RespondsWithManifest(manifest interface{}) *mockhttp.MockHttp {
	data, err := yaml.Marshal(manifest)
	Expect(err).NotTo(HaveOccurred())
	return t.RespondsWith(data)
}
