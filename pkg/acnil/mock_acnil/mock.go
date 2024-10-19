// Code generated by MockGen. DO NOT EDIT.
// Source: handler.go
//
// Generated by this command:
//
//	mockgen -source=handler.go -destination mock_acnil/mock.go
//
// Package mock_acnil is a generated GoMock package.
package mock_acnil

import (
	context "context"
	reflect "reflect"

	acnil "github.com/acnil/acnil-bot/pkg/acnil"
	gomock "go.uber.org/mock/gomock"
	telebot "gopkg.in/telebot.v3"
)

// MockMembersDatabase is a mock of MembersDatabase interface.
type MockMembersDatabase struct {
	ctrl     *gomock.Controller
	recorder *MockMembersDatabaseMockRecorder
}

// MockMembersDatabaseMockRecorder is the mock recorder for MockMembersDatabase.
type MockMembersDatabaseMockRecorder struct {
	mock *MockMembersDatabase
}

// NewMockMembersDatabase creates a new mock instance.
func NewMockMembersDatabase(ctrl *gomock.Controller) *MockMembersDatabase {
	mock := &MockMembersDatabase{ctrl: ctrl}
	mock.recorder = &MockMembersDatabaseMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockMembersDatabase) EXPECT() *MockMembersDatabaseMockRecorder {
	return m.recorder
}

// Append mocks base method.
func (m *MockMembersDatabase) Append(ctx context.Context, member acnil.Member) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Append", ctx, member)
	ret0, _ := ret[0].(error)
	return ret0
}

// Append indicates an expected call of Append.
func (mr *MockMembersDatabaseMockRecorder) Append(ctx, member any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Append", reflect.TypeOf((*MockMembersDatabase)(nil).Append), ctx, member)
}

// Get mocks base method.
func (m *MockMembersDatabase) Get(ctx context.Context, telegramID int64) (*acnil.Member, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", ctx, telegramID)
	ret0, _ := ret[0].(*acnil.Member)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockMembersDatabaseMockRecorder) Get(ctx, telegramID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockMembersDatabase)(nil).Get), ctx, telegramID)
}

// List mocks base method.
func (m *MockMembersDatabase) List(ctx context.Context) ([]acnil.Member, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", ctx)
	ret0, _ := ret[0].([]acnil.Member)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockMembersDatabaseMockRecorder) List(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockMembersDatabase)(nil).List), ctx)
}

// Update mocks base method.
func (m *MockMembersDatabase) Update(ctx context.Context, member acnil.Member) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", ctx, member)
	ret0, _ := ret[0].(error)
	return ret0
}

// Update indicates an expected call of Update.
func (mr *MockMembersDatabaseMockRecorder) Update(ctx, member any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockMembersDatabase)(nil).Update), ctx, member)
}

// MockGameDatabase is a mock of GameDatabase interface.
type MockGameDatabase struct {
	ctrl     *gomock.Controller
	recorder *MockGameDatabaseMockRecorder
}

// MockGameDatabaseMockRecorder is the mock recorder for MockGameDatabase.
type MockGameDatabaseMockRecorder struct {
	mock *MockGameDatabase
}

// NewMockGameDatabase creates a new mock instance.
func NewMockGameDatabase(ctrl *gomock.Controller) *MockGameDatabase {
	mock := &MockGameDatabase{ctrl: ctrl}
	mock.recorder = &MockGameDatabaseMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockGameDatabase) EXPECT() *MockGameDatabaseMockRecorder {
	return m.recorder
}

// Find mocks base method.
func (m *MockGameDatabase) Find(ctx context.Context, name string) ([]acnil.Game, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Find", ctx, name)
	ret0, _ := ret[0].([]acnil.Game)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Find indicates an expected call of Find.
func (mr *MockGameDatabaseMockRecorder) Find(ctx, name any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Find", reflect.TypeOf((*MockGameDatabase)(nil).Find), ctx, name)
}

// Get mocks base method.
func (m *MockGameDatabase) Get(ctx context.Context, id, name string) (*acnil.Game, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", ctx, id, name)
	ret0, _ := ret[0].(*acnil.Game)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockGameDatabaseMockRecorder) Get(ctx, id, name any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockGameDatabase)(nil).Get), ctx, id, name)
}

// List mocks base method.
func (m *MockGameDatabase) List(ctx context.Context) ([]acnil.Game, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", ctx)
	ret0, _ := ret[0].([]acnil.Game)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockGameDatabaseMockRecorder) List(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockGameDatabase)(nil).List), ctx)
}

// Update mocks base method.
func (m *MockGameDatabase) Update(ctx context.Context, game ...acnil.Game) error {
	m.ctrl.T.Helper()
	varargs := []any{ctx}
	for _, a := range game {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Update", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// Update indicates an expected call of Update.
func (mr *MockGameDatabaseMockRecorder) Update(ctx any, game ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{ctx}, game...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockGameDatabase)(nil).Update), varargs...)
}

// MockSender is a mock of Sender interface.
type MockSender struct {
	ctrl     *gomock.Controller
	recorder *MockSenderMockRecorder
}

// MockSenderMockRecorder is the mock recorder for MockSender.
type MockSenderMockRecorder struct {
	mock *MockSender
}

// NewMockSender creates a new mock instance.
func NewMockSender(ctrl *gomock.Controller) *MockSender {
	mock := &MockSender{ctrl: ctrl}
	mock.recorder = &MockSenderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSender) EXPECT() *MockSenderMockRecorder {
	return m.recorder
}

// Send mocks base method.
func (m *MockSender) Send(to telebot.Recipient, what any, opts ...any) (*telebot.Message, error) {
	m.ctrl.T.Helper()
	varargs := []any{to, what}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Send", varargs...)
	ret0, _ := ret[0].(*telebot.Message)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Send indicates an expected call of Send.
func (mr *MockSenderMockRecorder) Send(to, what any, opts ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{to, what}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Send", reflect.TypeOf((*MockSender)(nil).Send), varargs...)
}

// MockROAudit is a mock of ROAudit interface.
type MockROAudit struct {
	ctrl     *gomock.Controller
	recorder *MockROAuditMockRecorder
}

// MockROAuditMockRecorder is the mock recorder for MockROAudit.
type MockROAuditMockRecorder struct {
	mock *MockROAudit
}

// NewMockROAudit creates a new mock instance.
func NewMockROAudit(ctrl *gomock.Controller) *MockROAudit {
	mock := &MockROAudit{ctrl: ctrl}
	mock.recorder = &MockROAuditMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockROAudit) EXPECT() *MockROAuditMockRecorder {
	return m.recorder
}

// Find mocks base method.
func (m *MockROAudit) Find(ctx context.Context, query acnil.Query) ([]acnil.AuditEntry, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Find", ctx, query)
	ret0, _ := ret[0].([]acnil.AuditEntry)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Find indicates an expected call of Find.
func (mr *MockROAuditMockRecorder) Find(ctx, query any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Find", reflect.TypeOf((*MockROAudit)(nil).Find), ctx, query)
}
