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

func (store *fakeStore) Update(_ context.Context, userID uuid.UUID, value models.Monitor) error {
	if store.err != nil {
		return store.err
	}
	for index := range store.listed {
		if store.listed[index].ID == value.ID && store.listed[index].UserID == userID {
			store.listed[index] = value
			return nil
		}
	}
	return ErrNotFound
}

func (store *fakeStore) UpdateFavicon(_ context.Context, userID, monitorID uuid.UUID, faviconPath string) error {
	if store.err != nil {
		return store.err
	}
	for index := range store.listed {
		if store.listed[index].ID == monitorID && store.listed[index].UserID == userID {
			store.listed[index].FaviconPath = faviconPath
			return nil
		}
	}
	for index := range store.created {
		if store.created[index].ID == monitorID && store.created[index].UserID == userID {
			store.created[index].FaviconPath = faviconPath
			return nil
		}
	}
	return ErrNotFound
}

func (store *fakeStore) Delete(_ context.Context, userID, monitorID uuid.UUID) error {
	if store.err != nil {
		return store.err
	}
	for index := range store.listed {
		if store.listed[index].ID == monitorID && store.listed[index].UserID == userID {
			store.listed = append(store.listed[:index], store.listed[index+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

func TestValidURL_AcceptsHTTPAndHTTPS(t *testing.T) {
	for _, value := range []string{"https://example.com", "http://example.com:8080/health"} {
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

func TestControllerUpdate_UpdatesAuthenticatedOwner(t *testing.T) {
	userID := uuid.New()
	monitorID := uuid.New()
	store := &fakeStore{listed: []models.Monitor{{ID: monitorID, UserID: userID, URL: "https://old.example", IntervalSeconds: 60}}}
	routes := http.NewServeMux()
	NewController(fakeAuthenticator{userID: userID}, store).RegisterRoutes(routes)
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/monitors/"+monitorID.String(), strings.NewReader(`{"url":" https://new.example ","interval_seconds":120}`))
	recorder := httptest.NewRecorder()
	routes.ServeHTTP(recorder, request)

	var body response
	if recorder.Code != http.StatusOK || json.NewDecoder(recorder.Body).Decode(&body) != nil || body.URL != "https://new.example" || body.IntervalSeconds != 120 {
		t.Fatalf("status=%d body=%s decoded=%#v", recorder.Code, recorder.Body.String(), body)
	}
}

func TestControllerDelete_DeletesAuthenticatedOwner(t *testing.T) {
	userID := uuid.New()
	monitorID := uuid.New()
	store := &fakeStore{listed: []models.Monitor{{ID: monitorID, UserID: userID}}}
	routes := http.NewServeMux()
	NewController(fakeAuthenticator{userID: userID}, store).RegisterRoutes(routes)
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/monitors/"+monitorID.String(), nil)
	response := httptest.NewRecorder()
	routes.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent || len(store.listed) != 0 {
		t.Fatalf("status=%d body=%s listed=%#v", response.Code, response.Body.String(), store.listed)
	}
}

func TestControllerUpdateAndDelete_ReturnNotFoundForOtherOwner(t *testing.T) {
	ownerID := uuid.New()
	monitorID := uuid.New()
	store := &fakeStore{listed: []models.Monitor{{ID: monitorID, UserID: ownerID}}}
	routes := http.NewServeMux()
	NewController(fakeAuthenticator{userID: uuid.New()}, store).RegisterRoutes(routes)

	for _, method := range []string{http.MethodPatch, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			var body *strings.Reader
			if method == http.MethodPatch {
				body = strings.NewReader(`{"url":"https://new.example","interval_seconds":120}`)
			} else {
				body = strings.NewReader("")
			}
			request := httptest.NewRequest(method, "/api/v1/monitors/"+monitorID.String(), body)
			response := httptest.NewRecorder()
			routes.ServeHTTP(response, request)
			if response.Code != http.StatusNotFound {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}
}

func TestControllerUpdateAndDelete_RejectInvalidMonitorID(t *testing.T) {
	routes := http.NewServeMux()
	NewController(fakeAuthenticator{userID: uuid.New()}, &fakeStore{}).RegisterRoutes(routes)

	for _, method := range []string{http.MethodPatch, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			var body *strings.Reader
			if method == http.MethodPatch {
				body = strings.NewReader(`{"url":"https://example.com","interval_seconds":60}`)
			} else {
				body = strings.NewReader("")
			}
			request := httptest.NewRequest(method, "/api/v1/monitors/not-a-uuid", body)
			response := httptest.NewRecorder()
			routes.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}
}

type fakeFaviconResolver struct {
	path  string
	err   error
	calls []string
}

func (resolver *fakeFaviconResolver) Fetch(_ context.Context, siteURL string, _ uuid.UUID) (string, error) {
	resolver.calls = append(resolver.calls, siteURL)
	return resolver.path, resolver.err
}

func TestControllerCreate_StoresAndReturnsFetchedFavicon(t *testing.T) {
	userID := uuid.New()
	store := &fakeStore{}
	resolver := &fakeFaviconResolver{path: "/uploads/favicons/site.png"}
	routes := http.NewServeMux()
	NewController(fakeAuthenticator{userID: userID}, store, resolver).RegisterRoutes(routes)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/monitors", strings.NewReader(`{"url":"https://example.com","interval_seconds":60}`))
	recorder := httptest.NewRecorder()
	routes.ServeHTTP(recorder, request)

	var body response
	if recorder.Code != http.StatusCreated || json.NewDecoder(recorder.Body).Decode(&body) != nil || body.FaviconURL != resolver.path {
		t.Fatalf("status=%d body=%s decoded=%#v", recorder.Code, recorder.Body.String(), body)
	}
	if len(resolver.calls) != 1 || store.created[0].FaviconPath != resolver.path {
		t.Fatalf("calls=%#v stored=%#v", resolver.calls, store.created)
	}
}

func TestControllerList_DoesNotFetchMissingFavicon(t *testing.T) {
	userID := uuid.New()
	monitorID := uuid.New()
	store := &fakeStore{listed: []models.Monitor{{ID: monitorID, UserID: userID, URL: "https://example.com", IntervalSeconds: 60}}}
	resolver := &fakeFaviconResolver{path: "/uploads/favicons/site.png"}
	routes := http.NewServeMux()
	NewController(fakeAuthenticator{userID: userID}, store, resolver).RegisterRoutes(routes)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/monitors", nil)
	recorder := httptest.NewRecorder()
	routes.ServeHTTP(recorder, request)

	var body []response
	if recorder.Code != http.StatusOK || json.NewDecoder(recorder.Body).Decode(&body) != nil || len(body) != 1 || body[0].FaviconURL != "" {
		t.Fatalf("status=%d body=%s decoded=%#v", recorder.Code, recorder.Body.String(), body)
	}
	if store.listed[0].FaviconPath != "" || len(resolver.calls) != 0 {
		t.Fatalf("stored monitor=%#v", store.listed[0])
	}
}
