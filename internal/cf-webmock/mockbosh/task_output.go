package mockbosh

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/cloudfoundry/bosh-backup-and-restore/internal/cf-webmock/mockhttp"
	. "github.com/onsi/gomega" //nolint:staticcheck
)

type VMsOutput struct {
	IPs       []string
	JobName   string `json:"job_name"`
	Index     *int   `json:"index"`
	ID        string `json:"id"`
	Bootstrap bool   `json:"bootstrap"`
}

type taskOutputMock struct {
	*mockhttp.MockHttp
}

func TaskEvent(taskId int) *taskOutputMock {
	mock := &taskOutputMock{MockHttp: mockhttp.NewMockedHttpRequest("GET", fmt.Sprintf("/tasks/%d/output?type=event", taskId))}
	return mock
}

func TaskOutput(taskId int) *taskOutputMock {
	mock := &taskOutputMock{MockHttp: mockhttp.NewMockedHttpRequest("GET", fmt.Sprintf("/tasks/%d/output?type=result", taskId))}
	return mock
}

func (t *taskOutputMock) RespondsWithVMsOutput(vms interface{}) *mockhttp.MockHttp {
	output := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(output)

	for _, line := range interfaceSlice(vms) {
		Expect(encoder.Encode(line)).ToNot(HaveOccurred())
	}

	return t.RespondsWith(string(output.Bytes())) //nolint:staticcheck
}

func (t *taskOutputMock) RespondsWithTaskOutput(taskOutput interface{}) *mockhttp.MockHttp {
	output := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(output)

	for _, line := range interfaceSlice(taskOutput) {
		Expect(encoder.Encode(line)).ToNot(HaveOccurred())
	}

	return t.RespondsWith(string(output.Bytes())) //nolint:staticcheck
}

func interfaceSlice(slice interface{}) []interface{} {
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		panic("needs to be called with a slice type")
	}

	ret := make([]interface{}, s.Len())

	for i := 0; i < s.Len(); i++ {
		ret[i] = s.Index(i).Interface()
	}

	return ret
}
