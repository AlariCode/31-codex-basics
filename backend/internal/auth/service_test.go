package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"uptime-backend/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestServiceRegister_IssuesAccessAndRefreshTokens(t *testing.T) {
	service, _ := newTestService()
	service.now = func() time.Time { return time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC) }
	pair, err := service.Register(context.Background(), RegistrationInput{Email: " Person@Example.com ", Name: "Person", Password: "secure-pass"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if pair.User.Email != "person@example.com" || pair.RefreshToken == "" || pair.ExpiresIn != int64((24*time.Hour).Seconds()) {
		t.Fatalf("unexpected token pair: %#v", pair)
	}
	parsed, err := jwt.Parse(pair.AccessToken, func(token *jwt.Token) (any, error) { return []byte("a sufficiently long test JWT secret value"), nil }, jwt.WithoutClaimsValidation())
	if err != nil || !parsed.Valid {
		t.Fatalf("parse access token: %v", err)
	}
	claims := parsed.Claims.(jwt.MapClaims)
	if claims["sub"] != pair.User.ID.String() || claims["token_type"] != "access" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}

func TestServiceRegister_RejectsInvalidInputAndDuplicateEmail(t *testing.T) {
	service, _ := newTestService()
	_, err := service.Register(context.Background(), RegistrationInput{Email: "not-an-email", Name: "Person", Password: "secure-pass"})
	if !errors.Is(err, ErrInvalidRegistrationInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
	input := RegistrationInput{Email: "person@example.com", Name: "Person", Password: "secure-pass"}
	if _, err := service.Register(context.Background(), input); err != nil {
		t.Fatalf("first registration: %v", err)
	}
	_, err = service.Register(context.Background(), input)
	if !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("expected duplicate email, got %v", err)
	}
}

func TestServiceLoginAndRefresh_RotatesRefreshToken(t *testing.T) {
	service, _ := newTestService()
	pair, err := service.Register(context.Background(), RegistrationInput{Email: "person@example.com", Name: "Person", Password: "secure-pass"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := service.Login(context.Background(), "person@example.com", "wrong-password"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
	loginPair, err := service.Login(context.Background(), "PERSON@example.com", "secure-pass")
	if err != nil || loginPair.AccessToken == "" {
		t.Fatalf("login: %v", err)
	}
	rotatedPair, err := service.Refresh(context.Background(), pair.RefreshToken)
	if err != nil || rotatedPair.RefreshToken == "" || rotatedPair.RefreshToken == pair.RefreshToken {
		t.Fatalf("refresh: pair=%#v err=%v", rotatedPair, err)
	}
	if _, err := service.Refresh(context.Background(), pair.RefreshToken); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("expected old token rejection, got %v", err)
	}
}

func TestServiceLogout_RevokesRefreshToken(t *testing.T) {
	service, store := newTestService()
	pair, err := service.Register(context.Background(), RegistrationInput{Email: "person@example.com", Name: "Person", Password: "secure-pass"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := service.Logout(context.Background(), pair.RefreshToken); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := service.Refresh(context.Background(), pair.RefreshToken); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("expected revoked token rejection, got %v", err)
	}
	if len(store.sessions) != 1 {
		t.Fatalf("unexpected session count: %d", len(store.sessions))
	}
}

func TestServiceUpdateProfile_NameWithSpaceUpdatesName(t *testing.T) {
	service, _ := newTestService()
	user, err := service.Register(context.Background(), RegistrationInput{Email: "person@example.com", Name: "Person", Password: "secure-pass"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	updated, err := service.UpdateProfile(context.Background(), user.User.ID, "Updated Person")
	if err != nil {
		t.Fatalf("update profile: %v", err)
	}
	if updated.Name != "Updated Person" {
		t.Fatalf("expected updated name, got %q", updated.Name)
	}
}

type testUserStore struct{ users map[string]models.User }

type testSessionStore struct {
	users    *testUserStore
	sessions map[string]models.RefreshSession
}

func newTestService() (*Service, *testSessionStore) {
	users := &testUserStore{users: make(map[string]models.User)}
	sessions := &testSessionStore{users: users, sessions: make(map[string]models.RefreshSession)}
	return NewService(users, sessions, "a sufficiently long test JWT secret value", 24*time.Hour, 30*24*time.Hour), sessions
}

func (store *testUserStore) Create(_ context.Context, user models.User) error {
	if _, exists := store.users[user.Email]; exists {
		return ErrEmailTaken
	}
	store.users[user.Email] = user
	return nil
}

func (store *testUserStore) FindByEmail(_ context.Context, email string) (models.User, error) {
	user, exists := store.users[email]
	if !exists {
		return models.User{}, ErrInvalidCredentials
	}
	return user, nil
}

func (store *testUserStore) FindByID(_ context.Context, id uuid.UUID) (models.User, error) {
	for _, user := range store.users {
		if user.ID == id {
			return user, nil
		}
	}
	return models.User{}, ErrInvalidCredentials
}

func (store *testUserStore) UpdateName(_ context.Context, id uuid.UUID, name string) (models.User, error) {
	for email, user := range store.users {
		if user.ID == id {
			user.Name = name
			store.users[email] = user
			return user, nil
		}
	}
	return models.User{}, ErrInvalidCredentials
}

func (store *testUserStore) UpdateAvatar(_ context.Context, id uuid.UUID, avatarPath string) (models.User, error) {
	for email, user := range store.users {
		if user.ID == id {
			user.AvatarPath = avatarPath
			store.users[email] = user
			return user, nil
		}
	}
	return models.User{}, ErrInvalidCredentials
}

func (store *testSessionStore) Create(_ context.Context, session models.RefreshSession) error {
	store.sessions[session.TokenHash] = session
	return nil
}

func (store *testSessionStore) Rotate(_ context.Context, oldHash string, replacement models.RefreshSession, now time.Time) (models.User, error) {
	session, exists := store.sessions[oldHash]
	if !exists || session.RevokedAt != nil || !session.ExpiresAt.After(now) {
		return models.User{}, ErrInvalidRefreshToken
	}
	session.RevokedAt = &now
	store.sessions[oldHash] = session
	replacement.UserID = session.UserID
	store.sessions[replacement.TokenHash] = replacement
	for _, user := range store.users.users {
		if user.ID == session.UserID {
			return user, nil
		}
	}
	return models.User{}, ErrInvalidRefreshToken
}

func (store *testSessionStore) Revoke(_ context.Context, tokenHash string, now time.Time) error {
	session, exists := store.sessions[tokenHash]
	if !exists || session.RevokedAt != nil {
		return nil
	}
	session.RevokedAt = &now
	store.sessions[tokenHash] = session
	return nil
}
