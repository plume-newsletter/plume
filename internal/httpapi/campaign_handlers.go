package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/campaign"
	"github.com/plume-newsletter/plume/internal/store/gen"
)

type campaignHandlers struct{ svc *campaign.Service }

// campaignResp serializes a campaign with body_json as a raw JSON string
// (gen.Campaign.BodyJson is []byte, which json.Marshal would base64-encode).
// The outer BodyJson field (depth 0) dominates the embedded one (depth 1).
type campaignResp struct {
	gen.Campaign
	BodyJson string `json:"body_json"`
}

func campaignToResp(c gen.Campaign) campaignResp {
	return campaignResp{Campaign: c, BodyJson: string(c.BodyJson)}
}

func (h campaignHandlers) list(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	campaigns, err := h.svc.List(r.Context(), owner)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	resps := make([]campaignResp, len(campaigns))
	for i, c := range campaigns {
		resps[i] = campaignToResp(c)
	}
	writeJSON(w, resps)
}

func (h campaignHandlers) create(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	var body struct {
		BrandID   string `json:"brandId"`
		Subject   string `json:"subject"`
		HtmlBody  string `json:"htmlBody"`
		PlainBody string `json:"plainBody"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Subject == "" {
		http.Error(w, "subject is required", http.StatusBadRequest)
		return
	}
	brandID, err := uuid.Parse(body.BrandID)
	if err != nil {
		http.Error(w, "bad brandId", http.StatusBadRequest)
		return
	}
	created, err := h.svc.Create(r.Context(), owner, brandID, campaign.CampaignInput{
		Subject: body.Subject, HtmlBody: body.HtmlBody, PlainBody: body.PlainBody,
	})
	if errors.Is(err, campaign.ErrBrandNotFound) {
		http.Error(w, "brand not found", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, campaignToResp(created))
}

func (h campaignHandlers) get(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	id, err := uuid.Parse(chiURLParam(r, "id"))
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	c, err := h.svc.Get(r.Context(), owner, id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, campaignToResp(c))
}

func (h campaignHandlers) update(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	id, err := uuid.Parse(chiURLParam(r, "id"))
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	var body struct {
		Subject  string          `json:"subject"`
		BodyJSON json.RawMessage `json:"bodyJson"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Subject == "" {
		http.Error(w, "subject is required", http.StatusBadRequest)
		return
	}
	c, err := h.svc.Update(r.Context(), owner, id, body.Subject, []byte(body.BodyJSON))
	if err != nil {
		http.Error(w, "could not update", http.StatusBadRequest)
		return
	}
	writeJSON(w, campaignToResp(c))
}

func (h campaignHandlers) delete(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	id, err := uuid.Parse(chiURLParam(r, "id"))
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	if err := h.svc.Delete(r.Context(), owner, id); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
