// Code generated by counterfeiter. DO NOT EDIT.
package fakes

import (
	"io"
	"sync"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh"
)

type FakeSSHConnection struct {
	RunStub        func(string) ([]byte, []byte, int, error)
	runMutex       sync.RWMutex
	runArgsForCall []struct {
		arg1 string
	}
	runReturns struct {
		result1 []byte
		result2 []byte
		result3 int
		result4 error
	}
	runReturnsOnCall map[int]struct {
		result1 []byte
		result2 []byte
		result3 int
		result4 error
	}
	StreamStub        func(string, io.Writer) ([]byte, int, error)
	streamMutex       sync.RWMutex
	streamArgsForCall []struct {
		arg1 string
		arg2 io.Writer
	}
	streamReturns struct {
		result1 []byte
		result2 int
		result3 error
	}
	streamReturnsOnCall map[int]struct {
		result1 []byte
		result2 int
		result3 error
	}
	StreamStdinStub        func(string, io.Reader) ([]byte, []byte, int, error)
	streamStdinMutex       sync.RWMutex
	streamStdinArgsForCall []struct {
		arg1 string
		arg2 io.Reader
	}
	streamStdinReturns struct {
		result1 []byte
		result2 []byte
		result3 int
		result4 error
	}
	streamStdinReturnsOnCall map[int]struct {
		result1 []byte
		result2 []byte
		result3 int
		result4 error
	}
	UsernameStub        func() string
	usernameMutex       sync.RWMutex
	usernameArgsForCall []struct {
	}
	usernameReturns struct {
		result1 string
	}
	usernameReturnsOnCall map[int]struct {
		result1 string
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeSSHConnection) Run(arg1 string) ([]byte, []byte, int, error) {
	fake.runMutex.Lock()
	ret, specificReturn := fake.runReturnsOnCall[len(fake.runArgsForCall)]
	fake.runArgsForCall = append(fake.runArgsForCall, struct {
		arg1 string
	}{arg1})
	stub := fake.RunStub
	fakeReturns := fake.runReturns
	fake.recordInvocation("Run", []interface{}{arg1})
	fake.runMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2, ret.result3, ret.result4
	}
	return fakeReturns.result1, fakeReturns.result2, fakeReturns.result3, fakeReturns.result4
}

func (fake *FakeSSHConnection) RunCallCount() int {
	fake.runMutex.RLock()
	defer fake.runMutex.RUnlock()
	return len(fake.runArgsForCall)
}

func (fake *FakeSSHConnection) RunCalls(stub func(string) ([]byte, []byte, int, error)) {
	fake.runMutex.Lock()
	defer fake.runMutex.Unlock()
	fake.RunStub = stub
}

func (fake *FakeSSHConnection) RunArgsForCall(i int) string {
	fake.runMutex.RLock()
	defer fake.runMutex.RUnlock()
	argsForCall := fake.runArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeSSHConnection) RunReturns(result1 []byte, result2 []byte, result3 int, result4 error) {
	fake.runMutex.Lock()
	defer fake.runMutex.Unlock()
	fake.RunStub = nil
	fake.runReturns = struct {
		result1 []byte
		result2 []byte
		result3 int
		result4 error
	}{result1, result2, result3, result4}
}

func (fake *FakeSSHConnection) RunReturnsOnCall(i int, result1 []byte, result2 []byte, result3 int, result4 error) {
	fake.runMutex.Lock()
	defer fake.runMutex.Unlock()
	fake.RunStub = nil
	if fake.runReturnsOnCall == nil {
		fake.runReturnsOnCall = make(map[int]struct {
			result1 []byte
			result2 []byte
			result3 int
			result4 error
		})
	}
	fake.runReturnsOnCall[i] = struct {
		result1 []byte
		result2 []byte
		result3 int
		result4 error
	}{result1, result2, result3, result4}
}

func (fake *FakeSSHConnection) Stream(arg1 string, arg2 io.Writer) ([]byte, int, error) {
	fake.streamMutex.Lock()
	ret, specificReturn := fake.streamReturnsOnCall[len(fake.streamArgsForCall)]
	fake.streamArgsForCall = append(fake.streamArgsForCall, struct {
		arg1 string
		arg2 io.Writer
	}{arg1, arg2})
	stub := fake.StreamStub
	fakeReturns := fake.streamReturns
	fake.recordInvocation("Stream", []interface{}{arg1, arg2})
	fake.streamMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2, ret.result3
	}
	return fakeReturns.result1, fakeReturns.result2, fakeReturns.result3
}

func (fake *FakeSSHConnection) StreamCallCount() int {
	fake.streamMutex.RLock()
	defer fake.streamMutex.RUnlock()
	return len(fake.streamArgsForCall)
}

func (fake *FakeSSHConnection) StreamCalls(stub func(string, io.Writer) ([]byte, int, error)) {
	fake.streamMutex.Lock()
	defer fake.streamMutex.Unlock()
	fake.StreamStub = stub
}

func (fake *FakeSSHConnection) StreamArgsForCall(i int) (string, io.Writer) {
	fake.streamMutex.RLock()
	defer fake.streamMutex.RUnlock()
	argsForCall := fake.streamArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeSSHConnection) StreamReturns(result1 []byte, result2 int, result3 error) {
	fake.streamMutex.Lock()
	defer fake.streamMutex.Unlock()
	fake.StreamStub = nil
	fake.streamReturns = struct {
		result1 []byte
		result2 int
		result3 error
	}{result1, result2, result3}
}

func (fake *FakeSSHConnection) StreamReturnsOnCall(i int, result1 []byte, result2 int, result3 error) {
	fake.streamMutex.Lock()
	defer fake.streamMutex.Unlock()
	fake.StreamStub = nil
	if fake.streamReturnsOnCall == nil {
		fake.streamReturnsOnCall = make(map[int]struct {
			result1 []byte
			result2 int
			result3 error
		})
	}
	fake.streamReturnsOnCall[i] = struct {
		result1 []byte
		result2 int
		result3 error
	}{result1, result2, result3}
}

func (fake *FakeSSHConnection) StreamStdin(arg1 string, arg2 io.Reader) ([]byte, []byte, int, error) {
	fake.streamStdinMutex.Lock()
	ret, specificReturn := fake.streamStdinReturnsOnCall[len(fake.streamStdinArgsForCall)]
	fake.streamStdinArgsForCall = append(fake.streamStdinArgsForCall, struct {
		arg1 string
		arg2 io.Reader
	}{arg1, arg2})
	stub := fake.StreamStdinStub
	fakeReturns := fake.streamStdinReturns
	fake.recordInvocation("StreamStdin", []interface{}{arg1, arg2})
	fake.streamStdinMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2, ret.result3, ret.result4
	}
	return fakeReturns.result1, fakeReturns.result2, fakeReturns.result3, fakeReturns.result4
}

func (fake *FakeSSHConnection) StreamStdinCallCount() int {
	fake.streamStdinMutex.RLock()
	defer fake.streamStdinMutex.RUnlock()
	return len(fake.streamStdinArgsForCall)
}

func (fake *FakeSSHConnection) StreamStdinCalls(stub func(string, io.Reader) ([]byte, []byte, int, error)) {
	fake.streamStdinMutex.Lock()
	defer fake.streamStdinMutex.Unlock()
	fake.StreamStdinStub = stub
}

func (fake *FakeSSHConnection) StreamStdinArgsForCall(i int) (string, io.Reader) {
	fake.streamStdinMutex.RLock()
	defer fake.streamStdinMutex.RUnlock()
	argsForCall := fake.streamStdinArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeSSHConnection) StreamStdinReturns(result1 []byte, result2 []byte, result3 int, result4 error) {
	fake.streamStdinMutex.Lock()
	defer fake.streamStdinMutex.Unlock()
	fake.StreamStdinStub = nil
	fake.streamStdinReturns = struct {
		result1 []byte
		result2 []byte
		result3 int
		result4 error
	}{result1, result2, result3, result4}
}

func (fake *FakeSSHConnection) StreamStdinReturnsOnCall(i int, result1 []byte, result2 []byte, result3 int, result4 error) {
	fake.streamStdinMutex.Lock()
	defer fake.streamStdinMutex.Unlock()
	fake.StreamStdinStub = nil
	if fake.streamStdinReturnsOnCall == nil {
		fake.streamStdinReturnsOnCall = make(map[int]struct {
			result1 []byte
			result2 []byte
			result3 int
			result4 error
		})
	}
	fake.streamStdinReturnsOnCall[i] = struct {
		result1 []byte
		result2 []byte
		result3 int
		result4 error
	}{result1, result2, result3, result4}
}

func (fake *FakeSSHConnection) Username() string {
	fake.usernameMutex.Lock()
	ret, specificReturn := fake.usernameReturnsOnCall[len(fake.usernameArgsForCall)]
	fake.usernameArgsForCall = append(fake.usernameArgsForCall, struct {
	}{})
	stub := fake.UsernameStub
	fakeReturns := fake.usernameReturns
	fake.recordInvocation("Username", []interface{}{})
	fake.usernameMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeSSHConnection) UsernameCallCount() int {
	fake.usernameMutex.RLock()
	defer fake.usernameMutex.RUnlock()
	return len(fake.usernameArgsForCall)
}

func (fake *FakeSSHConnection) UsernameCalls(stub func() string) {
	fake.usernameMutex.Lock()
	defer fake.usernameMutex.Unlock()
	fake.UsernameStub = stub
}

func (fake *FakeSSHConnection) UsernameReturns(result1 string) {
	fake.usernameMutex.Lock()
	defer fake.usernameMutex.Unlock()
	fake.UsernameStub = nil
	fake.usernameReturns = struct {
		result1 string
	}{result1}
}

func (fake *FakeSSHConnection) UsernameReturnsOnCall(i int, result1 string) {
	fake.usernameMutex.Lock()
	defer fake.usernameMutex.Unlock()
	fake.UsernameStub = nil
	if fake.usernameReturnsOnCall == nil {
		fake.usernameReturnsOnCall = make(map[int]struct {
			result1 string
		})
	}
	fake.usernameReturnsOnCall[i] = struct {
		result1 string
	}{result1}
}

func (fake *FakeSSHConnection) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.runMutex.RLock()
	defer fake.runMutex.RUnlock()
	fake.streamMutex.RLock()
	defer fake.streamMutex.RUnlock()
	fake.streamStdinMutex.RLock()
	defer fake.streamStdinMutex.RUnlock()
	fake.usernameMutex.RLock()
	defer fake.usernameMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeSSHConnection) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ ssh.SSHConnection = new(FakeSSHConnection)
