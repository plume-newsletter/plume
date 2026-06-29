package team_test

import (
	"context"
	"testing"

	"github.com/plume-newsletter/plume/internal/auth"
	"github.com/plume-newsletter/plume/internal/email"
	"github.com/plume-newsletter/plume/internal/store/gen"
	"github.com/plume-newsletter/plume/internal/team"
	"github.com/plume-newsletter/plume/internal/testsupport"
)

func TestTeamInviteAcceptAndGuards(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := gen.New(pool)
	ctx := context.Background()
	if err := auth.EnsureAdmin(ctx, q, "owner@x.test", "pw123456"); err != nil {
		t.Fatal(err)
	}
	owner, _ := q.GetAdminByEmail(ctx, "owner@x.test")
	ws := owner.WorkspaceID

	svc := team.New(q, email.NoopResolver(), "http://x") // see note: use a no-op resolver

	inv, url, err := svc.Invite(ctx, ws, "Editor@X.test", "editor")
	if err != nil {
		t.Fatalf("invite: %v", err)
	}
	if url == "" || inv.Email != "editor@x.test" || inv.Role != "editor" {
		t.Fatalf("bad invite: %+v / %q", inv, url)
	}

	pub, err := svc.GetInvite(ctx, inv.Token)
	if err != nil || pub.Email != "editor@x.test" {
		t.Fatalf("getinvite: %v / %+v", err, pub)
	}

	newUser, err := svc.AcceptInvite(ctx, inv.Token, "Ed Itor", "pw654321")
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	if newUser.WorkspaceID != ws || newUser.Role != "editor" {
		t.Fatalf("new user not in workspace/role: %+v", newUser)
	}
	// token can't be reused
	if _, err := svc.AcceptInvite(ctx, inv.Token, "x", "pw000000"); err == nil {
		t.Error("reused token accepted")
	}

	members, _ := svc.Members(ctx, ws)
	if len(members) != 2 {
		t.Fatalf("members = %d, want 2", len(members))
	}

	// guards: invite cannot grant owner
	if _, _, err := svc.Invite(ctx, ws, "x@x.test", "owner"); err != team.ErrInvalid {
		t.Errorf("owner invite = %v, want ErrInvalid", err)
	}
	// last-owner guard: demoting the only owner fails
	if err := svc.SetRole(ctx, ws, owner.ID, "editor"); err != team.ErrNotAllowed {
		t.Errorf("demote last owner = %v, want ErrNotAllowed", err)
	}
	// can't remove self
	if err := svc.RemoveMember(ctx, ws, owner.ID, owner.ID); err == nil {
		t.Error("removed self")
	}
	// remove the editor (ok)
	if err := svc.RemoveMember(ctx, ws, newUser.ID, owner.ID); err != nil {
		t.Fatalf("remove editor: %v", err)
	}
}
