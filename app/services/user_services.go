package services

import (
	"DataTracker/app/models"
	"DataTracker/app/repositories"
	"DataTracker/app/utils"
	"context"
	"errors"
	"fmt"

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

// --- CRUD Operations ---

func (s *UserService) Create(user models.User) error {
	// 1. Check Uniqueness (Optimized via Bloom Filter)
	if taken, _ := s.IsUsernameTaken(user.Username); taken {
		return fmt.Errorf("username already exists")
	}
	if taken, _ := s.IsEmailTaken(user.Email); taken {
		return fmt.Errorf("email already exists")
	}

	// 2. Hash Password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashedPassword)

	// 3. Persist
	err = s.Repo.Create(context.Background(), user)
	if err == nil {
		// 4. Update Bloom Filters upon success
		s.UsernameFilter.Insert([]byte(user.Username))
		s.EmailFilter.Insert([]byte(user.Email))
	}
	// 5. we auto set role from new users to "user" to avoid them giving themselves privileges
	user.Role = "user"
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
		// Hash the NEW password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
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

	// 1. Handle Username Change (Validation + Bloom Filter)
	if newUsername, ok := updates["username"].(string); ok && newUsername != "" {
		// Only check if it's actually different (optional but recommended)
		taken, err := s.IsUsernameTaken(newUsername)
		if err != nil {
			return err
		}
		if taken {
			return errors.New("new username already exists")
		}
		// Insert into Bloom Filter
		s.UsernameFilter.Insert([]byte(newUsername))
	}

	// 2. Handle Password Change (Hashing)
	if rawPassword, ok := updates["password"].(string); ok && rawPassword != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(rawPassword), bcrypt.DefaultCost)
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
