// Package email defines the provider seam Plume sends through.
package email

import "context"

type Message struct {
	From     string
	FromName string
	ReplyTo  string
	To       string
	ToName   string
	Subject  string
	HTML     string
	Text     string
	Headers  map[string]string
}

// Provider sends a single message. Implementations: logprovider (default), ses.
type Provider interface {
	Send(ctx context.Context, msg Message) error
	Name() string
}
