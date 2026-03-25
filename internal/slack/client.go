package slack

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	slackapi "github.com/slack-go/slack"
)

type SlackClient interface {
	AuthTest() (*AuthInfo, error)
	ListChannels() ([]Channel, error)
	ListDMs() ([]Conversation, error)
	ListStarredChannelIDs() (map[string]bool, error)
	GetHistory(channelID string, cursor string) (*HistoryResult, error)
	GetThreadReplies(channelID, threadTS string) ([]Message, error)
	SendMessage(channelID, text string) error
	ReplyToThread(channelID, threadTS, text string) error
	AddReaction(channelID, timestamp, emoji string) error
	RemoveReaction(channelID, timestamp, emoji string) error
	GetUsers() ([]User, error)
}

type Client struct {
	api     *slackapi.Client
	baseURL string
	token   string
	cookie  string
}

type ClientOption func(*Client)

func WithBaseURL(url string) ClientOption {
	return func(c *Client) { c.baseURL = url }
}

func NewClient(token, cookie string, opts ...ClientOption) *Client {
	c := &Client{token: token, cookie: cookie}
	for _, o := range opts {
		o(c)
	}

	options := []slackapi.Option{}
	if cookie != "" {
		options = append(options, slackapi.OptionHTTPClient(&http.Client{
			Transport: &cookieTransport{cookie: cookie, token: token},
			Timeout:   30 * time.Second,
		}))
	} else {
		options = append(options, slackapi.OptionHTTPClient(&http.Client{
			Timeout: 30 * time.Second,
		}))
	}
	if c.baseURL != "" {
		options = append(options, slackapi.OptionAPIURL(c.baseURL+"/"))
	}

	c.api = slackapi.New(token, options...)
	return c
}

type cookieTransport struct {
	cookie string
	token  string
}

func (t *cookieTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cookie := t.cookie
	if !strings.HasPrefix(cookie, "d=") {
		cookie = "d=" + cookie
	}
	req.Header.Set("Cookie", cookie)
	if t.token != "" {
		req.Header.Set("Authorization", "Bearer "+t.token)
	}
	return http.DefaultTransport.RoundTrip(req)
}

func (c *Client) AuthTest() (*AuthInfo, error) {
	resp, err := c.api.AuthTest()
	if err != nil {
		return nil, err
	}
	return &AuthInfo{
		UserID: resp.UserID,
		User:   resp.User,
		TeamID: resp.TeamID,
		Team:   resp.Team,
		URL:    resp.URL,
	}, nil
}

func (c *Client) ListChannels() ([]Channel, error) {
	params := &slackapi.GetConversationsForUserParameters{
		Types:           []string{"public_channel", "private_channel"},
		ExcludeArchived: true,
		Limit:           200,
	}

	var channels []Channel
	for {
		convs, cursor, err := c.getUserConversationsWithRetry(params)
		if err != nil {
			log.Printf("ListChannels stopping after %d channels: %v", len(channels), err)
			if len(channels) > 0 {
				return channels, nil
			}
			return nil, err
		}
		for _, ch := range convs {
			channels = append(channels, Channel{
				ID:        ch.ID,
				Name:      ch.Name,
				IsPrivate: ch.IsPrivate,
				IsMember:  true,
				Topic:     ch.Topic.Value,
			})
		}
		if cursor == "" {
			break
		}
		params.Cursor = cursor
	}
	log.Printf("ListChannels loaded %d channels total", len(channels))
	return channels, nil
}

func (c *Client) ListDMs() ([]Conversation, error) {
	params := &slackapi.GetConversationsParameters{
		Types: []string{"im", "mpim"},
		Limit: 100,
	}

	chans, _, err := c.getConversationsWithRetry(params)
	if err != nil {
		return nil, err
	}

	var convs []Conversation
	for _, ch := range chans {
		if ch.User == "" {
			continue
		}
		convs = append(convs, Conversation{
			ID:     ch.ID,
			UserID: ch.User,
			Name:   ch.User,
		})
	}
	return convs, nil
}

func (c *Client) ListStarredChannelIDs() (map[string]bool, error) {
	params := slackapi.NewStarsParameters()
	params.Count = 200
	items, _, err := c.api.ListStars(params)
	if err != nil {
		return nil, err
	}
	starred := make(map[string]bool)
	for _, item := range items {
		if item.Channel != "" {
			starred[item.Channel] = true
		}
	}
	log.Printf("ListStarredChannelIDs: %d starred", len(starred))
	return starred, nil
}

func (c *Client) getUserConversationsWithRetry(params *slackapi.GetConversationsForUserParameters) ([]slackapi.Channel, string, error) {
	for attempt := 0; attempt < 5; attempt++ {
		convs, cursor, err := c.api.GetConversationsForUser(params)
		if err == nil {
			return convs, cursor, nil
		}
		if rateLimited, delay := isRateLimited(err); rateLimited {
			time.Sleep(delay)
			continue
		}
		return nil, "", err
	}
	return nil, "", fmt.Errorf("rate limited after 5 retries")
}

func (c *Client) getConversationsWithRetry(params *slackapi.GetConversationsParameters) ([]slackapi.Channel, string, error) {
	for attempt := 0; attempt < 5; attempt++ {
		convs, cursor, err := c.api.GetConversations(params)
		if err == nil {
			return convs, cursor, nil
		}
		if rateLimited, delay := isRateLimited(err); rateLimited {
			time.Sleep(delay)
			continue
		}
		return nil, "", err
	}
	return nil, "", fmt.Errorf("rate limited after 5 retries")
}

func isRateLimited(err error) (bool, time.Duration) {
	if rlErr, ok := err.(*slackapi.RateLimitedError); ok {
		return true, rlErr.RetryAfter
	}
	if strings.Contains(err.Error(), "rate limit") {
		return true, 5 * time.Second
	}
	return false, 0
}

func (c *Client) GetHistory(channelID string, cursor string) (*HistoryResult, error) {
	params := &slackapi.GetConversationHistoryParameters{
		ChannelID: channelID,
		Limit:     50,
		Cursor:    cursor,
	}

	resp, err := c.api.GetConversationHistory(params)
	if err != nil {
		return nil, err
	}

	messages := make([]Message, 0, len(resp.Messages))
	for _, m := range resp.Messages {
		messages = append(messages, convertMessage(m))
	}

	return &HistoryResult{
		Messages: messages,
		Cursor:   resp.ResponseMetaData.NextCursor,
		HasMore:  resp.HasMore,
	}, nil
}

func (c *Client) GetThreadReplies(channelID, threadTS string) ([]Message, error) {
	msgs, _, _, err := c.api.GetConversationReplies(&slackapi.GetConversationRepliesParameters{
		ChannelID: channelID,
		Timestamp: threadTS,
	})
	if err != nil {
		return nil, err
	}

	result := make([]Message, 0, len(msgs))
	for _, m := range msgs {
		result = append(result, convertMessage(m))
	}
	return result, nil
}

func (c *Client) SendMessage(channelID, text string) error {
	if c.baseURL != "" {
		return c.sendMessageHTTP(channelID, text, "")
	}
	_, _, err := c.api.PostMessage(channelID, slackapi.MsgOptionText(text, false))
	return err
}

func (c *Client) ReplyToThread(channelID, threadTS, text string) error {
	if c.baseURL != "" {
		return c.sendMessageHTTP(channelID, text, threadTS)
	}
	_, _, err := c.api.PostMessage(channelID,
		slackapi.MsgOptionText(text, false),
		slackapi.MsgOptionTS(threadTS),
	)
	return err
}

func (c *Client) sendMessageHTTP(channelID, text, threadTS string) error {
	data := url.Values{
		"channel": {channelID},
		"text":    {text},
	}
	if threadTS != "" {
		data.Set("thread_ts", threadTS)
	}

	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://slack.com/api"
	}

	req, err := http.NewRequest("POST", baseURL+"/chat.postMessage",
		strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+c.token)
	if c.cookie != "" {
		req.Header.Set("Cookie", c.cookie)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack API returned %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) AddReaction(channelID, timestamp, emoji string) error {
	return c.api.AddReaction(emoji, slackapi.ItemRef{
		Channel:   channelID,
		Timestamp: timestamp,
	})
}

func (c *Client) RemoveReaction(channelID, timestamp, emoji string) error {
	return c.api.RemoveReaction(emoji, slackapi.ItemRef{
		Channel:   channelID,
		Timestamp: timestamp,
	})
}

func (c *Client) GetUsers() ([]User, error) {
	slackUsers, err := c.api.GetUsers()
	if err != nil {
		return nil, err
	}

	users := make([]User, 0, len(slackUsers))
	for _, u := range slackUsers {
		displayName := u.Profile.DisplayName
		if displayName == "" {
			displayName = u.RealName
		}
		users = append(users, User{
			ID:          u.ID,
			Name:        u.Name,
			DisplayName: displayName,
			IsBot:       u.IsBot,
		})
	}
	return users, nil
}

func convertMessage(m slackapi.Message) Message {
	var reactions []Reaction
	for _, r := range m.Reactions {
		reactions = append(reactions, Reaction{
			Name:  r.Name,
			Count: r.Count,
			Users: r.Users,
		})
	}

	return Message{
		UserID:     m.User,
		Username:   m.Username,
		Text:       m.Text,
		Timestamp:  m.Timestamp,
		ThreadTS:   m.ThreadTimestamp,
		ReplyCount: m.ReplyCount,
		Reactions:  reactions,
		Edited:     m.Edited != nil,
	}
}
