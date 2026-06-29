package httpapi

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/subscriber"
)

type importHandlers struct{ svc *subscriber.Service }

func (h importHandlers) importCSV(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	listID, err := uuid.Parse(chiURLParam(r, "listId"))
	if err != nil {
		http.Error(w, "bad listId", http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing 'file' upload", http.StatusBadRequest)
		return
	}
	defer file.Close()

	res, err := h.svc.ImportCSV(r.Context(), owner, listID, file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, res)
}
