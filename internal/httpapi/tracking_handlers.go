package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/tracking"
)

type trackingHandlers struct{ svc *tracking.Service }

// onePixelGIF is a 1x1 transparent GIF (43 bytes), the smallest valid GIF89a
// image, served by the open-tracking pixel handler.
var onePixelGIF = []byte{
	0x47, 0x49, 0x46, 0x38, 0x39, 0x61, // GIF89a
	0x01, 0x00, 0x01, 0x00, // width=1, height=1
	0x80, 0x00, 0x00, // packed fields, background color index, pixel aspect ratio
	0xff, 0xff, 0xff, 0x00, 0x00, 0x00, // global color table: white, black
	0x21, 0xf9, 0x04, 0x01, 0x00, 0x00, 0x00, 0x00, // graphic control extension (transparent index 0)
	0x2c, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, // image descriptor
	0x02, 0x02, 0x44, 0x01, 0x00, // image data
	0x3b, // trailer
}

// open serves GET /t/{recipientId}: it always returns the tracking pixel,
// even for an unknown or malformed recipient id, so the public endpoint never
// leaks whether an id is valid.
func (h trackingHandlers) open(w http.ResponseWriter, r *http.Request) {
	if id, err := uuid.Parse(chiURLParam(r, "recipientId")); err == nil {
		if err := h.svc.RecordOpen(r.Context(), id); err != nil {
			log.Printf("tracking: RecordOpen: %v", err)
		}
	}
	w.Header().Set("Content-Type", "image/gif")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(onePixelGIF)
}

// click serves GET /l/{linkId}/{recipientId}: it redirects to the link's
// destination URL. An unknown link id returns 404 (there is nothing to
// redirect to); a malformed id is treated the same way.
func (h trackingHandlers) click(w http.ResponseWriter, r *http.Request) {
	linkID, err := uuid.Parse(chiURLParam(r, "linkId"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	recipientID, err := uuid.Parse(chiURLParam(r, "recipientId"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	url, err := h.svc.RecordClick(r.Context(), linkID, recipientID)
	if errors.Is(err, tracking.ErrLinkNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, url, http.StatusFound)
}

// sesEnvelope is the outer SNS notification envelope. SNS signature
// verification (the X-Amz-SNS-* headers / SigningCertURL) is deferred to a
// Phase 5 hardening pass; for now we parse and act on the payload as-is.
type sesEnvelope struct {
	Type         string `json:"Type"`
	SubscribeURL string `json:"SubscribeURL"`
	Message      string `json:"Message"`
}

// sesMessage is the inner SES event notification (the envelope's Message
// field, itself JSON-encoded).
type sesMessage struct {
	NotificationType string `json:"notificationType"`
	Bounce           struct {
		BouncedRecipients []sesRecipient `json:"bouncedRecipients"`
	} `json:"bounce"`
	Complaint struct {
		ComplainedRecipients []sesRecipient `json:"complainedRecipients"`
	} `json:"complaint"`
}

type sesRecipient struct {
	EmailAddress string `json:"emailAddress"`
}

// sesWebhook serves POST /webhook/ses: it parses the SNS envelope and, for
// bounce/complaint notifications, records the event against every affected
// subscriber email. It always returns 200 so SNS does not retry-storm on a
// payload we can't (or choose not to) process.
func (h trackingHandlers) sesWebhook(w http.ResponseWriter, r *http.Request) {
	defer func() { w.WriteHeader(http.StatusOK) }()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("tracking: ses webhook: read body: %v", err)
		return
	}

	var env sesEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		log.Printf("tracking: ses webhook: bad envelope: %v", err)
		return
	}

	switch env.Type {
	case "SubscriptionConfirmation":
		// Confirming the subscription requires a human (or an operator script)
		// to visit SubscribeURL; we just log it so it's discoverable in
		// server output during setup.
		log.Printf("tracking: ses webhook: SNS SubscriptionConfirmation, visit to confirm: %s", env.SubscribeURL)
	case "Notification":
		var msg sesMessage
		if err := json.Unmarshal([]byte(env.Message), &msg); err != nil {
			log.Printf("tracking: ses webhook: bad inner message: %v", err)
			return
		}
		switch msg.NotificationType {
		case "Bounce":
			for _, rec := range msg.Bounce.BouncedRecipients {
				if err := h.svc.RecordBounce(r.Context(), rec.EmailAddress); err != nil {
					log.Printf("tracking: RecordBounce(%s): %v", rec.EmailAddress, err)
				}
			}
		case "Complaint":
			for _, rec := range msg.Complaint.ComplainedRecipients {
				if err := h.svc.RecordComplaint(r.Context(), rec.EmailAddress); err != nil {
					log.Printf("tracking: RecordComplaint(%s): %v", rec.EmailAddress, err)
				}
			}
		}
	}
}
