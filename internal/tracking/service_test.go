package tracking_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/brand"
	"github.com/plume-newsletter/plume/internal/campaign"
	"github.com/plume-newsletter/plume/internal/hooks"
	"github.com/plume-newsletter/plume/internal/list"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/plume-newsletter/plume/internal/subscriber"
	"github.com/plume-newsletter/plume/internal/testsupport"
	"github.com/plume-newsletter/plume/internal/tracking"
)

func seed(t *testing.T, ctx context.Context, q *gen.Queries, h *hooks.Hooks) (owner uuid.UUID, c gen.Campaign, sub gen.Subscriber, recipient gen.CampaignRecipient, link gen.Link) {
	t.Helper()
	owner = uuid.New()

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
	sub, _, err = subscriber.New(q, h).Add(ctx, owner, l.ID, subscriber.SubscriberInput{Email: "reader@test.com", Status: "active"})
	if err != nil {
		t.Fatalf("seed subscriber: %v", err)
	}
	c, err = campaign.New(q).Create(ctx, owner, b.ID, campaign.CampaignInput{
		Subject:   "Hello",
		HtmlBody:  `<html><body><p>Hi <a href="https://acme.test/sale">sale</a></p></body></html>`,
		PlainBody: "Hi",
	})
	if err != nil {
		t.Fatalf("seed campaign: %v", err)
	}
	link, err = q.CreateLink(ctx, gen.CreateLinkParams{ID: uuid.New(), CampaignID: c.ID, Url: "https://acme.test/sale"})
	if err != nil {
		t.Fatalf("seed link: %v", err)
	}
	recipient, err = q.CreateRecipient(ctx, gen.CreateRecipientParams{ID: uuid.New(), CampaignID: c.ID, SubscriberID: sub.ID})
	if err != nil {
		t.Fatalf("seed recipient: %v", err)
	}
	return owner, c, sub, recipient, link
}

func TestRecordOpenInsertsEventAndFiresHook(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	h := hooks.New()

	_, c, sub, recipient, _ := seed(t, ctx, q, h)

	var opened int
	h.AddAction(tracking.HookOpened, 10, func(_ context.Context, _ any) error {
		opened++
		return nil
	})

	svc := tracking.New(q, h)
	if err := svc.RecordOpen(ctx, recipient.ID); err != nil {
		t.Fatalf("RecordOpen: %v", err)
	}
	if opened != 1 {
		t.Fatalf("email.opened fired %d times, want 1", opened)
	}

	events, err := q.ListEmailEventsForCampaign(ctx, c.ID)
	if err != nil {
		t.Fatalf("ListEmailEventsForCampaign: %v", err)
	}
	if len(events) != 1 || events[0].Type != "open" || events[0].SubscriberID != sub.ID {
		t.Fatalf("events = %+v, want one open event for subscriber %s", events, sub.ID)
	}
}

func TestRecordOpenUnknownRecipientIsSilent(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	h := hooks.New()

	svc := tracking.New(q, h)
	if err := svc.RecordOpen(ctx, uuid.New()); err != nil {
		t.Fatalf("RecordOpen with unknown recipient should not error: %v", err)
	}
}

func TestRecordClickReturnsURLAndInsertsEvent(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	h := hooks.New()

	_, c, sub, recipient, link := seed(t, ctx, q, h)

	var clicked int
	h.AddAction(tracking.HookClicked, 10, func(_ context.Context, _ any) error {
		clicked++
		return nil
	})

	svc := tracking.New(q, h)
	url, err := svc.RecordClick(ctx, link.ID, recipient.ID)
	if err != nil {
		t.Fatalf("RecordClick: %v", err)
	}
	if url != "https://acme.test/sale" {
		t.Fatalf("RecordClick url = %q, want https://acme.test/sale", url)
	}
	if clicked != 1 {
		t.Fatalf("link.clicked fired %d times, want 1", clicked)
	}

	events, err := q.ListEmailEventsForCampaign(ctx, c.ID)
	if err != nil {
		t.Fatalf("ListEmailEventsForCampaign: %v", err)
	}
	if len(events) != 1 || events[0].Type != "click" || events[0].SubscriberID != sub.ID {
		t.Fatalf("events = %+v, want one click event for subscriber %s", events, sub.ID)
	}
	if !events[0].LinkID.Valid || events[0].LinkID.Bytes != link.ID {
		t.Fatalf("click event link_id = %+v, want %s", events[0].LinkID, link.ID)
	}
}

func TestRecordClickSucceedsWhenActionHandlerErrors(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	h := hooks.New()

	_, _, _, recipient, link := seed(t, ctx, q, h)

	h.AddAction(tracking.HookClicked, 10, func(_ context.Context, _ any) error {
		return errors.New("handler boom")
	})

	svc := tracking.New(q, h)
	url, err := svc.RecordClick(ctx, link.ID, recipient.ID)
	if err != nil {
		t.Fatalf("RecordClick should succeed despite handler error: %v", err)
	}
	if url != "https://acme.test/sale" {
		t.Fatalf("RecordClick url = %q, want https://acme.test/sale", url)
	}
}

func TestRecordClickUnknownLinkReturnsErrLinkNotFound(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	h := hooks.New()

	_, _, _, recipient, _ := seed(t, ctx, q, h)

	svc := tracking.New(q, h)
	if _, err := svc.RecordClick(ctx, uuid.New(), recipient.ID); !errors.Is(err, tracking.ErrLinkNotFound) {
		t.Fatalf("RecordClick with unknown link: err = %v, want ErrLinkNotFound", err)
	}
}

func TestRecordComplaintSetsStatusAndSuppresses(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	h := hooks.New()

	owner, _, sub, _, _ := seed(t, ctx, q, h)

	var complained int
	h.AddAction(tracking.HookComplained, 10, func(_ context.Context, _ any) error {
		complained++
		return nil
	})

	svc := tracking.New(q, h)
	if err := svc.RecordComplaint(ctx, sub.Email); err != nil {
		t.Fatalf("RecordComplaint: %v", err)
	}
	if complained != 1 {
		t.Fatalf("email.complained fired %d times, want 1", complained)
	}

	updated, err := q.GetSubscriberByID(ctx, sub.ID)
	if err != nil {
		t.Fatalf("GetSubscriberByID: %v", err)
	}
	if updated.Status != "complained" {
		t.Fatalf("subscriber status = %q, want complained", updated.Status)
	}

	suppressed, err := q.IsSuppressed(ctx, gen.IsSuppressedParams{OwnerID: owner, Email: sub.Email})
	if err != nil {
		t.Fatalf("IsSuppressed: %v", err)
	}
	if !suppressed {
		t.Fatalf("expected suppression entry for %s", sub.Email)
	}
}

func TestRecordBounceSetsStatusAndSuppresses(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	h := hooks.New()

	owner, _, sub, _, _ := seed(t, ctx, q, h)

	var bounced int
	h.AddAction(tracking.HookBounced, 10, func(_ context.Context, _ any) error {
		bounced++
		return nil
	})

	svc := tracking.New(q, h)
	if err := svc.RecordBounce(ctx, sub.Email); err != nil {
		t.Fatalf("RecordBounce: %v", err)
	}
	if bounced != 1 {
		t.Fatalf("email.bounced fired %d times, want 1", bounced)
	}

	updated, err := q.GetSubscriberByID(ctx, sub.ID)
	if err != nil {
		t.Fatalf("GetSubscriberByID: %v", err)
	}
	if updated.Status != "bounced" {
		t.Fatalf("subscriber status = %q, want bounced", updated.Status)
	}

	suppressed, err := q.IsSuppressed(ctx, gen.IsSuppressedParams{OwnerID: owner, Email: sub.Email})
	if err != nil {
		t.Fatalf("IsSuppressed: %v", err)
	}
	if !suppressed {
		t.Fatalf("expected suppression entry for %s", sub.Email)
	}
}

func TestRecordBounceUnknownEmailIsSilent(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	h := hooks.New()

	svc := tracking.New(q, h)
	if err := svc.RecordBounce(ctx, "nobody@nowhere.test"); err != nil {
		t.Fatalf("RecordBounce with unknown email should not error: %v", err)
	}
}
