package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	CookieName = "distry_session"
	SessionTTL = 30 * 24 * time.Hour
)

var (
	ErrTaken              = errors.New("username or email already taken")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUnauthenticated    = errors.New("unauthenticated")
	ErrValidation         = errors.New("validation failed")
)

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,32}$`)

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type UserRepo interface {
	Create(ctx context.Context, username, email, passwordHash string) (User, error)
	ByEmail(ctx context.Context, email string) (User, string, error)
}

type SessionRepo interface {
	Create(ctx context.Context, tokenHash []byte, userID string, expires time.Time) error
	UserByTokenHash(ctx context.Context, tokenHash []byte) (User, error)
	Delete(ctx context.Context, tokenHash []byte) error
}

type Clock interface {
	Now() time.Time
}

type TokenSource interface {
	Token() (string, error)
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now()
}

type randomTokenSource struct{}

func (randomTokenSource) Token() (string, error) {
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(token), nil
}

type Service struct {
	users    UserRepo
	sessions SessionRepo
	clock    Clock
	tokens   TokenSource
}

func NewService(users UserRepo, sessions SessionRepo) *Service {
	return NewServiceWithDeps(users, sessions, realClock{}, randomTokenSource{})
}

func NewServiceWithDeps(users UserRepo, sessions SessionRepo, clock Clock, tokens TokenSource) *Service {
	return &Service{users: users, sessions: sessions, clock: clock, tokens: tokens}
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(hash), err
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func (s *Service) SignUp(ctx context.Context, username, email, password string) (User, string, error) {
	username = strings.TrimSpace(username)
	email, err := validateSignUp(username, email, password)
	if err != nil {
		return User{}, "", err
	}

	passwordHash, err := HashPassword(password)
	if err != nil {
		return User{}, "", err
	}
	user, err := s.users.Create(ctx, username, email, passwordHash)
	if err != nil {
		return User{}, "", err
	}

	token, err := s.createSession(ctx, user.ID)
	if err != nil {
		return User{}, "", err
	}
	return user, token, nil
}

func (s *Service) LogIn(ctx context.Context, email, password string) (User, string, error) {
	email, err := normalizeEmail(email)
	if err != nil {
		return User{}, "", ErrInvalidCredentials
	}

	user, passwordHash, err := s.users.ByEmail(ctx, email)
	if err != nil {
		return User{}, "", ErrInvalidCredentials
	}
	if !CheckPassword(passwordHash, password) {
		return User{}, "", ErrInvalidCredentials
	}

	token, err := s.createSession(ctx, user.ID)
	if err != nil {
		return User{}, "", err
	}
	return user, token, nil
}

func (s *Service) Authenticate(ctx context.Context, token string) (User, error) {
	if token == "" {
		return User{}, ErrUnauthenticated
	}
	user, err := s.sessions.UserByTokenHash(ctx, tokenHash(token))
	if err != nil {
		return User{}, ErrUnauthenticated
	}
	return user, nil
}

func (s *Service) LogOut(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.sessions.Delete(ctx, tokenHash(token))
}

func (s *Service) createSession(ctx context.Context, userID string) (string, error) {
	token, err := s.tokens.Token()
	if err != nil {
		return "", err
	}
	if err := s.sessions.Create(ctx, tokenHash(token), userID, s.clock.Now().Add(SessionTTL)); err != nil {
		return "", err
	}
	return token, nil
}

func validateSignUp(username, email, password string) (string, error) {
	if !usernamePattern.MatchString(username) || len(password) < 8 {
		return "", ErrValidation
	}
	return normalizeEmail(email)
}

func normalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || !strings.Contains(email, ".") {
		return "", ErrValidation
	}
	return email, nil
}

func tokenHash(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
