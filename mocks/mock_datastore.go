// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/gunnermanx/simplegameserver/datastore (interfaces: Datastore)

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	model "github.com/gunnermanx/simplegameserver/datastore/model"
)

// MockDatastore is a mock of Datastore interface.
type MockDatastore struct {
	ctrl     *gomock.Controller
	recorder *MockDatastoreMockRecorder
}

// MockDatastoreMockRecorder is the mock recorder for MockDatastore.
type MockDatastoreMockRecorder struct {
	mock *MockDatastore
}

// NewMockDatastore creates a new mock instance.
func NewMockDatastore(ctrl *gomock.Controller) *MockDatastore {
	mock := &MockDatastore{ctrl: ctrl}
	mock.recorder = &MockDatastoreMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDatastore) EXPECT() *MockDatastoreMockRecorder {
	return m.recorder
}

// FindMatchmakingData mocks base method.
func (m *MockDatastore) FindMatchmakingData(arg0 string) (model.MatchmakingData, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindMatchmakingData", arg0)
	ret0, _ := ret[0].(model.MatchmakingData)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindMatchmakingData indicates an expected call of FindMatchmakingData.
func (mr *MockDatastoreMockRecorder) FindMatchmakingData(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindMatchmakingData", reflect.TypeOf((*MockDatastore)(nil).FindMatchmakingData), arg0)
}

// FindUser mocks base method.
func (m *MockDatastore) FindUser(arg0 string) (model.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindUser", arg0)
	ret0, _ := ret[0].(model.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindUser indicates an expected call of FindUser.
func (mr *MockDatastoreMockRecorder) FindUser(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindUser", reflect.TypeOf((*MockDatastore)(nil).FindUser), arg0)
}
