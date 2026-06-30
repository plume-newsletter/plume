package httpapi

import (
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/plume-newsletter/plume/web"
)

// NewRouter builds the HTTP handler: JSON API under /api and the embedded SPA elsewhere.
func NewRouter(authDeps AuthDeps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("ok"))
	})

	// Declare sfh once so both the public landing and the authed CRUD can use it.
	sfh := signupformHandlers{svc: authDeps.SignupForms}

	// Team handlers: used in both the public invite routes and the authed group.
	tmh := teamHandlers{svc: authDeps.Team, cookie: authDeps.Cookie, secure: authDeps.Secure}

	// Public tracking endpoints: no auth, mounted at the router root (not
	// under /api) so the URLs match exactly what render.Register emits.
	th := trackingHandlers{svc: authDeps.Tracking}
	r.Get("/t/{recipientId}", th.open)
	r.Get("/l/{linkId}/{recipientId}", th.click)
	r.Post("/webhook/ses", th.sesWebhook)

	uh := unsubscribeHandlers{svc: authDeps.Unsubscribe}
	r.Get("/u/{recipientId}", uh.confirmPage)
	r.Post("/u/{recipientId}", uh.action)

	suh := signupHandlers{svc: authDeps.Signup}
	r.Post("/subscribe/{listId}", suh.subscribe)
	r.Get("/confirm/{subscriberId}", suh.confirm)

	// Public hosted landing page — no auth required.
	r.Get("/f/{id}", sfh.landing)

	r.Route("/api", func(api chi.Router) {
		api.Post("/login", authDeps.login)
		api.Post("/logout", authDeps.logout)
		api.Get("/me", authDeps.me)

		// Public invite routes — no session required.
		api.Get("/invites/{token}", tmh.inviteInfo)
		api.Post("/invites/{token}/accept", tmh.acceptInvite)

		api.Group(func(pr chi.Router) {
			pr.Use(requireAuth(authDeps.Cookie, authDeps.Queries, authDeps.APIKeys))
			bh := brandHandlers{svc: authDeps.Brands}
			pr.Get("/brands", bh.list)
			pr.Post("/brands", bh.create)
			pr.Get("/brands/{id}", bh.get)
			pr.Put("/brands/{id}", bh.update)
			pr.Delete("/brands/{id}", bh.delete)

			lh := listHandlers{svc: authDeps.Lists}
			pr.Get("/lists", lh.list)
			pr.Post("/lists", lh.create)
			pr.Get("/lists/{id}", lh.get)
			pr.Put("/lists/{id}", lh.update)
			pr.Delete("/lists/{id}", lh.delete)

			sh := subscriberHandlers{svc: authDeps.Subscribers}
			pr.Post("/lists/{listId}/subscribers", sh.add)
			pr.Get("/lists/{listId}/subscribers", sh.list)
			pr.Put("/subscribers/{id}/status", sh.setStatus)
			pr.Delete("/subscribers/{id}", sh.delete)

			ih := importHandlers{svc: authDeps.Subscribers}
			pr.Post("/lists/{listId}/import", ih.importCSV)

			ch := campaignHandlers{svc: authDeps.Campaigns}
			pr.Get("/campaigns", ch.list)
			pr.Post("/campaigns", ch.create)
			pr.Get("/campaigns/{id}", ch.get)
			pr.Put("/campaigns/{id}", ch.update)
			pr.Delete("/campaigns/{id}", ch.delete)

			snh := sendingHandlers{svc: authDeps.Sending}
			pr.Post("/campaigns/{id}/send", snh.send)
			pr.Post("/campaigns/{id}/test", snh.sendTest)

			rh := reportHandlers{svc: authDeps.Reports}
			pr.Get("/campaigns/{id}/report", rh.campaign)

			pr.Get("/signup-forms", sfh.list)
			pr.Post("/signup-forms", sfh.create)
			pr.Get("/signup-forms/{id}", sfh.get)
			pr.Put("/signup-forms/{id}", sfh.update)
			pr.Delete("/signup-forms/{id}", sfh.delete)

			anh := analyticsHandlers{svc: authDeps.Analytics}
			pr.Get("/analytics/overview", anh.overview)
			pr.Get("/analytics/deliverability", anh.deliverability)

			sth := settingsHandlers{svc: authDeps.Settings}
			pr.Get("/settings", sth.get)
			pr.Put("/settings/ses", sth.setSES)
			pr.Put("/settings/ai", sth.setAI)

			akh := apikeyHandlers{svc: authDeps.APIKeys}
			pr.Get("/api-keys", akh.list)
			pr.Post("/api-keys", akh.create)
			pr.Delete("/api-keys/{id}", akh.delete)

			whh := webhookHandlers{svc: authDeps.Webhooks}
			pr.Get("/webhooks", whh.list)
			pr.Post("/webhooks", whh.create)
			pr.Delete("/webhooks/{id}", whh.delete)

			aih := aiHandlers{ai: authDeps.AI, cfg: authDeps.Settings, analytics: authDeps.Analytics, segments: authDeps.Segments}
			pr.Post("/ai/rewrite", aih.rewrite)
			pr.Post("/ai/chat", aih.chat)
			pr.Post("/ai/suggest", aih.suggest)
			pr.Post("/ai/insights", aih.insights)
			pr.Post("/ai/segment-rules", aih.segmentRules)

			bkh := blocksHandlers{}
			pr.Post("/blocks/render", bkh.render)

			sgh := segmentHandlers{svc: authDeps.Segments}
			pr.Post("/segments/preview", sgh.preview)
			pr.Get("/segments/fields", sgh.fields)
			pr.Get("/segments", sgh.list)
			pr.Post("/segments", sgh.create)
			pr.Get("/segments/{id}", sgh.get)
			pr.Put("/segments/{id}", sgh.update)
			pr.Delete("/segments/{id}", sgh.delete)

			pr.Get("/team", tmh.members)
			pr.Post("/team/invites", tmh.invite)
			pr.Get("/team/invites", tmh.listInvites)
			pr.Delete("/team/invites/{id}", tmh.revokeInvite)
			pr.Put("/team/members/{id}/role", tmh.setRole)
			pr.Delete("/team/members/{id}", tmh.removeMember)
			pr.Get("/workspace", tmh.getWorkspace)
			pr.Put("/workspace", tmh.renameWorkspace)

			abh := abtestHandlers{svc: authDeps.ABTests}
			pr.Get("/ab-tests", abh.list)
			pr.Post("/ab-tests", abh.create)
			pr.Get("/ab-tests/{id}", abh.get)
			pr.Delete("/ab-tests/{id}", abh.del)
			pr.Post("/ab-tests/{id}/start", abh.start)
			pr.Get("/ab-tests/{id}/results", abh.results)
			pr.Post("/ab-tests/{id}/winner", abh.winner)

			amh := automationHandlers{svc: authDeps.Automations}
			pr.Get("/automations", amh.list)
			pr.Post("/automations", amh.create)
			pr.Get("/automations/{id}", amh.get)
			pr.Put("/automations/{id}", amh.update)
			pr.Delete("/automations/{id}", amh.del)
			pr.Put("/automations/{id}/steps", amh.steps)
			pr.Post("/automations/{id}/status", amh.status)

			tph := templateHandlers{svc: authDeps.Templates}
			pr.Get("/templates", tph.list)
			pr.Post("/templates", tph.create)
			pr.Delete("/templates/{id}", tph.del)
			pr.Post("/templates/{id}/use", tph.use)
		})
	})

	r.Handle("/*", spaHandler())
	return r
}

func spaHandler() http.Handler {
	dist, err := fs.Sub(web.Dist, "dist")
	if err != nil {
		panic(err) // embed is compile-time guaranteed; a failure here is a build bug
	}
	fileServer := http.FileServer(http.FS(dist))
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Serve the file if it exists; otherwise fall back to index.html (SPA routing).
		if _, err := fs.Stat(dist, normalizePath(req.URL.Path)); err != nil {
			req = req.Clone(req.Context())
			req.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, req)
	})
}

func normalizePath(p string) string {
	if p == "/" || p == "" {
		return "index.html"
	}
	return p[1:] // strip leading slash for fs lookup
}
