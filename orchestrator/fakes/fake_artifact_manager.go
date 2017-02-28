// This file was generated by counterfeiter
package fakes

import (
	"sync"

	"github.com/pivotal-cf/bosh-backup-and-restore/orchestrator"
)

type FakeArtifactManager struct {
	CreateStub        func(string, orchestrator.Logger) (orchestrator.Artifact, error)
	createMutex       sync.RWMutex
	createArgsForCall []struct {
		arg1 string
		arg2 orchestrator.Logger
	}
	createReturns struct {
		result1 orchestrator.Artifact
		result2 error
	}
	OpenStub        func(string, orchestrator.Logger) (orchestrator.Artifact, error)
	openMutex       sync.RWMutex
	openArgsForCall []struct {
		arg1 string
		arg2 orchestrator.Logger
	}
	openReturns struct {
		result1 orchestrator.Artifact
		result2 error
	}
	ExistsStub        func(string) bool
	existsMutex       sync.RWMutex
	existsArgsForCall []struct {
		arg1 string
	}
	existsReturns struct {
		result1 bool
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeArtifactManager) Create(arg1 string, arg2 orchestrator.Logger) (orchestrator.Artifact, error) {
	fake.createMutex.Lock()
	fake.createArgsForCall = append(fake.createArgsForCall, struct {
		arg1 string
		arg2 orchestrator.Logger
	}{arg1, arg2})
	fake.recordInvocation("Create", []interface{}{arg1, arg2})
	fake.createMutex.Unlock()
	if fake.CreateStub != nil {
		return fake.CreateStub(arg1, arg2)
	}
	return fake.createReturns.result1, fake.createReturns.result2
}

func (fake *FakeArtifactManager) CreateCallCount() int {
	fake.createMutex.RLock()
	defer fake.createMutex.RUnlock()
	return len(fake.createArgsForCall)
}

func (fake *FakeArtifactManager) CreateArgsForCall(i int) (string, orchestrator.Logger) {
	fake.createMutex.RLock()
	defer fake.createMutex.RUnlock()
	return fake.createArgsForCall[i].arg1, fake.createArgsForCall[i].arg2
}

func (fake *FakeArtifactManager) CreateReturns(result1 orchestrator.Artifact, result2 error) {
	fake.CreateStub = nil
	fake.createReturns = struct {
		result1 orchestrator.Artifact
		result2 error
	}{result1, result2}
}

func (fake *FakeArtifactManager) Open(arg1 string, arg2 orchestrator.Logger) (orchestrator.Artifact, error) {
	fake.openMutex.Lock()
	fake.openArgsForCall = append(fake.openArgsForCall, struct {
		arg1 string
		arg2 orchestrator.Logger
	}{arg1, arg2})
	fake.recordInvocation("Open", []interface{}{arg1, arg2})
	fake.openMutex.Unlock()
	if fake.OpenStub != nil {
		return fake.OpenStub(arg1, arg2)
	}
	return fake.openReturns.result1, fake.openReturns.result2
}

func (fake *FakeArtifactManager) OpenCallCount() int {
	fake.openMutex.RLock()
	defer fake.openMutex.RUnlock()
	return len(fake.openArgsForCall)
}

func (fake *FakeArtifactManager) OpenArgsForCall(i int) (string, orchestrator.Logger) {
	fake.openMutex.RLock()
	defer fake.openMutex.RUnlock()
	return fake.openArgsForCall[i].arg1, fake.openArgsForCall[i].arg2
}

func (fake *FakeArtifactManager) OpenReturns(result1 orchestrator.Artifact, result2 error) {
	fake.OpenStub = nil
	fake.openReturns = struct {
		result1 orchestrator.Artifact
		result2 error
	}{result1, result2}
}

func (fake *FakeArtifactManager) Exists(arg1 string) bool {
	fake.existsMutex.Lock()
	fake.existsArgsForCall = append(fake.existsArgsForCall, struct {
		arg1 string
	}{arg1})
	fake.recordInvocation("Exists", []interface{}{arg1})
	fake.existsMutex.Unlock()
	if fake.ExistsStub != nil {
		return fake.ExistsStub(arg1)
	}
	return fake.existsReturns.result1
}

func (fake *FakeArtifactManager) ExistsCallCount() int {
	fake.existsMutex.RLock()
	defer fake.existsMutex.RUnlock()
	return len(fake.existsArgsForCall)
}

func (fake *FakeArtifactManager) ExistsArgsForCall(i int) string {
	fake.existsMutex.RLock()
	defer fake.existsMutex.RUnlock()
	return fake.existsArgsForCall[i].arg1
}

func (fake *FakeArtifactManager) ExistsReturns(result1 bool) {
	fake.ExistsStub = nil
	fake.existsReturns = struct {
		result1 bool
	}{result1}
}

func (fake *FakeArtifactManager) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.createMutex.RLock()
	defer fake.createMutex.RUnlock()
	fake.openMutex.RLock()
	defer fake.openMutex.RUnlock()
	fake.existsMutex.RLock()
	defer fake.existsMutex.RUnlock()
	return fake.invocations
}

func (fake *FakeArtifactManager) recordInvocation(key string, args []interface{}) {
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

var _ orchestrator.ArtifactManager = new(FakeArtifactManager)
