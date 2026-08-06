package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"uptime-backend/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrInvalidRegistrationInput lets callers reject registration data without field-level leakage.
	ErrInvalidRegistrationInput = errors.New("invalid registration input")
	// ErrInvalidProfileInput identifies profile data that cannot be stored.
	ErrInvalidProfileInput = errors.New("invalid profile input")
)

// RegistrationInput groups account fields so registration validation has one explicit boundary.
type RegistrationInput struct {
	Email    string
	Name     string
	Password string
}

// TokenPair carries access and refresh credentials together with the user context returned to the client.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
	User         models.User
}

// Service centralizes token, credential, and account rules while stores handle persistence.
type Service struct {
	users      UserStore
	sessions   SessionStore
	jwtSecret  []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
	now        func() time.Time
}

// NewService constructs a service with injectable stores and signing configuration.
func NewService(users UserStore, sessions SessionStore, jwtSecret string, accessTTL, refreshTTL time.Duration) *Service {
	return &Service{users: users, sessions: sessions, jwtSecret: []byte(jwtSecret), accessTTL: accessTTL, refreshTTL: refreshTTL, now: time.Now}
}

// Register validates and persists an account before issuing its first token pair.
func (service *Service) Register(ctx context.Context, input RegistrationInput) (TokenPair, error) {
	email, name, err := validateRegistration(input)
	if err != nil {
		return TokenPair{}, err
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return TokenPair{}, fmt.Errorf("hash password: %w", err)
	}
	user := models.User{ID: uuid.New(), Email: email, Name: name, PasswordHash: string(passwordHash)}
	if err := service.users.Create(ctx, user); err != nil {
		return TokenPair{}, err
	}
	return service.issue(ctx, user)
}

// Login normalizes the lookup key and uses one public error for unknown users and bad passwords.
func (service *Service) Login(ctx context.Context, email, password string) (TokenPair, error) {
	user, err := service.users.FindByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			return TokenPair{}, ErrInvalidCredentials
		}
		return TokenPair{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return TokenPair{}, ErrInvalidCredentials
	}
	return service.issue(ctx, user)
}

// Profile loads the user identified by a previously validated access token.
func (service *Service) Profile(ctx context.Context, userID uuid.UUID) (models.User, error) {
	return service.users.FindByID(ctx, userID)
}

// UserIDFromRequest accepts only access JWTs signed with the configured algorithm and secret.
func (service *Service) UserIDFromRequest(request *http.Request) (uuid.UUID, error) {
	const prefix = "Bearer "
	header := request.Header.Get("Authorization")
	if !strings.HasPrefix(header, prefix) {
		return uuid.Nil, ErrInvalidCredentials
	}
	token, err := jwt.Parse(strings.TrimSpace(strings.TrimPrefix(header, prefix)), func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return service.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return uuid.Nil, ErrInvalidCredentials
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["token_type"] != "access" {
		return uuid.Nil, ErrInvalidCredentials
	}
	sub, ok := claims["sub"].(string)
	userID, err := uuid.Parse(sub)
	if !ok || err != nil || userID == uuid.Nil {
		return uuid.Nil, ErrInvalidCredentials
	}
	return userID, nil
}

// UpdateProfile trims and bounds the display name before delegating persistence.
func (service *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, name string) (models.User, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 100 {
		return models.User{}, ErrInvalidProfileInput
	}
	return service.users.UpdateName(ctx, userID, name)
}

// UpdateAvatar delegates the already validated stored filename to persistence.
func (service *Service) UpdateAvatar(ctx context.Context, userID uuid.UUID, avatarPath string) (models.User, error) {
	return service.users.UpdateAvatar(ctx, userID, avatarPath)
}

// Refresh rotates the refresh session before issuing a replacement pair, making reuse detectable.
func (service *Service) Refresh(ctx context.Context, refreshToken string) (TokenPair, error) {
	if refreshToken == "" {
		return TokenPair{}, ErrInvalidRefreshToken
	}
	now := service.now().UTC()
	newToken, newHash, err := newRefreshToken()
	if err != nil {
		return TokenPair{}, err
	}
	replacement := NewSession(uuid.Nil, newHash, now.Add(service.refreshTTL))
	user, err := service.sessions.Rotate(ctx, hashRefreshToken(refreshToken), replacement, now)
	if err != nil {
		return TokenPair{}, err
	}
	return service.issueWithRefresh(ctx, user, newToken, replacement)
}

// Logout revokes the refresh session and treats an absent token as an already completed logout.
func (service *Service) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	return service.sessions.Revoke(ctx, hashRefreshToken(refreshToken), service.now().UTC())
}

func (service *Service) issue(ctx context.Context, user models.User) (TokenPair, error) {
	refreshToken, refreshHash, err := newRefreshToken()
	if err != nil {
		return TokenPair{}, err
	}
	session := NewSession(user.ID, refreshHash, service.now().UTC().Add(service.refreshTTL))
	if err := service.sessions.Create(ctx, session); err != nil {
		return TokenPair{}, err
	}
	return service.issueWithRefresh(ctx, user, refreshToken, session)
}

func (service *Service) issueWithRefresh(_ context.Context, user models.User, refreshToken string, _ models.RefreshSession) (TokenPair, error) {
	now := service.now().UTC()
	expiresAt := now.Add(service.accessTTL)
	claims := jwt.MapClaims{"sub": user.ID.String(), "token_type": "access", "iat": now.Unix(), "exp": expiresAt.Unix()}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(service.jwtSecret)
	if err != nil {
		return TokenPair{}, fmt.Errorf("sign access token: %w", err)
	}
	return TokenPair{AccessToken: signed, RefreshToken: refreshToken, ExpiresIn: int64(service.accessTTL.Seconds()), User: user}, nil
}

func validateRegistration(input RegistrationInput) (string, string, error) {
	email := normalizeEmail(input.Email)
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email || len(email) > 254 {
		return "", "", ErrInvalidRegistrationInput
	}
	name := strings.TrimSpace(input.Name)
	if name == "" || len(name) > 100 || len(input.Password) < 8 || len(input.Password) > 72 {
		return "", "", ErrInvalidRegistrationInput
	}
	return email, name, nil
}

func normalizeEmail(value string) string { return strings.ToLower(strings.TrimSpace(value)) }

func newRefreshToken() (string, string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(bytes)
	return token, hashRefreshToken(token), nil
}

func hashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
