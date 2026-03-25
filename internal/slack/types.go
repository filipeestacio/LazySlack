package slack

import "time"

type AuthInfo struct {
	UserID string
	User   string
	TeamID string
	Team   string
	URL    string
}

type Channel struct {
	ID        string
	Name      string
	IsPrivate bool
	IsMember  bool
	Topic     string
}

type Conversation struct {
	ID     string
	UserID string
	Name   string
}

type User struct {
	ID          string
	Name        string
	DisplayName string
	IsBot       bool
}

type Message struct {
	UserID     string
	Username   string
	Text       string
	Timestamp  string
	ThreadTS   string
	ReplyCount int
	Reactions  []Reaction
	Edited     bool
}

type Reaction struct {
	Name  string
	Count int
	Users []string
}

type HistoryResult struct {
	Messages []Message
	Cursor   string
	HasMore  bool
}

func (m Message) Time() time.Time {
	var sec, usec int64
	for i, c := range m.Timestamp {
		if c == '.' {
			sec = parseInt(m.Timestamp[:i])
			usec = parseInt(m.Timestamp[i+1:])
			break
		}
	}
	return time.Unix(sec, usec*1000)
}

func parseInt(s string) int64 {
	var n int64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int64(c-'0')
		}
	}
	return n
}
