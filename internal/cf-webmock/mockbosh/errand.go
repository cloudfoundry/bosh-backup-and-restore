package mockbosh

import (
	"fmt"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/internal/cf-webmock/mockhttp"
)

type errandMock struct {
	*mockhttp.MockHttp
}

func Errand(deploymentName, errandName string) *errandMock {
	return &errandMock{
		MockHttp: mockhttp.NewMockedHttpRequest("POST", fmt.Sprintf("/deployments/%s/errands/%s/runs", deploymentName, errandName)),
	}
}

func (e *errandMock) RedirectsToTask(taskID int) *mockhttp.MockHttp {
	return e.RedirectsTo(taskURL(taskID))
}
