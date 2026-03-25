package slack

import (
	"sync"
	"time"
)

type HistoryFetcher interface {
	GetHistory(channelID, cursor string) (*HistoryResult, error)
}

type Poller struct {
	client   HistoryFetcher
	interval time.Duration
	onNew    func([]Message)

	mu        sync.Mutex
	channelID string
	lastTS    string
	stop      chan struct{}
}

func NewPoller(client HistoryFetcher, interval time.Duration, onNew func([]Message)) *Poller {
	return &Poller{
		client:   client,
		interval: interval,
		onNew:    onNew,
		stop:     make(chan struct{}),
	}
}

func (p *Poller) SetChannel(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.channelID = id
	p.lastTS = ""
}

func (p *Poller) Start() {
	go p.loop()
}

func (p *Poller) Stop() {
	close(p.stop)
}

func (p *Poller) FetchNow() {
	p.poll()
}

func (p *Poller) loop() {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stop:
			return
		case <-ticker.C:
			p.poll()
		}
	}
}

func (p *Poller) poll() {
	p.mu.Lock()
	chID := p.channelID
	lastTS := p.lastTS
	p.mu.Unlock()

	if chID == "" {
		return
	}

	result, err := p.client.GetHistory(chID, "")
	if err != nil {
		return
	}

	if len(result.Messages) == 0 {
		return
	}

	var newMsgs []Message
	for _, m := range result.Messages {
		if m.Timestamp > lastTS {
			newMsgs = append(newMsgs, m)
		}
	}

	if len(newMsgs) > 0 {
		p.mu.Lock()
		p.lastTS = result.Messages[0].Timestamp
		p.mu.Unlock()
		p.onNew(newMsgs)
	} else if lastTS == "" {
		p.mu.Lock()
		p.lastTS = result.Messages[0].Timestamp
		p.mu.Unlock()
		p.onNew(result.Messages)
	}
}
