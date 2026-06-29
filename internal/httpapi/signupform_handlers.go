package httpapi

import (
	"encoding/json"
	"errors"
	"html/template"
	"net/http"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/signupform"
)

type signupformHandlers struct{ svc *signupform.Service }

type signupformBody struct {
	ListID      string `json:"listId"`
	Name        string `json:"name"`
	Heading     string `json:"heading"`
	Description string `json:"description"`
	ButtonText  string `json:"buttonText"`
}

func (b signupformBody) toInput() (signupform.FormInput, error) {
	lid, err := uuid.Parse(b.ListID)
	if err != nil {
		return signupform.FormInput{}, signupform.ErrInvalid
	}
	return signupform.FormInput{ListID: lid, Name: b.Name, Heading: b.Heading, Description: b.Description, ButtonText: b.ButtonText}, nil
}

func (h signupformHandlers) writeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, signupform.ErrInvalid):
		http.Error(w, "invalid form", http.StatusBadRequest)
	case errors.Is(err, signupform.ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
	default:
		http.Error(w, "server error", http.StatusInternalServerError)
	}
}

func (h signupformHandlers) list(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	forms, err := h.svc.List(r.Context(), owner)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, forms)
}

func (h signupformHandlers) create(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	var b signupformBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	in, err := b.toInput()
	if err != nil {
		h.writeErr(w, err)
		return
	}
	f, err := h.svc.Create(r.Context(), owner, in)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, f)
}

func (h signupformHandlers) get(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	id, err := uuid.Parse(chiURLParam(r, "id"))
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	f, err := h.svc.Get(r.Context(), owner, id)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, f)
}

func (h signupformHandlers) update(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	id, err := uuid.Parse(chiURLParam(r, "id"))
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	var b signupformBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	in, err := b.toInput()
	if err != nil {
		h.writeErr(w, err)
		return
	}
	f, err := h.svc.Update(r.Context(), owner, id, in)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	writeJSON(w, f)
}

func (h signupformHandlers) delete(w http.ResponseWriter, r *http.Request) {
	owner, _ := adminID(r.Context())
	id, err := uuid.Parse(chiURLParam(r, "id"))
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	if err := h.svc.Delete(r.Context(), owner, id); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// landingTmpl auto-escapes Heading/Description/ButtonText (untrusted owner input).
var landingTmpl = template.Must(template.New("landing").Parse(`<!DOCTYPE html>
<html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Heading}}</title>
<style>body{margin:0;font-family:system-ui,sans-serif;background:#f1f5f9;display:flex;min-height:100vh;align-items:center;justify-content:center}
.card{width:100%;max-width:360px;background:#fff;border-radius:14px;box-shadow:0 10px 40px rgba(15,23,42,.12);padding:28px}
.mark{width:40px;height:40px;border-radius:10px;background:#1E40AF;margin-bottom:16px}
h1{font-size:1.25rem;color:#0f172a;margin:0 0 6px}p.desc{color:#64748b;font-size:.9rem;margin:0 0 18px}
input{width:100%;box-sizing:border-box;border:1px solid #e2e8f0;border-radius:9px;padding:11px 13px;font-size:.9rem;margin-bottom:10px}
button{width:100%;background:#D97706;color:#fff;border:none;font-weight:600;font-size:.92rem;padding:12px;border-radius:9px;cursor:pointer}
small{display:block;text-align:center;color:#94a3b8;font-size:.72rem;margin-top:12px}</style></head>
<body><div class="card"><div class="mark"></div>
<h1>{{.Heading}}</h1><p class="desc">{{.Description}}</p>
<form method="post" action="/subscribe/{{.ListID}}">
<input type="email" name="email" placeholder="you@company.com" required>
<button type="submit">{{.ButtonText}}</button></form>
<small>Double opt-in · GDPR ready</small></div></body></html>`))

func (h signupformHandlers) landing(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chiURLParam(r, "id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	f, err := h.svc.GetPublic(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = landingTmpl.Execute(w, f)
}
