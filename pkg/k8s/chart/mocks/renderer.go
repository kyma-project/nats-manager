// Code generated by mockery v2.40.2. DO NOT EDIT.

package mocks

import (
	chart "github.com/kyma-project/nats-manager/pkg/k8s/chart"
	mock "github.com/stretchr/testify/mock"
)

// Renderer is an autogenerated mock type for the Renderer type
type Renderer struct {
	mock.Mock
}

type Renderer_Expecter struct {
	mock *mock.Mock
}

func (_m *Renderer) EXPECT() *Renderer_Expecter {
	return &Renderer_Expecter{mock: &_m.Mock}
}

// RenderManifest provides a mock function with given fields: _a0
func (_m *Renderer) RenderManifest(_a0 *chart.ReleaseInstance) (string, error) {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for RenderManifest")
	}

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(*chart.ReleaseInstance) (string, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(*chart.ReleaseInstance) string); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(*chart.ReleaseInstance) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Renderer_RenderManifest_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'RenderManifest'
type Renderer_RenderManifest_Call struct {
	*mock.Call
}

// RenderManifest is a helper method to define mock.On call
//   - _a0 *chart.ReleaseInstance
func (_e *Renderer_Expecter) RenderManifest(_a0 interface{}) *Renderer_RenderManifest_Call {
	return &Renderer_RenderManifest_Call{Call: _e.mock.On("RenderManifest", _a0)}
}

func (_c *Renderer_RenderManifest_Call) Run(run func(_a0 *chart.ReleaseInstance)) *Renderer_RenderManifest_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*chart.ReleaseInstance))
	})
	return _c
}

func (_c *Renderer_RenderManifest_Call) Return(_a0 string, _a1 error) *Renderer_RenderManifest_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Renderer_RenderManifest_Call) RunAndReturn(run func(*chart.ReleaseInstance) (string, error)) *Renderer_RenderManifest_Call {
	_c.Call.Return(run)
	return _c
}

// RenderManifestAsUnstructured provides a mock function with given fields: _a0
func (_m *Renderer) RenderManifestAsUnstructured(_a0 *chart.ReleaseInstance) (*chart.ManifestResources, error) {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for RenderManifestAsUnstructured")
	}

	var r0 *chart.ManifestResources
	var r1 error
	if rf, ok := ret.Get(0).(func(*chart.ReleaseInstance) (*chart.ManifestResources, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(*chart.ReleaseInstance) *chart.ManifestResources); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*chart.ManifestResources)
		}
	}

	if rf, ok := ret.Get(1).(func(*chart.ReleaseInstance) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Renderer_RenderManifestAsUnstructured_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'RenderManifestAsUnstructured'
type Renderer_RenderManifestAsUnstructured_Call struct {
	*mock.Call
}

// RenderManifestAsUnstructured is a helper method to define mock.On call
//   - _a0 *chart.ReleaseInstance
func (_e *Renderer_Expecter) RenderManifestAsUnstructured(_a0 interface{}) *Renderer_RenderManifestAsUnstructured_Call {
	return &Renderer_RenderManifestAsUnstructured_Call{Call: _e.mock.On("RenderManifestAsUnstructured", _a0)}
}

func (_c *Renderer_RenderManifestAsUnstructured_Call) Run(run func(_a0 *chart.ReleaseInstance)) *Renderer_RenderManifestAsUnstructured_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*chart.ReleaseInstance))
	})
	return _c
}

func (_c *Renderer_RenderManifestAsUnstructured_Call) Return(_a0 *chart.ManifestResources, _a1 error) *Renderer_RenderManifestAsUnstructured_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Renderer_RenderManifestAsUnstructured_Call) RunAndReturn(run func(*chart.ReleaseInstance) (*chart.ManifestResources, error)) *Renderer_RenderManifestAsUnstructured_Call {
	_c.Call.Return(run)
	return _c
}

// NewRenderer creates a new instance of Renderer. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewRenderer(t interface {
	mock.TestingT
	Cleanup(func())
}) *Renderer {
	mock := &Renderer{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
