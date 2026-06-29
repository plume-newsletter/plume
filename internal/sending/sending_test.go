package sending_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/brand"
	"github.com/plume-newsletter/plume/internal/campaign"
	"github.com/plume-newsletter/plume/internal/email"
	"github.com/plume-newsletter/plume/internal/email/logprovider"
	"github.com/plume-newsletter/plume/internal/hooks"
	"github.com/plume-newsletter/plume/internal/list"
	"github.com/plume-newsletter/plume/internal/render"
	"github.com/plume-newsletter/plume/internal/sending"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/plume-newsletter/plume/internal/subscriber"
	"github.com/plume-newsletter/plume/internal/testsupport"
)

func TestEnqueueAndWorkerSendsToActiveSubscribers(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	owner := uuid.New()

	h := hooks.New()
	render.Register(h)
	prov := logprovider.New(&strings.Builder{})

	b, err := brand.New(q).Create(ctx, owner, brand.BrandInput{
		Name: "Acme", FromName: "Acme News", FromEmail: "news@acme.test", ReplyTo: "reply@acme.test",
	})
	if err != nil {
		t.Fatalf("seed brand: %v", err)
	}
	l, err := list.New(q).Create(ctx, owner, b.ID, "Main List")
	if err != nil {
		t.Fatalf("seed list: %v", err)
	}

	subSvc := subscriber.New(q, h)
	sub1, _, err := subSvc.Add(ctx, owner, l.ID, subscriber.SubscriberInput{Email: "a@test.com", Status: "active"})
	if err != nil {
		t.Fatalf("seed sub1: %v", err)
	}
	sub2, _, err := subSvc.Add(ctx, owner, l.ID, subscriber.SubscriberInput{Email: "b@test.com", Status: "active"})
	if err != nil {
		t.Fatalf("seed sub2: %v", err)
	}
	// A pending (non-active) subscriber should NOT receive the campaign.
	if _, _, err := subSvc.Add(ctx, owner, l.ID, subscriber.SubscriberInput{Email: "c@test.com", Status: "pending"}); err != nil {
		t.Fatalf("seed sub3: %v", err)
	}

	c, err := campaign.New(q).Create(ctx, owner, b.ID, campaign.CampaignInput{
		Subject:   "Hello",
		HtmlBody:  `<html><body><p>Hi <a href="https://acme.test/sale">sale</a></p></body></html>`,
		PlainBody: "Hi, check out https://acme.test/sale",
	})
	if err != nil {
		t.Fatalf("seed campaign: %v", err)
	}

	var sendingFired, sentFired int
	h.AddAction(sending.HookCampaignSending, 10, func(_ context.Context, _ any) error {
		sendingFired++
		return nil
	})
	h.AddAction(sending.HookCampaignSent, 10, func(_ context.Context, _ any) error {
		sentFired++
		return nil
	})

	svc := sending.New(pool, q, h)
	n, err := svc.Enqueue(ctx, owner, c.ID, l.ID)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if n != 2 {
		t.Fatalf("Enqueue recipient count = %d, want 2", n)
	}
	if sendingFired != 1 {
		t.Fatalf("campaign.sending fired %d times, want 1", sendingFired)
	}

	links, err := q.ListLinksForCampaign(ctx, c.ID)
	if err != nil || len(links) != 1 {
		t.Fatalf("ListLinksForCampaign: links=%+v err=%v", links, err)
	}
	linkID := links[0].ID

	worker := sending.NewWorker(pool, q, h, email.NewStaticResolver(prov), "https://mail.example.com", 10)

	// Drain the queue.
	for i := 0; i < 10; i++ {
		processed, err := worker.RunOnce(ctx)
		if err != nil {
			t.Fatalf("RunOnce: %v", err)
		}
		if processed == 0 {
			break
		}
	}

	sent := prov.Sent()
	if len(sent) != 2 {
		t.Fatalf("Sent() = %d messages, want 2", len(sent))
	}
	for _, msg := range sent {
		if !strings.Contains(msg.HTML, "/t/") {
			t.Errorf("message HTML missing open pixel: %s", msg.HTML)
		}
		if !strings.Contains(msg.HTML, "/l/"+linkID.String()+"/") {
			t.Errorf("message HTML missing rewritten link: %s", msg.HTML)
		}
		if msg.From != "news@acme.test" || msg.FromName != "Acme News" || msg.ReplyTo != "reply@acme.test" {
			t.Errorf("message envelope wrong: %+v", msg)
		}
		if msg.Subject != "Hello" {
			t.Errorf("message subject = %q, want Hello", msg.Subject)
		}
	}

	if sentFired != 1 {
		t.Fatalf("campaign.sent fired %d times, want 1", sentFired)
	}

	r1, err := q.GetSubscriberByID(ctx, sub1.ID)
	if err != nil {
		t.Fatalf("GetSubscriberByID sub1: %v", err)
	}
	r2, err := q.GetSubscriberByID(ctx, sub2.ID)
	if err != nil {
		t.Fatalf("GetSubscriberByID sub2: %v", err)
	}
	_ = r1
	_ = r2

	recipients, err := q.ClaimQueuedRecipients(ctx, 10)
	if err != nil {
		t.Fatalf("ClaimQueuedRecipients: %v", err)
	}
	if len(recipients) != 0 {
		t.Fatalf("expected no queued recipients remaining, got %d", len(recipients))
	}

	finalCampaign, err := q.GetCampaignByID(ctx, c.ID)
	if err != nil {
		t.Fatalf("GetCampaignByID: %v", err)
	}
	if finalCampaign.Status != "sent" {
		t.Fatalf("campaign status = %q, want sent", finalCampaign.Status)
	}
}

func TestWorkerSkipsSuppressedRecipient(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	owner := uuid.New()

	h := hooks.New()
	render.Register(h)
	prov := logprovider.New(&strings.Builder{})

	b, err := brand.New(q).Create(ctx, owner, brand.BrandInput{
		Name: "Acme", FromName: "Acme News", FromEmail: "news@acme.test", ReplyTo: "reply@acme.test",
	})
	if err != nil {
		t.Fatalf("seed brand: %v", err)
	}
	l, err := list.New(q).Create(ctx, owner, b.ID, "Main List")
	if err != nil {
		t.Fatalf("seed list: %v", err)
	}

	subSvc := subscriber.New(q, h)
	sub, _, err := subSvc.Add(ctx, owner, l.ID, subscriber.SubscriberInput{Email: "suppressed@test.com", Status: "active"})
	if err != nil {
		t.Fatalf("seed subscriber: %v", err)
	}

	if err := q.InsertSuppression(ctx, gen.InsertSuppressionParams{
		ID: uuid.New(), OwnerID: owner, Email: sub.Email, Reason: "bounce",
	}); err != nil {
		t.Fatalf("seed suppression: %v", err)
	}

	c, err := campaign.New(q).Create(ctx, owner, b.ID, campaign.CampaignInput{
		Subject:   "Hello",
		HtmlBody:  `<html><body><p>Hi</p></body></html>`,
		PlainBody: "Hi",
	})
	if err != nil {
		t.Fatalf("seed campaign: %v", err)
	}

	svc := sending.New(pool, q, h)
	if _, err := svc.Enqueue(ctx, owner, c.ID, l.ID); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	queued, err := q.ClaimQueuedRecipients(ctx, 10)
	if err != nil {
		t.Fatalf("ClaimQueuedRecipients (pre-run capture): %v", err)
	}
	if len(queued) != 1 {
		t.Fatalf("queued recipients = %d, want 1", len(queued))
	}
	recipientID := queued[0].ID

	worker := sending.NewWorker(pool, q, h, email.NewStaticResolver(prov), "https://mail.example.com", 10)
	for i := 0; i < 10; i++ {
		processed, err := worker.RunOnce(ctx)
		if err != nil {
			t.Fatalf("RunOnce: %v", err)
		}
		if processed == 0 {
			break
		}
	}

	for _, msg := range prov.Sent() {
		if msg.To == sub.Email {
			t.Fatalf("provider.Sent() should not include suppressed recipient %s", sub.Email)
		}
	}

	recipient, err := q.GetRecipientByID(ctx, recipientID)
	if err != nil {
		t.Fatalf("GetRecipientByID: %v", err)
	}
	if recipient.Status != "failed" {
		t.Fatalf("recipient status = %q, want failed", recipient.Status)
	}
	if !recipient.Error.Valid || recipient.Error.String != "suppressed" {
		t.Fatalf("recipient error = %+v, want suppressed", recipient.Error)
	}
}

func TestEnqueueRejectsDoubleSend(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	owner := uuid.New()

	h := hooks.New()
	render.Register(h)

	b, err := brand.New(q).Create(ctx, owner, brand.BrandInput{
		Name: "Acme", FromName: "Acme News", FromEmail: "news@acme.test", ReplyTo: "reply@acme.test",
	})
	if err != nil {
		t.Fatalf("seed brand: %v", err)
	}
	l, err := list.New(q).Create(ctx, owner, b.ID, "Main List")
	if err != nil {
		t.Fatalf("seed list: %v", err)
	}
	subSvc := subscriber.New(q, h)
	if _, _, err := subSvc.Add(ctx, owner, l.ID, subscriber.SubscriberInput{Email: "a@test.com", Status: "active"}); err != nil {
		t.Fatalf("seed sub: %v", err)
	}
	c, err := campaign.New(q).Create(ctx, owner, b.ID, campaign.CampaignInput{
		Subject:   "Hello",
		HtmlBody:  `<html><body><p>Hi</p></body></html>`,
		PlainBody: "Hi",
	})
	if err != nil {
		t.Fatalf("seed campaign: %v", err)
	}

	svc := sending.New(pool, q, h)
	n, err := svc.Enqueue(ctx, owner, c.ID, l.ID)
	if err != nil {
		t.Fatalf("first Enqueue: %v", err)
	}
	if n != 1 {
		t.Fatalf("first Enqueue recipient count = %d, want 1", n)
	}

	if _, err := svc.Enqueue(ctx, owner, c.ID, l.ID); !errors.Is(err, sending.ErrAlreadyQueued) {
		t.Fatalf("second Enqueue: err = %v, want ErrAlreadyQueued", err)
	}

	recipients, err := q.ClaimQueuedRecipients(ctx, 10)
	if err != nil {
		t.Fatalf("ClaimQueuedRecipients: %v", err)
	}
	if len(recipients) != 1 {
		t.Fatalf("queued recipients = %d, want 1 (no double-send)", len(recipients))
	}
}

func TestEnqueueSucceedsWhenActionHandlerErrors(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	owner := uuid.New()

	h := hooks.New()
	render.Register(h)
	h.AddAction(sending.HookCampaignSending, 10, func(_ context.Context, _ any) error {
		return errors.New("handler boom")
	})

	b, err := brand.New(q).Create(ctx, owner, brand.BrandInput{
		Name: "Acme", FromName: "Acme News", FromEmail: "news@acme.test", ReplyTo: "reply@acme.test",
	})
	if err != nil {
		t.Fatalf("seed brand: %v", err)
	}
	l, err := list.New(q).Create(ctx, owner, b.ID, "Main List")
	if err != nil {
		t.Fatalf("seed list: %v", err)
	}
	subSvc := subscriber.New(q, h)
	if _, _, err := subSvc.Add(ctx, owner, l.ID, subscriber.SubscriberInput{Email: "a@test.com", Status: "active"}); err != nil {
		t.Fatalf("seed sub: %v", err)
	}
	c, err := campaign.New(q).Create(ctx, owner, b.ID, campaign.CampaignInput{
		Subject:   "Hello",
		HtmlBody:  `<html><body><p>Hi</p></body></html>`,
		PlainBody: "Hi",
	})
	if err != nil {
		t.Fatalf("seed campaign: %v", err)
	}

	svc := sending.New(pool, q, h)
	n, err := svc.Enqueue(ctx, owner, c.ID, l.ID)
	if err != nil {
		t.Fatalf("Enqueue should succeed despite handler error: %v", err)
	}
	if n != 1 {
		t.Fatalf("Enqueue recipient count = %d, want 1", n)
	}
}

func TestWorkerHonorsPerRecipientSubjectVariant(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	owner := uuid.New()

	h := hooks.New()
	render.Register(h)
	prov := logprovider.New(&strings.Builder{})

	b, err := brand.New(q).Create(ctx, owner, brand.BrandInput{
		Name: "Acme", FromName: "Acme News", FromEmail: "news@acme.test", ReplyTo: "reply@acme.test",
	})
	if err != nil {
		t.Fatalf("seed brand: %v", err)
	}
	l, err := list.New(q).Create(ctx, owner, b.ID, "Main List")
	if err != nil {
		t.Fatalf("seed list: %v", err)
	}

	subSvc := subscriber.New(q, h)
	sub, _, err := subSvc.Add(ctx, owner, l.ID, subscriber.SubscriberInput{Email: "variant@test.com", Status: "active"})
	if err != nil {
		t.Fatalf("seed subscriber: %v", err)
	}

	c, err := campaign.New(q).Create(ctx, owner, b.ID, campaign.CampaignInput{
		Subject:   "Campaign Subj",
		HtmlBody:  `<html><body><p>Hi</p></body></html>`,
		PlainBody: "Hi",
	})
	if err != nil {
		t.Fatalf("seed campaign: %v", err)
	}

	// Seed a variant recipient directly (bypassing Enqueue) with a per-recipient subject.
	if _, err := q.CreateRecipientVariant(ctx, gen.CreateRecipientVariantParams{
		ID:           uuid.New(),
		CampaignID:   c.ID,
		SubscriberID: sub.ID,
		Variant:      "a",
		Subject:      "Variant Subj",
	}); err != nil {
		t.Fatalf("CreateRecipientVariant: %v", err)
	}

	worker := sending.NewWorker(pool, q, h, email.NewStaticResolver(prov), "https://mail.example.com", 10)
	for i := 0; i < 10; i++ {
		processed, err := worker.RunOnce(ctx)
		if err != nil {
			t.Fatalf("RunOnce: %v", err)
		}
		if processed == 0 {
			break
		}
	}

	sent := prov.Sent()
	if len(sent) != 1 {
		t.Fatalf("Sent() = %d messages, want 1", len(sent))
	}
	if sent[0].Subject != "Variant Subj" {
		t.Errorf("message subject = %q, want %q", sent[0].Subject, "Variant Subj")
	}
}

func TestEnqueueNotFound(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	owner := uuid.New()

	h := hooks.New()
	render.Register(h)
	svc := sending.New(pool, q, h)

	b, err := brand.New(q).Create(ctx, owner, brand.BrandInput{
		Name: "Acme", FromName: "Acme News", FromEmail: "news@acme.test", ReplyTo: "reply@acme.test",
	})
	if err != nil {
		t.Fatalf("seed brand: %v", err)
	}
	l, err := list.New(q).Create(ctx, owner, b.ID, "Main List")
	if err != nil {
		t.Fatalf("seed list: %v", err)
	}
	c, err := campaign.New(q).Create(ctx, owner, b.ID, campaign.CampaignInput{
		Subject:   "Hello",
		HtmlBody:  `<html><body><p>Hi</p></body></html>`,
		PlainBody: "Hi",
	})
	if err != nil {
		t.Fatalf("seed campaign: %v", err)
	}

	if _, err := svc.Enqueue(ctx, owner, uuid.New(), l.ID); !errors.Is(err, sending.ErrCampaignNotFound) {
		t.Fatalf("Enqueue with unknown campaign: err = %v, want ErrCampaignNotFound", err)
	}

	if _, err := svc.Enqueue(ctx, owner, c.ID, uuid.New()); !errors.Is(err, sending.ErrListNotFound) {
		t.Fatalf("Enqueue with unknown list: err = %v, want ErrListNotFound", err)
	}
}
