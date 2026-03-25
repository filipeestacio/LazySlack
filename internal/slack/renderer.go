package slack

import (
	"regexp"
	"strings"
)

type UserResolver interface {
	ResolveUser(id string) string
	ResolveChannel(id string) string
}

type Renderer struct {
	resolver UserResolver
}

func NewRenderer(resolver UserResolver) *Renderer {
	return &Renderer{resolver: resolver}
}

var (
	userMentionRe = regexp.MustCompile(`<@(U[A-Z0-9]+)>`)
	channelLinkRe = regexp.MustCompile(`<#(C[A-Z0-9]+)(?:\|([^>]+))?>`)
	urlRe         = regexp.MustCompile(`<(https?://[^|>]+)\|([^>]+)>`)
	urlBareRe     = regexp.MustCompile(`<(https?://[^>]+)>`)
	boldRe        = regexp.MustCompile(`\*([^*]+)\*`)
	italicRe      = regexp.MustCompile(`_([^_]+)_`)
	strikeRe      = regexp.MustCompile(`~([^~]+)~`)
	codeInlineRe  = regexp.MustCompile("`([^`]+)`")
)

func (r *Renderer) RenderPlain(text string) string {
	text = userMentionRe.ReplaceAllStringFunc(text, func(match string) string {
		id := userMentionRe.FindStringSubmatch(match)[1]
		if r.resolver != nil {
			return "@" + r.resolver.ResolveUser(id)
		}
		return "@" + id
	})

	text = channelLinkRe.ReplaceAllStringFunc(text, func(match string) string {
		parts := channelLinkRe.FindStringSubmatch(match)
		if parts[2] != "" {
			return "#" + parts[2]
		}
		if r.resolver != nil {
			return "#" + r.resolver.ResolveChannel(parts[1])
		}
		return "#" + parts[1]
	})

	text = urlRe.ReplaceAllString(text, "$2 ($1)")
	text = urlBareRe.ReplaceAllString(text, "$1")

	text = boldRe.ReplaceAllString(text, "$1")
	text = italicRe.ReplaceAllString(text, "$1")
	text = strikeRe.ReplaceAllString(text, "$1")
	text = codeInlineRe.ReplaceAllString(text, "$1")

	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")

	return text
}
