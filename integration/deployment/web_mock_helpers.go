package deployment

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/internal/cf-webmock/mockbosh"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/internal/cf-webmock/mockhttp"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/testcluster"
)

func MockDirectorWith(director *mockhttp.Server, info mockhttp.MockedResponseBuilder, vmsResponse []mockhttp.MockedResponseBuilder, manifestResponse []mockhttp.MockedResponseBuilder, sshResponse []mockhttp.MockedResponseBuilder, cleanupResponse []mockhttp.MockedResponseBuilder) {
	director.VerifyAndMock(AppendBuilders(
		[]mockhttp.MockedResponseBuilder{info},
		vmsResponse,
		manifestResponse,
		sshResponse,
		cleanupResponse,
	)...)
}

func InfoWithBasicAuth() []mockhttp.MockedResponseBuilder {
	return []mockhttp.MockedResponseBuilder{
		mockbosh.Info().WithAuthTypeBasic(),
	}
}

func InfoWithBasicAuthFails(errorMessage string) []mockhttp.MockedResponseBuilder {
	return []mockhttp.MockedResponseBuilder{
		mockbosh.Info().WithAuthTypeBasic().Fails(errorMessage),
	}
}

func Deployments(deployments []string) []mockhttp.MockedResponseBuilder {
	return []mockhttp.MockedResponseBuilder{
		mockbosh.GetDeployments().RespondsWithListOfDeployments(deployments),
	}
}

func DeploymentsFails(errorMessage string) []mockhttp.MockedResponseBuilder {
	return []mockhttp.MockedResponseBuilder{
		mockbosh.GetDeployments().Fails(errorMessage),
	}
}

func VmsForDeployment(deploymentName string, responseInstances []mockbosh.VMsOutput) []mockhttp.MockedResponseBuilder {
	randomTaskID := generateTaskId()
	return []mockhttp.MockedResponseBuilder{
		mockbosh.VMsForDeployment(deploymentName).RedirectsToTask(randomTaskID),
		mockbosh.Task(randomTaskID).RespondsWithTaskContainingState(mockbosh.TaskDone),
		mockbosh.Task(randomTaskID).RespondsWithTaskContainingState(mockbosh.TaskDone),
		mockbosh.TaskEvent(randomTaskID).RespondsWithVMsOutput([]string{}),
		mockbosh.TaskOutput(randomTaskID).RespondsWithVMsOutput(responseInstances),
	}
}

func VmsForDeploymentFails(deploymentName string) []mockhttp.MockedResponseBuilder {
	return []mockhttp.MockedResponseBuilder{
		mockbosh.VMsForDeployment(deploymentName).Fails("director unreachable"),
	}
}

func DownloadManifest(deploymentName string, manifest string) []mockhttp.MockedResponseBuilder {
	return []mockhttp.MockedResponseBuilder{
		mockbosh.Manifest(deploymentName).RespondsWith([]byte(manifest)),
	}
}

func AppendBuilders(arrayOfArrayOfBuilders ...[]mockhttp.MockedResponseBuilder) []mockhttp.MockedResponseBuilder {
	var flattenedArrayOfBuilders []mockhttp.MockedResponseBuilder
	for _, arrayOfBuilders := range arrayOfArrayOfBuilders {
		flattenedArrayOfBuilders = append(flattenedArrayOfBuilders, arrayOfBuilders...)
	}
	return flattenedArrayOfBuilders
}

func SetupSSH(deploymentName, instanceGroup, instanceID string, instanceIndex int, instance *testcluster.Instance) []mockhttp.MockedResponseBuilder {
	vmOutput := mockbosh.VMsOutput{
		ID:    instanceID,
		Index: &instanceIndex,
	}

	return SetupSSHForAllInstances(
		deploymentName,
		instanceGroup,
		[]mockbosh.VMsOutput{vmOutput},
		[]*testcluster.Instance{instance},
	)
}

func SetupSSHForAllInstances(deploymentName string, instanceGroup string, vmsOutputs []mockbosh.VMsOutput, instances []*testcluster.Instance) []mockhttp.MockedResponseBuilder {
	randomTaskID := generateTaskId()

	buildResponse := func() string {
		var response []string
		for i, vm := range vmsOutputs {
			response = append(response, fmt.Sprintf(`{"status":"success",
"ip":"%s",
"host_public_key":"%s",
"id":"%s",
"index":%d}`,
				instances[i].Address(),
				instances[i].HostPublicKey(),
				vm.ID,
				vm.Index,
			))
		}
		return fmt.Sprintf("[%s]", strings.Join(response, ","))
	}

	return []mockhttp.MockedResponseBuilder{
		mockbosh.StartSSHSession(deploymentName).SetSSHResponseCallback(func(username, key string) {
			for _, instance := range instances {
				instance.CreateUser(username, key)
			}
		}).ForInstanceGroup(instanceGroup).RedirectsToTask(randomTaskID),
		mockbosh.Task(randomTaskID).RespondsWithTaskContainingState(mockbosh.TaskDone),
		mockbosh.Task(randomTaskID).RespondsWithTaskContainingState(mockbosh.TaskDone),
		mockbosh.TaskEvent(randomTaskID).RespondsWith("{}"),
		mockbosh.TaskOutput(randomTaskID).RespondsWith(buildResponse()),
	}
}

func CleanupSSH(deploymentName, instanceGroup string) []mockhttp.MockedResponseBuilder {
	randomTaskID := generateTaskId()
	return []mockhttp.MockedResponseBuilder{
		mockbosh.CleanupSSHSession(deploymentName).ForInstanceGroup(instanceGroup).RedirectsToTask(randomTaskID),
		mockbosh.Task(randomTaskID).RespondsWithTaskContainingState(mockbosh.TaskDone),
	}
}

func CleanupSSHFails(deploymentName, instanceGroup, errorMessage string) []mockhttp.MockedResponseBuilder {
	return []mockhttp.MockedResponseBuilder{
		mockbosh.CleanupSSHSession(deploymentName).ForInstanceGroup(instanceGroup).Fails(errorMessage),
	}
}

func generateTaskId() int {
	return rand.Int()
}
