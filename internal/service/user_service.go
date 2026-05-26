package service

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

var (
	ErrUserEmailExists  = errors.New("email sudah digunakan")
	ErrCannotDeleteSelf = errors.New("tidak dapat menghapus akun sendiri")
	ErrCannotEditSelf   = errors.New("tidak dapat mengubah role akun sendiri")
	ErrInvalidRole      = errors.New("role tidak valid")
)

type UserService struct {
	userRepo repository.UserRepository
	roleRepo repository.RoleRepository
}

func NewUserService(userRepo repository.UserRepository, roleRepo repository.RoleRepository) *UserService {
	return &UserService{userRepo: userRepo, roleRepo: roleRepo}
}

type UserListItem struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Email       string  `json:"email"`
	Role        string  `json:"role"`
	Phone       string  `json:"phone"`
	IsActive    bool    `json:"is_active"`
	LastLoginAt *string `json:"last_login_at,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

func (s *UserService) List(ctx context.Context, tenantID string) ([]UserListItem, error) {
	users, err := s.userRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	items := make([]UserListItem, 0, len(users))
	for _, u := range users {
		item := UserListItem{
			ID:        u.ID,
			Name:      u.Name,
			Email:     u.Email,
			Role:      u.Role,
			Phone:     u.Phone,
			IsActive:  u.IsActive,
			CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
		if u.LastLoginAt != nil {
			t := u.LastLoginAt.Format("2006-01-02T15:04:05Z")
			item.LastLoginAt = &t
		}
		items = append(items, item)
	}
	return items, nil
}

type CreateUserInput struct {
	TenantID string
	Name     string
	Email    string
	Password string
	Role     string
	Phone    string
}

func (s *UserService) Create(ctx context.Context, input CreateUserInput) (*UserListItem, error) {
	if !s.isValidRole(ctx, input.TenantID, input.Role) {
		return nil, ErrInvalidRole
	}

	existing, err := s.userRepo.FindByEmail(ctx, input.TenantID, input.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrUserEmailExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		TenantID:     input.TenantID,
		Name:         input.Name,
		Email:        input.Email,
		PasswordHash: string(hash),
		Role:         input.Role,
		Phone:        input.Phone,
		IsActive:     true,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	item := &UserListItem{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Role:      user.Role,
		Phone:     user.Phone,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	return item, nil
}

type UpdateUserInput struct {
	TenantID    string
	UserID      string
	CurrentUser string // ID of the user performing the edit
	Name        string
	Email       string
	Role        string
	Phone       string
	IsActive    *bool
	Password    string // optional — only set if non-empty
}

func (s *UserService) Update(ctx context.Context, input UpdateUserInput) (*UserListItem, error) {
	if !s.isValidRole(ctx, input.TenantID, input.Role) {
		return nil, ErrInvalidRole
	}

	user, err := s.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.TenantID != input.TenantID {
		return nil, ErrUserNotFound
	}

	// Prevent editing own role
	if input.UserID == input.CurrentUser && user.Role != input.Role {
		return nil, ErrCannotEditSelf
	}

	// Check email uniqueness if changed
	if input.Email != user.Email {
		existing, err := s.userRepo.FindByEmail(ctx, input.TenantID, input.Email)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return nil, ErrUserEmailExists
		}
	}

	user.Name = input.Name
	user.Email = input.Email
	user.Role = input.Role
	user.Phone = input.Phone

	if input.IsActive != nil {
		user.IsActive = *input.IsActive
	}

	if input.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		user.PasswordHash = string(hash)
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	item := &UserListItem{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Role:      user.Role,
		Phone:     user.Phone,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if user.LastLoginAt != nil {
		t := user.LastLoginAt.Format("2006-01-02T15:04:05Z")
		item.LastLoginAt = &t
	}
	return item, nil
}

func (s *UserService) Delete(ctx context.Context, tenantID, userID, currentUserID string) error {
	if userID == currentUserID {
		return ErrCannotDeleteSelf
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil || user.TenantID != tenantID {
		return ErrUserNotFound
	}

	return s.userRepo.Delete(ctx, tenantID, userID)
}

func (s *UserService) ToggleActive(ctx context.Context, tenantID, userID, currentUserID string) (*UserListItem, error) {
	if userID == currentUserID {
		return nil, ErrCannotEditSelf
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.TenantID != tenantID {
		return nil, ErrUserNotFound
	}

	user.IsActive = !user.IsActive
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	item := &UserListItem{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Role:      user.Role,
		Phone:     user.Phone,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if user.LastLoginAt != nil {
		t := user.LastLoginAt.Format("2006-01-02T15:04:05Z")
		item.LastLoginAt = &t
	}
	return item, nil
}

func (s *UserService) isValidRole(ctx context.Context, tenantID, slug string) bool {
	// Ensure default roles exist before checking
	count, err := s.roleRepo.CountByTenant(ctx, tenantID)
	if err != nil {
		return false
	}
	if count == 0 {
		return slug == "owner" || slug == "admin" || slug == "technician"
	}
	role, err := s.roleRepo.FindBySlug(ctx, tenantID, slug)
	return err == nil && role != nil
}
