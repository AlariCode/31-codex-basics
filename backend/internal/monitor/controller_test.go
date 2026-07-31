package monitor

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"uptime-backend/internal/models"

	"github.com/google/uuid"
)

type fakeAuthenticator struct {
	userID uuid.UUID
	err    error
}

func (authenticator fakeAuthenticator) UserIDFromRequest(*http.Request) (uuid.UUID, error) {
	if authenticator.err != nil {
		return uuid.Nil, authenticator.err
	}
	return authenticator.userID, nil
}

type fakeStore struct {
	created []models.Monitor
	listed  []models.Monitor
	err     error
}

func (store *fakeStore) Create(_ context.Context, value models.Monitor) error {
	if store.err != nil {
		return store.err
	}
	store.created = append(store.created, value)
	return nil
}

func (store *fakeStore) List(_ context.Context, userID uuid.UUID) ([]models.Monitor, error) {
	if store.err != nil {
		return nil, store.err
	}
	result := make([]models.Monitor, 0)
	for _, value := range store.listed {
		if value.UserID == userID {
			result = append(result, value)
		}
	}
	return result, nil
}

func TestValidURL_AcceptsHTTPAndHTTPS(t *testing.T) {
	for _, value := range []string{"https://example.com", "http://localhost:8080/health"} {
		if !validURL(value) {
			t.Errorf("validURL(%q) = false", value)
		}
	}
}

func TestValidURL_RejectsMissingSchemeOrHost(t *testing.T) {
	for _, value := range []string{"example.com", "ftp://example.com", "https://", "https://user:pass@example.com"} {
		if validURL(value) {
			t.Errorf("validURL(%q) = true", value)
		}
	}
}

func TestControllerCreate_StoresAuthenticatedOwner(t *testing.T) {
	userID := uuid.New()
	store := &fakeStore{}
	routes := http.NewServeMux()
	NewController(fakeAuthenticator{userID: userID}, store).RegisterRoutes(routes)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/monitors", strings.NewReader(`{"url":" https://example.com/health ","interval_seconds":60}`))
	response := httptest.NewRecorder()
	routes.ServeHTTP(response, request)

	if response.Code != http.StatusCreated || len(store.created) != 1 {
		t.Fatalf("status=%d created=%#v body=%s", response.Code, store.created, response.Body.String())
	}
	if store.created[0].UserID != userID || store.created[0].URL != "https://example.com/health" {
		t.Fatalf("stored monitor=%#v", store.created[0])
	}
}

func TestControllerCreate_RejectsInvalidInputAndOversizedBody(t *testing.T) {
	for name, body := range map[string]string{
		"unknown field":  `{"url":"https://example.com","interval_seconds":60,"extra":true}`,
		"trailing JSON":  `{"url":"https://example.com","interval_seconds":60}{}`,
		"large URL":      `{"url":"https://` + strings.Repeat("a", maxMonitorURLLength) + `","interval_seconds":60}`,
		"large body":     `{"url":"https://example.com","interval_seconds":60,"padding":"` + strings.Repeat("a", maxMonitorBodySize) + `"}`,
		"large interval": `{"url":"https://example.com","interval_seconds":604801}`,
	} {
		t.Run(name, func(t *testing.T) {
			routes := http.NewServeMux()
			NewController(fakeAuthenticator{userID: uuid.New()}, &fakeStore{}).RegisterRoutes(routes)
			request := httptest.NewRequest(http.MethodPost, "/api/v1/monitors", strings.NewReader(body))
			response := httptest.NewRecorder()
			routes.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}
}

func TestControllerProtectedRoutes_ReturnUnauthorized(t *testing.T) {
	routes := http.NewServeMux()
	NewController(fakeAuthenticator{err: errors.New("invalid token")}, &fakeStore{}).RegisterRoutes(routes)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/monitors", nil)
	response := httptest.NewRecorder()
	routes.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestControllerList_ReturnsOnlyAuthenticatedOwner(t *testing.T) {
	userID := uuid.New()
	store := &fakeStore{listed: []models.Monitor{
		{ID: uuid.New(), UserID: userID, URL: "https://one.example", IntervalSeconds: 60},
		{ID: uuid.New(), UserID: uuid.New(), URL: "https://other.example", IntervalSeconds: 120},
	}}
	routes := http.NewServeMux()
	NewController(fakeAuthenticator{userID: userID}, store).RegisterRoutes(routes)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/monitors", nil)
	recorder := httptest.NewRecorder()
	routes.ServeHTTP(recorder, request)

	var body []response
	if recorder.Code != http.StatusOK || json.NewDecoder(recorder.Body).Decode(&body) != nil || len(body) != 1 || body[0].URL != "https://one.example" {
		t.Fatalf("status=%d body=%s decoded=%#v", recorder.Code, recorder.Body.String(), body)
	}
}
