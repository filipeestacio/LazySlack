package slack

import (
	"sync"
	"testing"
	"time"
)

type mockSlackClient struct {
	mu      sync.Mutex
	history []Message
	calls   int
	err     error
}

func (m *mockSlackClient) GetHistory(channelID, cursor string) (*HistoryResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	return &HistoryResult{Messages: m.history}, nil
}

func (m *mockSlackClient) getCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

func TestPollerDeliversNewMessages(t *testing.T) {
	mock := &mockSlackClient{
		history: []Message{{Text: "hello", Timestamp: "1706000001.000000"}},
	}

	msgs := make(chan []Message, 10)
	p := NewPoller(mock, 50*time.Millisecond, func(m []Message) { msgs <- m })

	p.SetChannel("C1")
	p.Start()
	defer p.Stop()

	select {
	case got := <-msgs:
		if len(got) == 0 || got[0].Text != "hello" {
			t.Errorf("unexpected message: %v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for messages")
	}
}

func TestPollerPausesWithNoChannel(t *testing.T) {
	mock := &mockSlackClient{
		history: []Message{{Text: "hello", Timestamp: "1706000001.000000"}},
	}

	p := NewPoller(mock, 50*time.Millisecond, func(m []Message) {})
	p.Start()
	defer p.Stop()

	time.Sleep(150 * time.Millisecond)
	if mock.getCalls() != 0 {
		t.Errorf("expected 0 calls with no channel, got %d", mock.getCalls())
	}
}
