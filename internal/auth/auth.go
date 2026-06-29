package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/plume-newsletter/plume/internal/store/gen"
)

func normalize(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

// EnsureAdmin creates the bootstrap workspace + owner user from email/password
// if none exists yet. No-op when an admin already exists or either value is empty.
//
// The workspace is created with workspace.id == user.id (mirroring the production
// migration which does INSERT INTO workspace SELECT id FROM admin_user). This
// preserves the scoping invariant: adminID(ctx) returns workspace_id, which for
// the bootstrap owner equals its own id, which equals every existing owner_id.
func EnsureAdmin(ctx context.Context, q *gen.Queries, email, password string) error {
	if email == "" || password == "" {
		return nil
	}
	n, err := q.CountAdmins(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	// Use the same UUID for user and workspace so workspace.id == user.id,
	// exactly as the production migration does. Services that call
	// GetAdminByID(ctx, adminID(ctx)) continue to work because adminID(ctx)
	// returns workspace_id == user.id.
	uid := uuid.New()
	if _, err := q.CreateWorkspace(ctx, gen.CreateWorkspaceParams{ID: uid, Name: "My workspace"}); err != nil {
		return err
	}
	_, err = q.CreateUser(ctx, gen.CreateUserParams{
		ID:           uid,
		Email:        normalize(email),
		PasswordHash: hash,
		FullName:     "",
		Role:         "owner",
		WorkspaceID:  uid,
	})
	return err
}

// Validate returns the admin and true when the email exists and the password matches.
func Validate(ctx context.Context, q *gen.Queries, email, password string) (gen.AdminUser, bool, error) {
	user, err := q.GetAdminByEmail(ctx, normalize(email))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return gen.AdminUser{}, false, nil // not found → invalid credentials, not an error
		}
		return gen.AdminUser{}, false, err // real DB error → propagate
	}
	if !VerifyPassword(password, user.PasswordHash) {
		return gen.AdminUser{}, false, nil
	}
	return user, true, nil
}
