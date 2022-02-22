// Code generated by counterfeiter. DO NOT EDIT.
package fakes

import (
	"sync"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh"
	"github.com/cloudfoundry/bosh-cli/director"
	"github.com/cloudfoundry/bosh-utils/uuid"
)

type FakeSSHOptsGenerator struct {
	Stub        func(uuid.Generator) (director.SSHOpts, string, error)
	mutex       sync.RWMutex
	argsForCall []struct {
		arg1 uuid.Generator
	}
	returns struct {
		result1 director.SSHOpts
		result2 string
		result3 error
	}
	returnsOnCall map[int]struct {
		result1 director.SSHOpts
		result2 string
		result3 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeSSHOptsGenerator) Spy(arg1 uuid.Generator) (director.SSHOpts, string, error) {
	fake.mutex.Lock()
	ret, specificReturn := fake.returnsOnCall[len(fake.argsForCall)]
	fake.argsForCall = append(fake.argsForCall, struct {
		arg1 uuid.Generator
	}{arg1})
	stub := fake.Stub
	returns := fake.returns
	fake.recordInvocation("SSHOptsGenerator", []interface{}{arg1})
	fake.mutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2, ret.result3
	}
	return returns.result1, returns.result2, returns.result3
}

func (fake *FakeSSHOptsGenerator) CallCount() int {
	fake.mutex.RLock()
	defer fake.mutex.RUnlock()
	return len(fake.argsForCall)
}

func (fake *FakeSSHOptsGenerator) Calls(stub func(uuid.Generator) (director.SSHOpts, string, error)) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	fake.Stub = stub
}

func (fake *FakeSSHOptsGenerator) ArgsForCall(i int) uuid.Generator {
	fake.mutex.RLock()
	defer fake.mutex.RUnlock()
	return fake.argsForCall[i].arg1
}

func (fake *FakeSSHOptsGenerator) Returns(result1 director.SSHOpts, result2 string, result3 error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	fake.Stub = nil
	fake.returns = struct {
		result1 director.SSHOpts
		result2 string
		result3 error
	}{result1, result2, result3}
}

func (fake *FakeSSHOptsGenerator) ReturnsOnCall(i int, result1 director.SSHOpts, result2 string, result3 error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	fake.Stub = nil
	if fake.returnsOnCall == nil {
		fake.returnsOnCall = make(map[int]struct {
			result1 director.SSHOpts
			result2 string
			result3 error
		})
	}
	fake.returnsOnCall[i] = struct {
		result1 director.SSHOpts
		result2 string
		result3 error
	}{result1, result2, result3}
}

func (fake *FakeSSHOptsGenerator) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.mutex.RLock()
	defer fake.mutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeSSHOptsGenerator) recordInvocation(key string, args []interface{}) {
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

var _ ssh.SSHOptsGenerator = new(FakeSSHOptsGenerator).Spy
