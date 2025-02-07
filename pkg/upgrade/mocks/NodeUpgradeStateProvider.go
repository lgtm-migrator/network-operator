/*
Copyright 2022 NVIDIA CORPORATION & AFFILIATES
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// Code generated by mockery v1.0.0. DO NOT EDIT.
//nolint
package mocks

import context "context"
import mock "github.com/stretchr/testify/mock"

import v1 "k8s.io/api/core/v1"

// NodeUpgradeStateProvider is an autogenerated mock type for the NodeUpgradeStateProvider type
type NodeUpgradeStateProvider struct {
	mock.Mock
}

// ChangeNodeUpgradeState provides a mock function with given fields: ctx, node, newNodeState
func (_m *NodeUpgradeStateProvider) ChangeNodeUpgradeState(ctx context.Context, node *v1.Node, newNodeState string) error {
	ret := _m.Called(ctx, node, newNodeState)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Node, string) error); ok {
		r0 = rf(ctx, node, newNodeState)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetNode provides a mock function with given fields: ctx, nodeName
func (_m *NodeUpgradeStateProvider) GetNode(ctx context.Context, nodeName string) (*v1.Node, error) {
	ret := _m.Called(ctx, nodeName)

	var r0 *v1.Node
	if rf, ok := ret.Get(0).(func(context.Context, string) *v1.Node); ok {
		r0 = rf(ctx, nodeName)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.Node)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, nodeName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
