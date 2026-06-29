// Package analytics aggregates an owner's recipient and email_event rows into
// an owner-level Overview (subscriber totals, engagement rates, pure SES send
// cost, growth/volume series, best send times, and per-campaign metrics).
package analytics

import (
	"context"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/store/gen"
)

const sesPer1K = 0.10 // pure SES send cost; dev tooling/subscriptions are not counted.

var weekday = [7]string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}

// Point is one day's subscriber growth or send-volume data.
type Point struct {
	Date   string `json:"date"`
	Gained int    `json:"gained,omitempty"`
	Lost   int    `json:"lost,omitempty"`
	Sent   int    `json:"sent,omitempty"`
	Opens  int    `json:"opens,omitempty"`
}

// SendTime is one (dow, hour) bucket labelled for display.
type SendTime struct {
	Label string  `json:"label"`
	Rate  float64 `json:"rate"`
}

// CampaignMetric is per-campaign engagement data for the campaigns list.
type CampaignMetric struct {
	ID        string  `json:"id"`
	Subject   string  `json:"subject"`
	Status    string  `json:"status"`
	Sent      int     `json:"sent"`
	OpenRate  float64 `json:"openRate"`
	ClickRate float64 `json:"clickRate"`
}

// TopCampaign is a summary entry for the top-campaigns-by-opens widget.
type TopCampaign struct {
	ID      string `json:"id"`
	Subject string `json:"subject"`
	Opens   int    `json:"opens"`
}

// Overview is the full analytics payload returned by GET /api/analytics/overview.
type Overview struct {
	Subscribers      int              `json:"subscribers"`
	NetNewSubs       int              `json:"netNewSubs"`
	AvgOpenRate      float64          `json:"avgOpenRate"`
	ClickRate        float64          `json:"clickRate"`
	SendCost         float64          `json:"sendCost"`
	SubscriberGrowth []Point          `json:"subscriberGrowth"`
	SendVolume       []Point          `json:"sendVolume"`
	BestSendTimes    []SendTime       `json:"bestSendTimes"`
	Campaigns        []CampaignMetric `json:"campaigns"`
	TopCampaigns     []TopCampaign    `json:"topCampaigns"`
}

// Service aggregates analytics queries into an owner-level Overview.
type Service struct{ q *gen.Queries }

// New returns a new Service backed by the given query set.
func New(q *gen.Queries) *Service { return &Service{q: q} }

func rate(num, den int64) float64 {
	if den == 0 {
		return 0
	}
	r := float64(num) / float64(den)
	if r > 1 {
		// Window-boundary artifact: an in-window open/click on a campaign sent
		// before the window inflates the numerator past the in-window sends.
		return 1
	}
	return r
}

// hourLabel formats a (dow 0=Sun..6=Sat, 24h hour) bucket as e.g. "Tue 9 AM".
func hourLabel(dow, hour int32) string {
	h, suffix := hour, "AM"
	switch {
	case hour == 0:
		h = 12
	case hour == 12:
		suffix = "PM"
	case hour > 12:
		h, suffix = hour-12, "PM"
	}
	d := "?"
	if dow >= 0 && int(dow) < len(weekday) {
		d = weekday[dow]
	}
	return d + " " + strconv.Itoa(int(h)) + " " + suffix
}

// Overview returns the analytics overview for owner over the given window (30 or 90 days).
func (s *Service) Overview(ctx context.Context, owner uuid.UUID, windowDays int) (Overview, error) {
	if windowDays != 90 {
		windowDays = 30
	}
	since := time.Now().AddDate(0, 0, -windowDays)
	weekAgo := time.Now().AddDate(0, 0, -7)
	var ov Overview

	subs, err := s.q.CountActiveSubscribersForOwner(ctx, owner)
	if err != nil {
		return ov, err
	}
	ov.Subscribers = int(subs)

	net, err := s.q.CountSubscribersCreatedSince(ctx, gen.CountSubscribersCreatedSinceParams{OwnerID: owner, CreatedAt: since})
	if err != nil {
		return ov, err
	}
	ov.NetNewSubs = int(net)

	sent, err := s.q.CountSentForOwnerSince(ctx, gen.CountSentForOwnerSinceParams{OwnerID: owner, SentAt: since})
	if err != nil {
		return ov, err
	}
	openers, err := s.q.CountDistinctOpenersForOwnerSince(ctx, gen.CountDistinctOpenersForOwnerSinceParams{OwnerID: owner, CreatedAt: since})
	if err != nil {
		return ov, err
	}
	clicks, err := s.q.CountEventsForOwnerSince(ctx, gen.CountEventsForOwnerSinceParams{OwnerID: owner, Type: "click", CreatedAt: since})
	if err != nil {
		return ov, err
	}
	ov.AvgOpenRate = rate(openers, sent)
	ov.ClickRate = rate(clicks, sent)
	ov.SendCost = float64(sent) / 1000.0 * sesPer1K

	// subscriber growth: merge gained + lost by date
	growth := map[string]*Point{}
	gained, err := s.q.SubscriberGainedByDay(ctx, gen.SubscriberGainedByDayParams{OwnerID: owner, CreatedAt: since})
	if err != nil {
		return ov, err
	}
	for _, g := range gained {
		d := g.Day.Format("2006-01-02")
		growth[d] = &Point{Date: d, Gained: int(g.N)}
	}
	lostRows, err := s.q.SubscriberLostByDay(ctx, gen.SubscriberLostByDayParams{OwnerID: owner, CreatedAt: since})
	if err != nil {
		return ov, err
	}
	for _, l := range lostRows {
		d := l.Day.Format("2006-01-02")
		if growth[d] == nil {
			growth[d] = &Point{Date: d}
		}
		growth[d].Lost = int(l.N)
	}
	ov.SubscriberGrowth = sortedPoints(growth)

	// send volume (last 7 days): merge sent + opens by date
	vol := map[string]*Point{}
	sv, err := s.q.SendVolumeByDay(ctx, gen.SendVolumeByDayParams{OwnerID: owner, SentAt: weekAgo})
	if err != nil {
		return ov, err
	}
	for _, v := range sv {
		d := v.Day.Format("2006-01-02")
		vol[d] = &Point{Date: d, Sent: int(v.Sent)}
	}
	od, err := s.q.OpensByDay(ctx, gen.OpensByDayParams{OwnerID: owner, CreatedAt: weekAgo})
	if err != nil {
		return ov, err
	}
	for _, o := range od {
		d := o.Day.Format("2006-01-02")
		if vol[d] == nil {
			vol[d] = &Point{Date: d}
		}
		vol[d].Opens = int(o.Opens)
	}
	ov.SendVolume = sortedPoints(vol)

	// best send times: rate is relative to the top bucket
	times, err := s.q.OpensByWeekdayHour(ctx, gen.OpensByWeekdayHourParams{OwnerID: owner, CreatedAt: since})
	if err != nil {
		return ov, err
	}
	if len(times) > 0 {
		top := float64(times[0].N)
		for _, t := range times {
			ov.BestSendTimes = append(ov.BestSendTimes, SendTime{Label: hourLabel(t.Dow, t.Hour), Rate: float64(t.N) / top})
		}
	}

	camps, err := s.q.CampaignsWithMetrics(ctx, owner)
	if err != nil {
		return ov, err
	}
	for _, c := range camps {
		ov.Campaigns = append(ov.Campaigns, CampaignMetric{
			ID: c.ID.String(), Subject: c.Subject, Status: c.Status, Sent: int(c.Sent),
			OpenRate: rate(c.UniqueOpens, c.Sent), ClickRate: rate(c.Clicks, c.Sent),
		})
	}

	topRows, err := s.q.TopCampaignsByOpens(ctx, gen.TopCampaignsByOpensParams{OwnerID: owner, CreatedAt: since})
	if err != nil {
		return ov, err
	}
	for _, c := range topRows {
		ov.TopCampaigns = append(ov.TopCampaigns, TopCampaign{ID: c.ID.String(), Subject: c.Subject, Opens: int(c.Opens)})
	}
	return ov, nil
}

func sortedPoints(m map[string]*Point) []Point {
	out := make([]Point, 0, len(m))
	for _, p := range m {
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date < out[j].Date })
	return out
}
