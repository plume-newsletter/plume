package httpapi

import (
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/unsubscribe"
)

type unsubscribeHandlers struct{ svc *unsubscribe.Service }

// confirmPage serves GET /u/{recipientId}: a confirmation page with a button
// that POSTs to the same URL. It performs NO mutation and makes no DB call,
// so it is safe against email-client / security-scanner link prefetch. A
// bad/unknown recipientId still renders the same generic page (no leak).
func (h unsubscribeHandlers) confirmPage(w http.ResponseWriter, r *http.Request) {
	recipientID := chiURLParam(r, "recipientId")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `<!DOCTYPE html><html><body>
<p>Click to confirm unsubscribe.</p>
<form method="POST" action="/u/%s"><button type="submit">Unsubscribe</button></form>
</body></html>`, recipientID)
}

// action serves POST /u/{recipientId}: it performs the unsubscribe and
// always returns a generic confirmation page. A malformed recipientId skips
// the service call (nothing to unsubscribe); a real error is logged but
// never surfaced, so the public endpoint never leaks anything and never
// fails the visitor's request.
func (h unsubscribeHandlers) action(w http.ResponseWriter, r *http.Request) {
	if id, err := uuid.Parse(chiURLParam(r, "recipientId")); err == nil {
		if err := h.svc.Unsubscribe(r.Context(), id); err != nil {
			log.Printf("unsubscribe: Unsubscribe: %v", err)
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`<!DOCTYPE html><html><body><p>You have been unsubscribed.</p></body></html>`))
}
