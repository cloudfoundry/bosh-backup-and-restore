// Code generated by counterfeiter. DO NOT EDIT.
package fakes

import (
	"sync"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
)

type FakeLockOrderer struct {
	OrderStub        func([]orchestrator.Job) ([][]orchestrator.Job, error)
	orderMutex       sync.RWMutex
	orderArgsForCall []struct {
		arg1 []orchestrator.Job
	}
	orderReturns struct {
		result1 [][]orchestrator.Job
		result2 error
	}
	orderReturnsOnCall map[int]struct {
		result1 [][]orchestrator.Job
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeLockOrderer) Order(arg1 []orchestrator.Job) ([][]orchestrator.Job, error) {
	var arg1Copy []orchestrator.Job
	if arg1 != nil {
		arg1Copy = make([]orchestrator.Job, len(arg1))
		copy(arg1Copy, arg1)
	}
	fake.orderMutex.Lock()
	ret, specificReturn := fake.orderReturnsOnCall[len(fake.orderArgsForCall)]
	fake.orderArgsForCall = append(fake.orderArgsForCall, struct {
		arg1 []orchestrator.Job
	}{arg1Copy})
	stub := fake.OrderStub
	fakeReturns := fake.orderReturns
	fake.recordInvocation("Order", []interface{}{arg1Copy})
	fake.orderMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeLockOrderer) OrderCallCount() int {
	fake.orderMutex.RLock()
	defer fake.orderMutex.RUnlock()
	return len(fake.orderArgsForCall)
}

func (fake *FakeLockOrderer) OrderCalls(stub func([]orchestrator.Job) ([][]orchestrator.Job, error)) {
	fake.orderMutex.Lock()
	defer fake.orderMutex.Unlock()
	fake.OrderStub = stub
}

func (fake *FakeLockOrderer) OrderArgsForCall(i int) []orchestrator.Job {
	fake.orderMutex.RLock()
	defer fake.orderMutex.RUnlock()
	argsForCall := fake.orderArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeLockOrderer) OrderReturns(result1 [][]orchestrator.Job, result2 error) {
	fake.orderMutex.Lock()
	defer fake.orderMutex.Unlock()
	fake.OrderStub = nil
	fake.orderReturns = struct {
		result1 [][]orchestrator.Job
		result2 error
	}{result1, result2}
}

func (fake *FakeLockOrderer) OrderReturnsOnCall(i int, result1 [][]orchestrator.Job, result2 error) {
	fake.orderMutex.Lock()
	defer fake.orderMutex.Unlock()
	fake.OrderStub = nil
	if fake.orderReturnsOnCall == nil {
		fake.orderReturnsOnCall = make(map[int]struct {
			result1 [][]orchestrator.Job
			result2 error
		})
	}
	fake.orderReturnsOnCall[i] = struct {
		result1 [][]orchestrator.Job
		result2 error
	}{result1, result2}
}

func (fake *FakeLockOrderer) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.orderMutex.RLock()
	defer fake.orderMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeLockOrderer) recordInvocation(key string, args []interface{}) {
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

var _ orchestrator.LockOrderer = new(FakeLockOrderer)
