// Package team manages workspace membership: members, role changes, removal,
// and the invite/accept flow. Team mutations are gated to owner/admin at the
// handler layer; the service enforces structural guards (last owner, no second
// owner via the API, no self-removal).
package team

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/plume-newsletter/plume/internal/auth"
	"github.com/plume-newsletter/plume/internal/email"
	"github.com/plume-newsletter/plume/internal/store/gen"
)

var ErrNotAllowed = errors.New("not allowed")
var ErrInvalid = errors.New("invalid")
var ErrNotFound = errors.New("not found")

type Member struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	FullName string `json:"fullName"`
	Role     string `json:"role"`
}
type Invite struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Token     string `json:"token"`
	ExpiresAt string `json:"expiresAt"`
	Accepted  bool   `json:"accepted"`
}
type InvitePublic struct {
	Email         string `json:"email"`
	WorkspaceName string `json:"workspaceName"`
}

type Service struct {
	q        *gen.Queries
	resolver email.Resolver
	baseURL  string
}

func New(q *gen.Queries, resolver email.Resolver, baseURL string) *Service {
	return &Service{q: q, resolver: resolver, baseURL: baseURL}
}

func normEmail(e string) string { return strings.ToLower(strings.TrimSpace(e)) }

func validInviteRole(role string) bool {
	return role == "admin" || role == "editor" || role == "viewer" // never owner via API
}

func newToken() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func (s *Service) Members(ctx context.Context, workspace uuid.UUID) ([]Member, error) {
	rows, err := s.q.ListUsersByWorkspace(ctx, workspace)
	if err != nil {
		return nil, err
	}
	out := make([]Member, 0, len(rows))
	for _, u := range rows {
		out = append(out, Member{ID: u.ID.String(), Email: u.Email, FullName: u.FullName, Role: u.Role})
	}
	return out, nil
}

func (s *Service) Invite(ctx context.Context, workspace uuid.UUID, addr, role string) (Invite, string, error) {
	addr = normEmail(addr)
	if addr == "" || !validInviteRole(role) {
		return Invite{}, "", ErrInvalid
	}
	// Reject inviting an email that already belongs to a user (single workspace),
	// so the error surfaces as 400 here rather than a 500 at accept time.
	if _, err := s.q.GetAdminByEmail(ctx, addr); err == nil {
		return Invite{}, "", ErrInvalid
	}
	token := newToken()
	row, err := s.q.CreateInvite(ctx, gen.CreateInviteParams{
		ID: uuid.New(), WorkspaceID: workspace, Email: addr, Role: role,
		Token: token, ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	})
	if err != nil {
		return Invite{}, "", err
	}
	acceptURL := fmt.Sprintf("%s/accept/%s", s.baseURL, token)
	s.sendInvite(ctx, workspace, addr, acceptURL) // best-effort
	return Invite{ID: row.ID.String(), Email: row.Email, Role: row.Role, Token: row.Token, ExpiresAt: row.ExpiresAt.Format(time.RFC3339)}, acceptURL, nil
}

// sendInvite best-effort emails the accept link using the workspace's first
// brand as the verified sender. Any failure (no brand, no provider, send error)
// is logged and ignored — the link is always returned to the caller.
func (s *Service) sendInvite(ctx context.Context, workspace uuid.UUID, addr, acceptURL string) {
	brands, err := s.q.ListBrandsByOwner(ctx, workspace)
	if err != nil || len(brands) == 0 {
		return
	}
	provider, err := s.resolver.Provider(ctx)
	if err != nil {
		log.Printf("team: resolve provider: %v", err)
		return
	}
	b := brands[0]
	// SECURITY: only the trusted acceptURL (baseURL + random token) is interpolated.
	msg := email.Message{
		From: b.FromEmail, FromName: b.FromName, ReplyTo: b.ReplyTo, To: addr,
		Subject: "You're invited to join the workspace",
		HTML:    fmt.Sprintf(`<p>You've been invited. Click to accept:</p><p><a href="%s">%s</a></p>`, acceptURL, acceptURL),
	}
	if err := provider.Send(ctx, msg); err != nil {
		log.Printf("team: send invite: %v", err)
	}
}

func (s *Service) ListInvites(ctx context.Context, workspace uuid.UUID) ([]Invite, error) {
	rows, err := s.q.ListInvitesByWorkspace(ctx, workspace)
	if err != nil {
		return nil, err
	}
	out := make([]Invite, 0, len(rows))
	for _, r := range rows {
		out = append(out, Invite{ID: r.ID.String(), Email: r.Email, Role: r.Role, Token: r.Token, ExpiresAt: r.ExpiresAt.Format(time.RFC3339)})
	}
	return out, nil
}

func (s *Service) RevokeInvite(ctx context.Context, workspace, id uuid.UUID) error {
	return s.q.DeleteInvite(ctx, gen.DeleteInviteParams{ID: id, WorkspaceID: workspace})
}

func (s *Service) SetRole(ctx context.Context, workspace, memberID uuid.UUID, role string) error {
	if !validInviteRole(role) {
		return ErrInvalid
	}
	cur, err := s.q.GetAdminByID(ctx, memberID)
	if err != nil || cur.WorkspaceID != workspace {
		return ErrNotFound
	}
	if cur.Role == "owner" { // demoting an owner — guard the last owner
		owners, err := s.q.CountOwners(ctx, workspace)
		if err != nil {
			return err
		}
		if owners <= 1 {
			return ErrNotAllowed
		}
	}
	_, err = s.q.UpdateUserRole(ctx, gen.UpdateUserRoleParams{ID: memberID, WorkspaceID: workspace, Role: role})
	return err
}

func (s *Service) RemoveMember(ctx context.Context, workspace, memberID, actingUser uuid.UUID) error {
	if memberID == actingUser {
		return ErrNotAllowed // can't remove self
	}
	cur, err := s.q.GetAdminByID(ctx, memberID)
	if err != nil || cur.WorkspaceID != workspace {
		return ErrNotFound
	}
	if cur.Role == "owner" {
		owners, err := s.q.CountOwners(ctx, workspace)
		if err != nil {
			return err
		}
		if owners <= 1 {
			return ErrNotAllowed
		}
	}
	return s.q.DeleteUser(ctx, gen.DeleteUserParams{ID: memberID, WorkspaceID: workspace})
}

func (s *Service) GetInvite(ctx context.Context, token string) (InvitePublic, error) {
	inv, err := s.q.GetInviteByToken(ctx, token)
	if err != nil {
		return InvitePublic{}, ErrNotFound
	}
	if inv.AcceptedAt.Valid || time.Now().After(inv.ExpiresAt) {
		return InvitePublic{}, ErrNotFound
	}
	ws, err := s.q.GetWorkspace(ctx, inv.WorkspaceID)
	if err != nil {
		return InvitePublic{}, err
	}
	return InvitePublic{Email: inv.Email, WorkspaceName: ws.Name}, nil
}

func (s *Service) AcceptInvite(ctx context.Context, token, fullName, password string) (gen.AdminUser, error) {
	inv, err := s.q.GetInviteByToken(ctx, token)
	if err != nil {
		return gen.AdminUser{}, ErrNotFound
	}
	if inv.AcceptedAt.Valid || time.Now().After(inv.ExpiresAt) {
		return gen.AdminUser{}, ErrNotFound
	}
	if len(password) < 8 {
		return gen.AdminUser{}, ErrInvalid
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return gen.AdminUser{}, err
	}
	user, err := s.q.CreateUser(ctx, gen.CreateUserParams{
		ID: uuid.New(), Email: inv.Email, PasswordHash: hash,
		FullName: fullName, Role: inv.Role, WorkspaceID: inv.WorkspaceID,
	})
	if err != nil {
		return gen.AdminUser{}, err
	}
	_ = s.q.MarkInviteAccepted(ctx, inv.ID)
	return user, nil
}

func (s *Service) Workspace(ctx context.Context, workspace uuid.UUID) (string, error) {
	ws, err := s.q.GetWorkspace(ctx, workspace)
	if err != nil {
		return "", err
	}
	return ws.Name, nil
}

func (s *Service) RenameWorkspace(ctx context.Context, workspace uuid.UUID, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalid
	}
	return s.q.UpdateWorkspaceName(ctx, gen.UpdateWorkspaceNameParams{ID: workspace, Name: name})
}
