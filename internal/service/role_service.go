package service

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

var (
	ErrRoleNotFound           = errors.New("role tidak ditemukan")
	ErrCannotDeleteSystemRole = errors.New("tidak dapat menghapus role bawaan")
	ErrCannotEditOwnerRole    = errors.New("tidak dapat mengubah permission role owner")
	ErrRoleSlugExists         = errors.New("slug role sudah digunakan")
	ErrRoleInUse              = errors.New("role masih digunakan oleh pengguna")
	ErrInvalidSlug            = errors.New("slug hanya boleh huruf kecil, angka, dan strip")
	ErrCannotGrantUnheld      = errors.New("tidak dapat memberikan permission yang tidak Anda miliki")
)

// actorMayGrant ensures a non-owner cannot grant permissions they do not
// themselves hold — otherwise anyone with roles.edit could self-escalate to
// full control. Owner and superadmin may grant anything.
func actorMayGrant(actorRole string, actorPerms, requested []string) error {
	if actorRole == "owner" || actorRole == "superadmin" {
		return nil
	}
	held := make(map[string]struct{}, len(actorPerms))
	for _, p := range actorPerms {
		held[p] = struct{}{}
	}
	for _, p := range requested {
		if _, ok := held[p]; !ok {
			return ErrCannotGrantUnheld
		}
	}
	return nil
}

var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$`)

type RoleService struct {
	roleRepo repository.RoleRepository
	userRepo repository.UserRepository
}

func NewRoleService(roleRepo repository.RoleRepository, userRepo repository.UserRepository) *RoleService {
	return &RoleService{roleRepo: roleRepo, userRepo: userRepo}
}

func (s *RoleService) List(ctx context.Context, tenantID string) ([]*model.Role, error) {
	if err := s.EnsureDefaults(ctx, tenantID); err != nil {
		return nil, err
	}
	return s.roleRepo.List(ctx, tenantID)
}

type CreateRoleInput struct {
	TenantID    string
	Name        string
	Slug        string
	Description string
	Permissions []string
	ActorRole        string   // role slug of the user performing this action
	ActorPermissions []string // permissions the actor currently holds
}

func (s *RoleService) Create(ctx context.Context, input CreateRoleInput) (*model.Role, error) {
	input.Slug = strings.ToLower(strings.TrimSpace(input.Slug))
	if !slugPattern.MatchString(input.Slug) {
		return nil, ErrInvalidSlug
	}

	existing, err := s.roleRepo.FindBySlug(ctx, input.TenantID, input.Slug)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrRoleSlugExists
	}

	perms := filterValidPermissions(input.Permissions)
	if err := actorMayGrant(input.ActorRole, input.ActorPermissions, perms); err != nil {
		return nil, err
	}

	role := &model.Role{
		TenantID:    input.TenantID,
		Name:        input.Name,
		Slug:        input.Slug,
		Description: input.Description,
		IsSystem:    false,
		Permissions: perms,
	}

	if err := s.roleRepo.Create(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

type UpdateRoleInput struct {
	TenantID    string
	RoleID      string
	Name        string
	Description string
	Permissions []string
	ActorRole        string   // role slug of the user performing this action
	ActorPermissions []string // permissions the actor currently holds
}

func (s *RoleService) Update(ctx context.Context, input UpdateRoleInput) (*model.Role, error) {
	role, err := s.roleRepo.FindByID(ctx, input.RoleID)
	if err != nil {
		return nil, err
	}
	if role == nil || role.TenantID != input.TenantID {
		return nil, ErrRoleNotFound
	}

	// Owner permissions cannot be changed
	if role.Slug == "owner" {
		return nil, ErrCannotEditOwnerRole
	}

	perms := filterValidPermissions(input.Permissions)
	if err := actorMayGrant(input.ActorRole, input.ActorPermissions, perms); err != nil {
		return nil, err
	}

	role.Name = input.Name
	role.Description = input.Description
	role.Permissions = perms

	if err := s.roleRepo.Update(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *RoleService) Delete(ctx context.Context, tenantID, roleID string) error {
	role, err := s.roleRepo.FindByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role == nil || role.TenantID != tenantID {
		return ErrRoleNotFound
	}
	if role.IsSystem {
		return ErrCannotDeleteSystemRole
	}

	count, err := s.userRepo.CountByRole(ctx, tenantID, role.Slug)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrRoleInUse
	}

	return s.roleRepo.Delete(ctx, tenantID, roleID)
}

func (s *RoleService) EnsureDefaults(ctx context.Context, tenantID string) error {
	count, err := s.roleRepo.CountByTenant(ctx, tenantID)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	defaults := []model.Role{
		{TenantID: tenantID, Name: "Owner", Slug: "owner", Description: "Pemilik tenant dengan akses penuh", IsSystem: true, Permissions: model.AllPermissionKeys()},
		{TenantID: tenantID, Name: "Admin", Slug: "admin", Description: "Administrator dengan akses luas", IsSystem: true, Permissions: model.DefaultAdminPermissions()},
		{TenantID: tenantID, Name: "Teknisi", Slug: "technician", Description: "Teknisi lapangan", IsSystem: true, Permissions: model.DefaultTechnicianPermissions()},
	}

	for i := range defaults {
		if err := s.roleRepo.Create(ctx, &defaults[i]); err != nil {
			return err
		}
	}
	return nil
}

func filterValidPermissions(input []string) []string {
	valid := make(map[string]bool, 80)
	for _, k := range model.AllPermissionKeys() {
		valid[k] = true
	}
	result := []string{}
	for _, p := range input {
		if valid[p] {
			result = append(result, p)
		}
	}
	return result
}
