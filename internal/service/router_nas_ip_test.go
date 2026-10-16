package service

import (
	"context"
	"errors"
	"testing"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

type nasIPRouterRepo struct {
	repository.RouterRepository
	router  *model.Router
	other   *model.Router
	err     error
	created *model.Router
	updated *model.Router
}

func (r *nasIPRouterRepo) FindByVPNIP(context.Context, string) (*model.Router, error) {
	return r.other, r.err
}

func (r *nasIPRouterRepo) FindByID(context.Context, string, string) (*model.Router, error) {
	return r.router, nil
}

func (r *nasIPRouterRepo) Create(_ context.Context, router *model.Router) error {
	r.created = router
	r.router = router
	return nil
}

func (r *nasIPRouterRepo) Update(_ context.Context, router *model.Router) error {
	r.updated = router
	return nil
}

func TestNormalizeNASIP(t *testing.T) {
	lookupErr := errors.New("lookup failed")
	for _, tc := range []struct {
		name, raw, selfID, want string
		other                   *model.Router
		lookupErr, wantErr      error
	}{
		{name: "empty clears address", raw: "  "},
		{name: "trim IPv4", raw: " 203.0.113.7 ", want: "203.0.113.7"},
		{name: "same router", raw: "203.0.113.7", selfID: "self", other: &model.Router{ID: "self"}, want: "203.0.113.7"},
		{name: "different router", raw: "203.0.113.7", selfID: "self", other: &model.Router{ID: "other"}, wantErr: ErrNASIPInUse},
		{name: "lookup failure", raw: "203.0.113.7", lookupErr: lookupErr, wantErr: lookupErr},
		{name: "hostname", raw: "router.example.com", wantErr: ErrInvalidNASIP},
		{name: "CIDR", raw: "203.0.113.7/24", wantErr: ErrInvalidNASIP},
		{name: "IPv6", raw: "2001:db8::1", wantErr: ErrInvalidNASIP},
		{name: "mapped IPv6", raw: "::ffff:203.0.113.7", wantErr: ErrInvalidNASIP},
		{name: "unspecified", raw: "0.0.0.0", wantErr: ErrInvalidNASIP},
		{name: "loopback", raw: "127.0.0.1", wantErr: ErrInvalidNASIP},
		{name: "multicast", raw: "224.0.0.1", wantErr: ErrInvalidNASIP},
		{name: "broadcast", raw: "255.255.255.255", wantErr: ErrInvalidNASIP},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &nasIPRouterRepo{other: tc.other, err: tc.lookupErr}
			s := NewRouterService(repo, nil, nil)
			got, err := s.normalizeNASIP(context.Background(), tc.raw, tc.selfID)
			if got != tc.want || !errors.Is(err, tc.wantErr) {
				t.Fatalf("got (%q, %v), want (%q, %v)", got, err, tc.want, tc.wantErr)
			}
		})
	}
}

func TestCreateRouterPassesNASIPToRepository(t *testing.T) {
	repo := &nasIPRouterRepo{}
	s := NewRouterService(repo, nil, nil)
	router, err := s.Create(context.Background(), CreateRouterInput{
		TenantID: "tenant", Name: "router", RouterType: "cisco", NASIP: " 203.0.113.7 ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if repo.created == nil || repo.created.NASIP != "203.0.113.7" || router.NASIP != "203.0.113.7" {
		t.Fatalf("NAS-IP was not passed to persistence: %#v", repo.created)
	}
}

func TestUpdateRouterOptionalNASIP(t *testing.T) {
	empty, replacement, invalid := "", "203.0.113.8", "invalid"
	for _, tc := range []struct {
		name    string
		input   *string
		want    string
		wantErr error
	}{
		{name: "omitted preserves", want: "203.0.113.7"},
		{name: "empty clears", input: &empty},
		{name: "set replaces", input: &replacement, want: replacement},
		{name: "invalid rejected", input: &invalid, wantErr: ErrInvalidNASIP},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &nasIPRouterRepo{router: &model.Router{ID: "self", NASIP: "203.0.113.7"}}
			s := NewRouterService(repo, nil, nil)
			_, err := s.Update(context.Background(), "tenant", "self", UpdateRouterInput{IsActive: true, NASIP: tc.input})
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}
			if err != nil {
				if repo.updated != nil {
					t.Fatal("invalid address reached persistence")
				}
				return
			}
			if repo.updated == nil || repo.updated.NASIP != tc.want {
				t.Fatalf("unexpected persisted router: %#v", repo.updated)
			}
		})
	}
}
