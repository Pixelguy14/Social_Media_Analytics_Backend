package services

import (
	"DataTracker/app/models"
	"DataTracker/app/repositories"
	"DataTracker/app/utils"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	Repo           repositories.UserRepository
	UsernameFilter *utils.ConcurrentBloomFilter
	EmailFilter    *utils.ConcurrentBloomFilter
}

func NewUserService(repo repositories.UserRepository, uFilter *utils.ConcurrentBloomFilter, eFilter *utils.ConcurrentBloomFilter) *UserService {
	return &UserService{
		Repo:           repo,
		UsernameFilter: uFilter,
		EmailFilter:    eFilter,
	}
}

// Checkers (Business Logic)
func (s *UserService) IsUsernameTaken(username string) (bool, error) {
	if !s.UsernameFilter.Contains([]byte(username)) {
		return false, nil
	}
	return s.Repo.ExistsByUsername(context.Background(), username)
}

func (s *UserService) IsEmailTaken(email string) (bool, error) {
	if !s.EmailFilter.Contains([]byte(email)) {
		return false, nil
	}
	return s.Repo.ExistsByEmail(context.Background(), email)
}

// IsPasswordSecure handles unicode characters for international support (Ñ, Á, Ç, etc.)
func IsPasswordSecure(password string) bool {
	if len(password) < 8 {
		return false
	}

	var hasUpper, hasNumber bool
	for _, r := range password {
		if unicode.IsUpper(r) { // This catches Ñ, Á, Ç, etc.
			hasUpper = true
		}
		if unicode.IsDigit(r) {
			hasNumber = true
		}
		if hasUpper && hasNumber {
			return true
		}
	}
	return false
}

// --- CRUD Operations ---

func (s *UserService) Create(user models.User) error {
	// 1. Check Uniqueness (Optimized via Bloom Filter)
	if taken, _ := s.IsUsernameTaken(user.Username); taken {
		return fmt.Errorf("username already exists")
	}
	if taken, _ := s.IsEmailTaken(user.Email); taken {
		return fmt.Errorf("email already exists")
	}

	// 1.5 Validate Password Complexity
	if !IsPasswordSecure(user.Password) {
		return fmt.Errorf("password insecure: must be 8+ chars with at least 1 number and 1 uppercase letter")
	}

	// 2. Hash Password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), 12)
	if err != nil {
		return err
	}
	user.Password = string(hashedPassword)

	// 3. we auto set role from new users to "user" to avoid them giving themselves privileges
	user.Role = "user"

	// 4. Persist
	err = s.Repo.Create(context.Background(), user)
	if err == nil {
		// 5. Update Bloom Filters upon success
		s.UsernameFilter.Insert([]byte(user.Username))
		s.EmailFilter.Insert([]byte(user.Email))
	}
	return err
}

// Login logic that user auth middleware
func (s *UserService) Login(email, password string) (*models.User, error) {
	// 1. Fetch user by email
	user, err := s.Repo.GetByEmail(context.Background(), email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	// 2. Compare hashed password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	return user, nil
}

func (s *UserService) UpdateUser(id string, user models.User) error {
	existingUser, err := s.Repo.GetByID(context.Background(), id)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// 1. Logic for Username Change
	if user.Username != existingUser.Username {
		taken, err := s.IsUsernameTaken(user.Username)
		if err != nil {
			return err // Database error
		}
		if taken {
			return errors.New("new username already exists")
		}
	}

	// 2. Logic for Password Change
	if user.Password != "" {
		// Validate Complexity before hashing
		if !IsPasswordSecure(user.Password) {
			return errors.New("new password insecure: must be 8+ chars with 1 number and 1 uppercase")
		}
		// Hash the NEW password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), 12)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}
		user.Password = string(hashedPassword)
	} else {
		// KEEP the OLD password if the field was empty in the request
		user.Password = existingUser.Password
	}

	// 3. Update Database
	if err := s.Repo.Update(context.Background(), id, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// 4. Sync Bloom Filter
	s.UsernameFilter.Insert([]byte(user.Username))
	return nil
}

func (s *UserService) UpdateUserFields(id string, updates map[string]interface{}) error {
	ctx := context.Background()

	// 1. Fetch current user to compare values
	currentUser, err := s.Repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}
	// 2. Handle Username Change
	if newUsername, ok := updates["username"].(string); ok && newUsername != "" {
		// Only check if the username is DIFFERENT from the current one
		if newUsername != currentUser.Username {
			taken, err := s.IsUsernameTaken(newUsername)
			if err != nil {
				return err
			}
			if taken {
				// Double check it's not a false positive or some other issue
				return errors.New("new username already exists")
			}
			// Insert into Bloom Filter
			s.UsernameFilter.Insert([]byte(newUsername))
		} else {
			// key is present but value is the same, remove it from updates to save DB write
			delete(updates, "username")
		}
	}

	// 2. Handle Password Change (Hashing)
	if rawPassword, ok := updates["password"].(string); ok && rawPassword != "" {
		// Validate Complexity
		if !IsPasswordSecure(rawPassword) {
			return errors.New("new password insecure: must be 8+ chars with 1 number and 1 uppercase")
		}
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(rawPassword), 12)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}
		updates["password"] = string(hashedPassword)
	} else {
		// Ensure we don't accidentally set a password to empty string if it was sent as ""
		delete(updates, "password")
	}

	// 3. Update Database via a specialized Repo method
	if err := s.Repo.UpdateFields(ctx, id, updates); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

func (s *UserService) GetUserByID(id string) (*models.User, error) {
	// 1. Validation: Ensure ID isn't empty
	if id == "" {
		return nil, errors.New("user ID is required")
	}

	// 2. Repository Call
	user, err := s.Repo.GetByID(context.Background(), id)
	if err != nil {
		// Wrap the error so the controller knows it's a 404 vs a 500
		return nil, fmt.Errorf("could not retrieve user: %w", err)
	}

	return user, nil
}

// Retrieve all available users
func (s *UserService) GetAllUsers(ctx context.Context) ([]models.User, error) {
	return s.Repo.GetAllUsers(ctx)
}

func (s *UserService) DeleteUser(id string) error {
	// 1. Check if user exists first
	// This prevents "Success" messages for deleting non-existent users
	_, err := s.Repo.GetByID(context.Background(), id)
	if err != nil {
		return fmt.Errorf("cannot delete: user not found: %w", err)
	}

	// 2. Perform deletion
	err = s.Repo.Delete(context.Background(), id)
	if err != nil {
		return fmt.Errorf("failed to delete user from database: %w", err)
	}

	// We don't remove from Bloom Filter because standard Bloom Filters don't support deletion.
	return nil
}

// --- Security Operations: Password Resets ---

func (s *UserService) InitiatePasswordReset(ctx context.Context, userID string) (string, error) {
	// 1. Generate Raw Token (32 bytes high entropy)
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("cryptographic failure: %v", err)
	}
	rawToken := hex.EncodeToString(raw)

	// 2. Hash Token (SHA-256) for DB storage
	hash := sha256.Sum256([]byte(rawToken))
	hashedToken := hex.EncodeToString(hash[:])

	// 3. Persist to DB
	token := models.PasswordResetToken{
		UserID:      userID,
		HashedToken: hashedToken,
		ExpiresAt:   time.Now().Add(1 * time.Hour),
	}

	if err := s.Repo.SaveResetToken(ctx, token); err != nil {
		return "", err
	}

	return rawToken, nil
}

func (s *UserService) CompletePasswordReset(ctx context.Context, rawToken string, newPassword string) error {
	// 1. Hash incoming rawToken to find it
	hash := sha256.Sum256([]byte(rawToken))
	hashedToken := hex.EncodeToString(hash[:])

	// 2. Retrieve Token from DB
	tokenRecord, err := s.Repo.GetResetTokenByHash(ctx, hashedToken)
	if err != nil {
		return errors.New("invalid or expired password reset link")
	}

	// 3. ONE-TIME USE: Delete immediately regardless of outcome
	defer s.Repo.DeleteResetToken(ctx, hashedToken)

	// 4. Validate Expiration
	if time.Now().After(tokenRecord.ExpiresAt) {
		return errors.New("reset link has expired")
	}

	// 5. Validate New Password Complexity
	if !IsPasswordSecure(newPassword) {
		return errors.New("new password does not meet complexity requirements")
	}

	// 6. Hash and Update User Record
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return err
	}

	updates := map[string]interface{}{
		"password": string(hashedPassword),
	}

	return s.Repo.UpdateFields(ctx, tokenRecord.UserID, updates)
}
