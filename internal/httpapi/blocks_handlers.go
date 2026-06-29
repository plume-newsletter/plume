package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/plume-newsletter/plume/internal/blocks"
)

type blocksHandlers struct{}

func (blocksHandlers) render(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Blocks []blocks.Block `json:"blocks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	html, _, err := blocks.Render(body.Blocks)
	if err != nil {
		http.Error(w, "render failed", http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]string{"html": html})
}
