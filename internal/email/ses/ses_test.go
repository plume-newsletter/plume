package ses

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/plume-newsletter/plume/internal/email"
)

type fakeSES struct{ last *sesv2.SendEmailInput }

func (f *fakeSES) SendEmail(_ context.Context, in *sesv2.SendEmailInput, _ ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error) {
	f.last = in
	return &sesv2.SendEmailOutput{}, nil
}

func TestSendMapsMessageToSESInput(t *testing.T) {
	f := &fakeSES{}
	p := NewWithClient(f)

	err := p.Send(context.Background(), email.Message{
		From: "n@acme.test", FromName: "Acme", ReplyTo: "r@acme.test",
		To: "a@x.test", Subject: "Hi", HTML: "<p>Hi</p>", Text: "Hi",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if p.Name() != "ses" {
		t.Fatalf("Name = %q", p.Name())
	}
	if f.last == nil || *f.last.FromEmailAddress != "Acme <n@acme.test>" {
		t.Fatalf("From mapping wrong: %+v", f.last)
	}
	if f.last.Destination.ToAddresses[0] != "a@x.test" {
		t.Fatalf("To mapping wrong: %+v", f.last.Destination)
	}
	if *f.last.Content.Simple.Subject.Data != "Hi" {
		t.Fatalf("Subject mapping wrong")
	}
}
