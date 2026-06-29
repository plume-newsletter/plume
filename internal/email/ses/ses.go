// Package ses sends email through AWS SES v2.
package ses

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/plume-newsletter/plume/internal/email"
)

// sesAPI is the slice of the SES client we use (lets tests inject a fake).
type sesAPI interface {
	SendEmail(ctx context.Context, in *sesv2.SendEmailInput, optFns ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error)
}

type Provider struct{ client sesAPI }

// NewWithClient builds a provider from an explicit SES client (used by tests and
// by NewFromCreds once a real client is constructed).
func NewWithClient(client sesAPI) *Provider { return &Provider{client: client} }

// NewFromCreds builds a Provider backed by a real SES v2 client configured with
// static credentials. It is a thin, untested wrapper (requires AWS to exercise);
// the mapping logic it delegates to is covered by TestSendMapsMessageToSESInput.
func NewFromCreds(ctx context.Context, accessKeyID, secretAccessKey, region string) (*Provider, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")),
	)
	if err != nil {
		return nil, err
	}
	return NewWithClient(sesv2.NewFromConfig(cfg)), nil
}

func (p *Provider) Name() string { return "ses" }

func (p *Provider) Send(ctx context.Context, msg email.Message) error {
	_, err := p.client.SendEmail(ctx, buildInput(msg))
	return err
}

func buildInput(msg email.Message) *sesv2.SendEmailInput {
	from := msg.From
	if msg.FromName != "" {
		from = msg.FromName + " <" + msg.From + ">"
	}
	in := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(from),
		Destination:      &types.Destination{ToAddresses: []string{msg.To}},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{Data: aws.String(msg.Subject)},
				Body: &types.Body{
					Html: &types.Content{Data: aws.String(msg.HTML)},
					Text: &types.Content{Data: aws.String(msg.Text)},
				},
			},
		},
	}
	if msg.ReplyTo != "" {
		in.ReplyToAddresses = []string{msg.ReplyTo}
	}
	return in
}
