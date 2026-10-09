package auth

import (
	"context"
	"errors"
	"time"

	"github.com/sanyam-malik/pulsewatch/internal/utils"

	"github.com/pquerna/otp/totp"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type Service interface {
	Register(ctx context.Context, dto RegisterDto) (*LoginResponse, error)
	Login(ctx context.Context, dto LoginDto) (*LoginResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error)
	UpdatePassword(ctx context.Context, userId string, dto UpdatePasswordDto) error
	ListGroups(ctx context.Context, userID string) ([]Group, error)
	CreateGroup(ctx context.Context, userID string, dto CreateGroupDto) (*Group, error)
	ListMembers(ctx context.Context, actorID, groupID string) ([]GroupMember, error)
	AddMember(ctx context.Context, actorID, groupID string, dto AddMemberDto) (*GroupMember, error)
	UpdateMemberRole(ctx context.Context, actorID, groupID, userID string, dto UpdateMemberRoleDto) error
	RemoveMember(ctx context.Context, actorID, groupID, userID string) error

	// 2FA methods
	SetupTwoFA(ctx context.Context, userId, password string) (secret string, provisioningURI string, err error)
	VerifyTwoFA(ctx context.Context, userId, code string) (bool, error)
	DisableTwoFA(ctx context.Context, userId, password string) error
}

type ServiceImpl struct {
	repo       Repository
	tokenMaker *TokenMaker
	logger     *zap.SugaredLogger
}

func NewService(
	repo Repository,
	tokenMaker *TokenMaker,
	logger *zap.SugaredLogger,
) Service {
	return &ServiceImpl{
		repo:       repo,
		tokenMaker: tokenMaker,
		logger:     logger.Named("[auth-service]"),
	}
}

func (s *ServiceImpl) Register(ctx context.Context, dto RegisterDto) (*LoginResponse, error) {
	count, err := s.repo.FindAllCount(ctx)
	if err != nil {
		return nil, err
	}

	if count > 0 {
		return nil, errors.New("admin already exists")
	}
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(dto.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Create new admin
	user := &Model{
		Email:     dto.Email,
		Password:  string(hashedPassword),
		Active:    true,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	// Save to database
	user, group, err := s.repo.CreateFirstOwner(ctx, user, "Default group")
	if err != nil {
		return nil, err
	}
	membership := &Membership{GroupID: group.ID, UserID: user.ID, Role: RoleOwner}
	user.GroupID, user.Role = membership.GroupID, membership.Role

	// Generate access token
	accessToken, err := s.tokenMaker.CreateAccessToken(ctx, user, membership)
	if err != nil {
		return nil, err
	}

	// Generate refresh token
	refreshToken, err := s.tokenMaker.CreateRefreshToken(ctx, user, membership)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

func (s *ServiceImpl) Login(ctx context.Context, dto LoginDto) (*LoginResponse, error) {
	// Find admin by email
	user, err := s.repo.FindByEmail(ctx, dto.Email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}
	if user == nil {
		return nil, errors.New("invalid credentials")
	}
	if !user.Active {
		return nil, errors.New("account is disabled")
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(dto.Password))
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Enforce 2FA if enabled
	if user.TwoFASecret != "" && user.TwoFAStatus {
		if dto.Token == "" {
			return nil, errors.New("2FA token required")
		}
		if !totp.Validate(dto.Token, user.TwoFASecret) {
			return nil, errors.New("invalid 2FA token")
		}
	}

	// Generate access token
	membership, err := s.repo.ResolveMembership(ctx, user.ID, "")
	if err != nil {
		return nil, err
	}
	if membership == nil {
		return nil, errors.New("account is not a member of a group")
	}
	user.GroupID, user.Role = membership.GroupID, membership.Role
	accessToken, err := s.tokenMaker.CreateAccessToken(ctx, user, membership)
	if err != nil {
		return nil, err
	}

	// Generate refresh token
	refreshToken, err := s.tokenMaker.CreateRefreshToken(ctx, user, membership)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

func (s *ServiceImpl) RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error) {
	// Verify refresh token
	claims, err := s.tokenMaker.VerifyToken(ctx, refreshToken, "refresh")
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	// Check if it's a refresh token
	if claims.Type != "refresh" {
		return nil, errors.New("invalid token type")
	}

	// Find admin by ID
	user, err := s.repo.FindByID(ctx, claims.UserID)
	if err != nil || user == nil {
		return nil, errors.New("user not found")
	}
	if !user.Active {
		return nil, errors.New("account is disabled")
	}
	membership, err := s.repo.ResolveMembership(ctx, user.ID, claims.GroupID)
	if err != nil {
		return nil, err
	}
	if membership == nil {
		membership, err = s.repo.ResolveMembership(ctx, user.ID, "")
		if err != nil || membership == nil {
			return nil, errors.New("group membership not found")
		}
	}
	user.GroupID, user.Role = membership.GroupID, membership.Role

	// Generate new access token
	accessToken, err := s.tokenMaker.CreateAccessToken(ctx, user, membership)
	if err != nil {
		return nil, err
	}

	// Generate new refresh token
	newRefreshToken, err := s.tokenMaker.CreateRefreshToken(ctx, user, membership)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		User:         user,
		RefreshToken: newRefreshToken,
		AccessToken:  accessToken,
	}, nil
}

func (s *ServiceImpl) ListGroups(ctx context.Context, userID string) ([]Group, error) {
	return s.repo.ListGroups(ctx, userID)
}

func (s *ServiceImpl) CreateGroup(ctx context.Context, userID string, dto CreateGroupDto) (*Group, error) {
	return s.repo.CreateGroup(ctx, userID, dto.Name)
}

func (s *ServiceImpl) requireManager(ctx context.Context, actorID, groupID string) error {
	membership, err := s.repo.ResolveMembership(ctx, actorID, groupID)
	if err != nil {
		return err
	}
	if membership == nil || (membership.Role != RoleOwner && membership.Role != RoleAdmin) {
		return errors.New("group administrator access required")
	}
	return nil
}

func (s *ServiceImpl) ListMembers(ctx context.Context, actorID, groupID string) ([]GroupMember, error) {
	if err := s.requireManager(ctx, actorID, groupID); err != nil {
		return nil, err
	}
	return s.repo.ListMembers(ctx, groupID)
}

func (s *ServiceImpl) AddMember(ctx context.Context, actorID, groupID string, dto AddMemberDto) (*GroupMember, error) {
	actor, err := s.repo.ResolveMembership(ctx, actorID, groupID)
	if err != nil {
		return nil, err
	}
	if actor == nil || (actor.Role != RoleOwner && actor.Role != RoleAdmin) {
		return nil, errors.New("group administrator access required")
	}
	if dto.Role == RoleAdmin && actor.Role != RoleOwner {
		return nil, errors.New("only a group owner can grant administrator access")
	}
	user, err := s.repo.FindByEmail(ctx, dto.Email)
	if err != nil {
		return nil, err
	}
	password := ""
	if user == nil {
		if err := utils.Validate.Var(dto.Password, "required,password"); err != nil {
			return nil, errors.New("password must be at least 8 characters long and contain uppercase, lowercase, number, and special character")
		}
		hashedPassword, hashErr := bcrypt.GenerateFromPassword([]byte(dto.Password), bcrypt.DefaultCost)
		if hashErr != nil {
			return nil, hashErr
		}
		password = string(hashedPassword)
	}
	return s.repo.AddMember(ctx, groupID, dto.Email, password, dto.Role)
}

func (s *ServiceImpl) UpdateMemberRole(ctx context.Context, actorID, groupID, userID string, dto UpdateMemberRoleDto) error {
	actor, err := s.repo.ResolveMembership(ctx, actorID, groupID)
	if err != nil {
		return err
	}
	if actor == nil || (actor.Role != RoleOwner && actor.Role != RoleAdmin) {
		return errors.New("group administrator access required")
	}
	target, err := s.repo.ResolveMembership(ctx, userID, groupID)
	if err != nil {
		return err
	}
	if target == nil {
		return errors.New("group membership not found")
	}
	if actor.Role != RoleOwner && target.Role == RoleOwner {
		return errors.New("only a group owner can change another owner's access")
	}
	if actorID == userID && target.Role == RoleOwner {
		return errors.New("an owner cannot change their own role")
	}
	if dto.Role == RoleOwner {
		return errors.New("owner role cannot be assigned directly")
	}
	if actor.Role != RoleOwner && dto.Role == RoleAdmin {
		return errors.New("only a group owner can grant administrator access")
	}
	if !validRole(dto.Role) {
		return errors.New("invalid role")
	}
	if err := s.requireManager(ctx, actorID, groupID); err != nil {
		return err
	}
	return s.repo.UpdateMemberRole(ctx, groupID, userID, dto.Role)
}

func (s *ServiceImpl) RemoveMember(ctx context.Context, actorID, groupID, userID string) error {
	actor, err := s.repo.ResolveMembership(ctx, actorID, groupID)
	if err != nil {
		return err
	}
	target, err := s.repo.ResolveMembership(ctx, userID, groupID)
	if err != nil {
		return err
	}
	if actor == nil || (actor.Role != RoleOwner && actor.Role != RoleAdmin) {
		return errors.New("group administrator access required")
	}
	if target == nil {
		return errors.New("group membership not found")
	}
	if actor.Role != RoleOwner && target.Role == RoleOwner {
		return errors.New("only a group owner can remove another owner")
	}
	if actorID == userID && target.Role == RoleOwner {
		return errors.New("an owner cannot remove their own membership")
	}
	if err := s.requireManager(ctx, actorID, groupID); err != nil {
		return err
	}
	return s.repo.RemoveMember(ctx, groupID, userID)
}

func (s *ServiceImpl) UpdatePassword(ctx context.Context, userId string, dto UpdatePasswordDto) error {
	// Find user by ID
	user, err := s.repo.FindByID(ctx, userId)
	if err != nil || user == nil {
		return errors.New("user not found")
	}

	// Verify current password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(dto.CurrentPassword))
	if err != nil {
		return errors.New("current password is incorrect")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(dto.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("failed to hash new password")
	}

	password := string(hashedPassword)
	updateModel := &UpdateModel{
		Password: &password,
	}

	// Update user in DB
	err = s.repo.Update(ctx, userId, updateModel)
	if err != nil {
		return errors.New("failed to update password")
	}

	return nil
}

func (s *ServiceImpl) SetupTwoFA(ctx context.Context, userId, password string) (string, string, error) {
	user, err := s.repo.FindByID(ctx, userId)
	if err != nil || user == nil {
		return "", "", errors.New("user not found")
	}

	// Require password verification
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", "", errors.New("invalid password")
	}
	var secretStr, provisioningURI string
	if user.TwoFASecret == "" {
		// Generate new secret and provisioning URI
		key, err := totp.Generate(totp.GenerateOpts{
			Issuer:      "Pulsewatch",
			AccountName: user.Email,
		})
		if err != nil {
			return "", "", err
		}
		secretStr = key.Secret()
		provisioningURI = key.URL()
		updateModel := &UpdateModel{
			TwoFASecret: &secretStr,
		}
		err = s.repo.Update(ctx, userId, updateModel)
		if err != nil {
			return "", "", err
		}
	} else {
		// If already set, just return existing
		secretStr = user.TwoFASecret
		// Recreate the provisioning URI
		key, err := totp.Generate(totp.GenerateOpts{
			Issuer:      "Pulsewatch",
			AccountName: user.Email,
			Secret:      []byte(user.TwoFASecret),
		})
		if err != nil {
			return "", "", err
		}
		provisioningURI = key.URL()
	}
	return secretStr, provisioningURI, nil
}

func (s *ServiceImpl) VerifyTwoFA(ctx context.Context, userId, code string) (bool, error) {
	user, err := s.repo.FindByID(ctx, userId)
	if err != nil || user == nil {
		return false, errors.New("user not found")
	}
	if user.TwoFASecret == "" {
		return false, errors.New("2FA not setup")
	}
	valid := totp.Validate(code, user.TwoFASecret)
	if valid {
		status := true
		updateModel := &UpdateModel{
			TwoFAStatus: &status,
		}
		s.repo.Update(ctx, userId, updateModel)
	}
	return valid, nil
}

func (s *ServiceImpl) DisableTwoFA(ctx context.Context, userId, password string) error {
	user, err := s.repo.FindByID(ctx, userId)
	if err != nil || user == nil {
		return errors.New("user not found")
	}
	// Require password verification
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return errors.New("invalid password")
	}

	// Prepare values for pointers
	secret := ""
	status := false

	updateModel := &UpdateModel{
		TwoFASecret: &secret,
		TwoFAStatus: &status,
	}

	err = s.repo.Update(ctx, userId, updateModel)
	if err != nil {
		return errors.New("failed to disable 2FA")
	}

	return nil
}
