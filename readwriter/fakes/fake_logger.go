// Code generated by counterfeiter. DO NOT EDIT.
package fakes

import (
	"sync"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/readwriter"
)

type FakeLogger struct {
	InfoStub        func(string, string, ...interface{})
	infoMutex       sync.RWMutex
	infoArgsForCall []struct {
		arg1 string
		arg2 string
		arg3 []interface{}
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeLogger) Info(arg1 string, arg2 string, arg3 ...interface{}) {
	fake.infoMutex.Lock()
	fake.infoArgsForCall = append(fake.infoArgsForCall, struct {
		arg1 string
		arg2 string
		arg3 []interface{}
	}{arg1, arg2, arg3})
	stub := fake.InfoStub
	fake.recordInvocation("Info", []interface{}{arg1, arg2, arg3})
	fake.infoMutex.Unlock()
	if stub != nil {
		fake.InfoStub(arg1, arg2, arg3...)
	}
}

func (fake *FakeLogger) InfoCallCount() int {
	fake.infoMutex.RLock()
	defer fake.infoMutex.RUnlock()
	return len(fake.infoArgsForCall)
}

func (fake *FakeLogger) InfoCalls(stub func(string, string, ...interface{})) {
	fake.infoMutex.Lock()
	defer fake.infoMutex.Unlock()
	fake.InfoStub = stub
}

func (fake *FakeLogger) InfoArgsForCall(i int) (string, string, []interface{}) {
	fake.infoMutex.RLock()
	defer fake.infoMutex.RUnlock()
	argsForCall := fake.infoArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *FakeLogger) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.infoMutex.RLock()
	defer fake.infoMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeLogger) recordInvocation(key string, args []interface{}) {
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

var _ readwriter.Logger = new(FakeLogger)
