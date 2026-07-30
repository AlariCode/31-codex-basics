package auth

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestControllerRegister_SetsRefreshCookieAndReturnsAccessToken(t *testing.T) {
	handler := newTestControllerHandler()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"email":"person@example.com","name":"Person","password":"secure-pass"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != refreshCookieName || !cookies[0].HttpOnly || cookies[0].Path != "/api/v1/auth" {
		t.Fatalf("unexpected cookies: %#v", cookies)
	}
	var body tokenResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil || body.AccessToken == "" || body.TokenType != "Bearer" {
		t.Fatalf("decode response: body=%#v err=%v", body, err)
	}
}

func TestControllerRefresh_WithoutCookieClearsCookie(t *testing.T) {
	response := httptest.NewRecorder()
	newTestControllerHandler().ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", response.Code)
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || cookies[0].MaxAge >= 0 {
		t.Fatalf("expected cleared cookie, got %#v", cookies)
	}
}

func TestControllerLogout_ClearsRefreshCookie(t *testing.T) {
	response := httptest.NewRecorder()
	newTestControllerHandler().ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d", response.Code)
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || cookies[0].MaxAge >= 0 {
		t.Fatalf("expected cleared cookie, got %#v", cookies)
	}
}

func TestControllerProfile_ReadsAndUpdatesAuthenticatedUser(t *testing.T) {
	handler := newTestControllerHandler()
	register := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"email":"person@example.com","name":"Person","password":"secure-pass"}`))
	register.Header.Set("Content-Type", "application/json")
	registerResponse := httptest.NewRecorder()
	handler.ServeHTTP(registerResponse, register)
	var authBody tokenResponse
	if err := json.NewDecoder(registerResponse.Body).Decode(&authBody); err != nil {
		t.Fatalf("decode registration: %v", err)
	}

	getRequest := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	getRequest.Header.Set("Authorization", "Bearer "+authBody.AccessToken)
	getResponse := httptest.NewRecorder()
	handler.ServeHTTP(getResponse, getRequest)
	if getResponse.Code != http.StatusOK || !strings.Contains(getResponse.Body.String(), `"name":"Person"`) {
		t.Fatalf("get profile: status=%d body=%s", getResponse.Code, getResponse.Body.String())
	}

	updateRequest := httptest.NewRequest(http.MethodPatch, "/api/v1/profile", strings.NewReader(`{"name":"Updated Person"}`))
	updateRequest.Header.Set("Authorization", "Bearer "+authBody.AccessToken)
	updateRequest.Header.Set("Content-Type", "application/json")
	updateResponse := httptest.NewRecorder()
	handler.ServeHTTP(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusOK || !strings.Contains(updateResponse.Body.String(), `"name":"Updated Person"`) {
		t.Fatalf("update profile: status=%d body=%s", updateResponse.Code, updateResponse.Body.String())
	}
}

func TestControllerUploadAvatar_StoresImageAndReturnsURL(t *testing.T) {
	service, _ := newTestService()
	routes := http.NewServeMux()
	avatarDir := t.TempDir()
	NewController(service, HTTPConfig{RefreshTokenTTL: 30 * 24 * time.Hour, AvatarDir: avatarDir}).RegisterRoutes(routes)

	register := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"email":"person@example.com","name":"Person","password":"secure-pass"}`))
	register.Header.Set("Content-Type", "application/json")
	registerResponse := httptest.NewRecorder()
	routes.ServeHTTP(registerResponse, register)
	var authBody tokenResponse
	if err := json.NewDecoder(registerResponse.Body).Decode(&authBody); err != nil {
		t.Fatalf("decode registration: %v", err)
	}

	var imageBytes bytes.Buffer
	if err := png.Encode(&imageBytes, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatalf("encode image: %v", err)
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("avatar", "avatar.png")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(imageBytes.Bytes()); err != nil {
		t.Fatalf("write image: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}

	upload := httptest.NewRequest(http.MethodPost, "/api/v1/profile/avatar", &body)
	upload.Header.Set("Authorization", "Bearer "+authBody.AccessToken)
	upload.Header.Set("Content-Type", writer.FormDataContentType())
	uploadResponse := httptest.NewRecorder()
	routes.ServeHTTP(uploadResponse, upload)
	if uploadResponse.Code != http.StatusOK || !strings.Contains(uploadResponse.Body.String(), `"avatar_url":"/uploads/avatars/`) {
		t.Fatalf("upload: status=%d body=%s", uploadResponse.Code, uploadResponse.Body.String())
	}
	entries, err := os.ReadDir(avatarDir)
	if err != nil || len(entries) != 1 || filepath.Ext(entries[0].Name()) != ".png" {
		t.Fatalf("stored avatar: entries=%#v err=%v", entries, err)
	}
}

func TestControllerUploadFile_AcceptsNonImageFiles(t *testing.T) {
	service, _ := newTestService()
	routes := http.NewServeMux()
	avatarDir := t.TempDir()
	NewController(service, HTTPConfig{RefreshTokenTTL: 30 * 24 * time.Hour, AvatarDir: avatarDir}).RegisterRoutes(routes)
	authBody := registerTestUser(t, routes)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "notes.txt")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write([]byte("hello")); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}

	upload := httptest.NewRequest(http.MethodPost, "/api/v1/uploads", &body)
	upload.Header.Set("Authorization", "Bearer "+authBody.AccessToken)
	upload.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	routes.ServeHTTP(response, upload)
	if response.Code != http.StatusCreated || !strings.Contains(response.Body.String(), `"content_type":"text/plain; charset=utf-8"`) {
		t.Fatalf("upload: status=%d body=%s", response.Code, response.Body.String())
	}
	entries, err := os.ReadDir(filepath.Join(avatarDir, "..", "files"))
	if err != nil || len(entries) != 1 || filepath.Ext(entries[0].Name()) != ".txt" {
		t.Fatalf("stored file: entries=%#v err=%v", entries, err)
	}
}

func registerTestUser(t *testing.T, routes http.Handler) tokenResponse {
	t.Helper()
	register := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"email":"person@example.com","name":"Person","password":"secure-pass"}`))
	register.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	routes.ServeHTTP(response, register)
	var authBody tokenResponse
	if err := json.NewDecoder(response.Body).Decode(&authBody); err != nil {
		t.Fatalf("decode registration: %v", err)
	}
	return authBody
}

func newTestControllerHandler() http.Handler {
	service, _ := newTestService()
	routes := http.NewServeMux()
	NewController(service, HTTPConfig{RefreshTokenTTL: 30 * 24 * time.Hour}).RegisterRoutes(routes)
	return routes
}
