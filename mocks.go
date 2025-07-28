package signal

import (
	"context"
	"fmt"
	"sync"
)

// MockExecutor for testing command execution
type MockExecutor struct {
	mu            sync.Mutex
	calls         []string
	exitCode      int
	err           error
	shouldFail    bool
	customResults map[string]mockExecResult
}

type mockExecResult struct {
	exitCode int
	err      error
}

func NewMockExecutor() *MockExecutor {
	return &MockExecutor{
		customResults: make(map[string]mockExecResult),
	}
}

func (m *MockExecutor) SetExitCode(code int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.exitCode = code
}

func (m *MockExecutor) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

func (m *MockExecutor) SetResultForCommand(cmd string, exitCode int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.customResults[cmd] = mockExecResult{exitCode: exitCode, err: err}
}

func (m *MockExecutor) Run(cmdLine string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, cmdLine)

	// Check for custom result first
	if result, exists := m.customResults[cmdLine]; exists {
		return result.exitCode, result.err
	}

	return m.exitCode, m.err
}

func (m *MockExecutor) GetCalls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.calls))
	copy(result, m.calls)
	return result
}

func (m *MockExecutor) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

// MockPublisher for testing SQS publishing
type MockPublisher struct {
	mu         sync.Mutex
	calls      []PublishInput
	err        error
	shouldFail bool
	failCount  int
	callCount  int
}

func NewMockPublisher() *MockPublisher {
	return &MockPublisher{}
}

func (m *MockPublisher) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

func (m *MockPublisher) SetFailFirstNCalls(n int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failCount = n
}

func (m *MockPublisher) Publish(ctx context.Context, input PublishInput) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, input)
	m.callCount++

	// Simulate failing first N calls (for retry testing)
	if m.callCount <= m.failCount {
		return fmt.Errorf("simulated transient error")
	}

	return m.err
}

func (m *MockPublisher) GetCalls() []PublishInput {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]PublishInput, len(m.calls))
	copy(result, m.calls)
	return result
}

func (m *MockPublisher) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

func (m *MockPublisher) GetLastCall() *PublishInput {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.calls) == 0 {
		return nil
	}
	return &m.calls[len(m.calls)-1]
}

// MockIMDSClient for testing instance ID fetching
type MockIMDSClient struct {
	mu         sync.Mutex
	instanceID string
	err        error
	callCount  int
}

func NewMockIMDSClient() *MockIMDSClient {
	return &MockIMDSClient{
		instanceID: "i-1234567890abcdef0", // Default fake instance ID
	}
}

func (m *MockIMDSClient) SetInstanceID(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.instanceID = id
}

func (m *MockIMDSClient) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

func (m *MockIMDSClient) GetInstanceID(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCount++
	return m.instanceID, m.err
}

func (m *MockIMDSClient) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}
