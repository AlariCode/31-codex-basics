package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"uptime-backend/internal/models"

	"github.com/google/uuid"
)

const refreshCookieName = "refresh_token"

const maxAvatarSize = 5 << 20
const maxUploadSize = 20 << 20

// HTTPConfig contains transport settings that affect cookie and upload behavior.
type HTTPConfig struct {
	RefreshTokenTTL time.Duration
	CookieSecure    bool
	AvatarDir       string
}

// Controller translates HTTP requests into authentication service calls and responses.
type Controller struct {
	service      *Service
	refreshTTL   time.Duration
	cookieSecure bool
	avatarDir    string
}

// NewController applies upload defaults while keeping cookie behavior environment-specific.
func NewController(service *Service, config HTTPConfig) *Controller {
	avatarDir := config.AvatarDir
	if avatarDir == "" {
		avatarDir = "uploads/avatars"
	}
	return &Controller{service: service, refreshTTL: config.RefreshTokenTTL, cookieSecure: config.CookieSecure, avatarDir: avatarDir}
}

// RegisterRoutes exposes authentication and profile endpoints on the provided mux.
func (controller *Controller) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/register", controller.register)
	mux.HandleFunc("POST /api/v1/auth/login", controller.login)
	mux.HandleFunc("POST /api/v1/auth/refresh", controller.refresh)
	mux.HandleFunc("POST /api/v1/auth/logout", controller.logout)
	mux.HandleFunc("GET /api/v1/profile", controller.profile)
	mux.HandleFunc("PATCH /api/v1/profile", controller.updateProfile)
	mux.HandleFunc("POST /api/v1/profile/avatar", controller.uploadAvatar)
	mux.HandleFunc("POST /api/v1/upload", controller.uploadFile)
	mux.HandleFunc("POST /api/v1/uploads", controller.uploadFile)
}

type credentialsRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type profileRequest struct {
	Name string `json:"name"`
}

type userResponse struct {
	ID        string  `json:"id"`
	Email     string  `json:"email"`
	Name      string  `json:"name"`
	AvatarURL *string `json:"avatar_url"`
}

type tokenResponse struct {
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type"`
	ExpiresIn   int64        `json:"expires_in"`
	User        userResponse `json:"user"`
}

type uploadResponse struct {
	URL         string `json:"url"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

func (controller *Controller) register(writer http.ResponseWriter, request *http.Request) {
	var body credentialsRequest
	if !decodeJSON(writer, request, &body) {
		return
	}
	pair, err := controller.service.Register(request.Context(), RegistrationInput{Email: body.Email, Name: body.Name, Password: body.Password})
	if err != nil {
		controller.writeAuthError(writer, err)
		return
	}
	controller.writeTokens(writer, http.StatusCreated, pair)
}

func (controller *Controller) login(writer http.ResponseWriter, request *http.Request) {
	var body credentialsRequest
	if !decodeJSON(writer, request, &body) {
		return
	}
	pair, err := controller.service.Login(request.Context(), body.Email, body.Password)
	if err != nil {
		controller.writeAuthError(writer, err)
		return
	}
	controller.writeTokens(writer, http.StatusOK, pair)
}

func (controller *Controller) refresh(writer http.ResponseWriter, request *http.Request) {
	cookie, err := request.Cookie(refreshCookieName)
	if err != nil {
		controller.clearRefreshCookie(writer)
		writeError(writer, http.StatusUnauthorized, "invalid refresh token")
		return
	}
	pair, err := controller.service.Refresh(request.Context(), cookie.Value)
	if err != nil {
		controller.clearRefreshCookie(writer)
		controller.writeAuthError(writer, err)
		return
	}
	controller.writeTokens(writer, http.StatusOK, pair)
}

func (controller *Controller) logout(writer http.ResponseWriter, request *http.Request) {
	cookie, err := request.Cookie(refreshCookieName)
	if err == nil {
		if err := controller.service.Logout(request.Context(), cookie.Value); err != nil {
			writeError(writer, http.StatusInternalServerError, "internal server error")
			return
		}
	}
	controller.clearRefreshCookie(writer)
	writer.WriteHeader(http.StatusNoContent)
}

func (controller *Controller) profile(writer http.ResponseWriter, request *http.Request) {
	userID, ok := controller.userID(writer, request)
	if !ok {
		return
	}
	user, err := controller.service.Profile(request.Context(), userID)
	if err != nil {
		writeError(writer, http.StatusUnauthorized, "invalid credentials")
		return
	}
	writeJSON(writer, http.StatusOK, responseUser(user))
}

func (controller *Controller) updateProfile(writer http.ResponseWriter, request *http.Request) {
	userID, ok := controller.userID(writer, request)
	if !ok {
		return
	}
	var body profileRequest
	if !decodeJSON(writer, request, &body) {
		return
	}
	user, err := controller.service.UpdateProfile(request.Context(), userID, body.Name)
	if errors.Is(err, ErrInvalidProfileInput) {
		writeError(writer, http.StatusBadRequest, "invalid profile input")
		return
	}
	if err != nil {
		writeError(writer, http.StatusUnauthorized, "invalid credentials")
		return
	}
	writeJSON(writer, http.StatusOK, responseUser(user))
}

func (controller *Controller) uploadAvatar(writer http.ResponseWriter, request *http.Request) {
	userID, ok := controller.userID(writer, request)
	if !ok {
		return
	}
	previousUser, err := controller.service.Profile(request.Context(), userID)
	if err != nil {
		writeError(writer, http.StatusUnauthorized, "invalid credentials")
		return
	}
	request.Body = http.MaxBytesReader(writer, request.Body, maxAvatarSize+1<<20)
	if err := request.ParseMultipartForm(maxAvatarSize + 1<<20); err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			writeError(writer, http.StatusRequestEntityTooLarge, "avatar is too large")
			return
		}
		writeError(writer, http.StatusBadRequest, "invalid avatar upload")
		return
	}
	defer request.MultipartForm.RemoveAll()
	file, header, err := request.FormFile("avatar")
	if err != nil {
		writeError(writer, http.StatusBadRequest, "avatar file is required")
		return
	}
	defer file.Close()
	if header.Size <= 0 || header.Size > maxAvatarSize {
		writeError(writer, http.StatusRequestEntityTooLarge, "avatar is too large")
		return
	}

	contentType, err := avatarContentType(file)
	if err != nil {
		writeError(writer, http.StatusBadRequest, "avatar must be a JPEG or PNG image")
		return
	}
	extension := ".jpg"
	if contentType == "image/png" {
		extension = ".png"
	}
	filename, path, err := storeUploadedFile(file, controller.avatarDir, maxAvatarSize, extension)
	if err != nil {
		if errors.Is(err, errUploadTooLarge) {
			writeError(writer, http.StatusRequestEntityTooLarge, "avatar is too large")
			return
		}
		writeError(writer, http.StatusInternalServerError, "internal server error")
		return
	}

	user, err := controller.service.UpdateAvatar(request.Context(), userID, filename)
	if err != nil {
		_ = os.Remove(path)
		writeError(writer, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if previousUser.AvatarPath != "" && previousUser.AvatarPath != filename {
		_ = os.Remove(filepath.Join(controller.avatarDir, filepath.Base(previousUser.AvatarPath)))
	}
	writeJSON(writer, http.StatusOK, responseUser(user))
}

func (controller *Controller) uploadFile(writer http.ResponseWriter, request *http.Request) {
	if _, ok := controller.userID(writer, request); !ok {
		return
	}
	request.Body = http.MaxBytesReader(writer, request.Body, maxUploadSize+1<<20)
	if err := request.ParseMultipartForm(maxUploadSize + 1<<20); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid file upload")
		return
	}
	defer request.MultipartForm.RemoveAll()
	file, header, err := request.FormFile("file")
	if err != nil {
		writeError(writer, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()
	if header.Size <= 0 || header.Size > maxUploadSize {
		writeError(writer, http.StatusRequestEntityTooLarge, "file is too large")
		return
	}
	contentType, err := detectedContentType(file)
	if err != nil {
		writeError(writer, http.StatusBadRequest, "invalid file upload")
		return
	}
	extension := safeExtension(header.Filename)
	filename, _, err := storeUploadedFile(file, filepath.Join(controller.avatarDir, "..", "files"), maxUploadSize, extension)
	if err != nil {
		if errors.Is(err, errUploadTooLarge) {
			writeError(writer, http.StatusRequestEntityTooLarge, "file is too large")
			return
		}
		writeError(writer, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(writer, http.StatusCreated, uploadResponse{URL: "/uploads/files/" + filename, Filename: filename, ContentType: contentType, Size: header.Size})
}

var errUploadTooLarge = errors.New("uploaded file is too large")

func storeUploadedFile(file multipart.File, directory string, maxSize int64, extension string) (string, string, error) {
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", "", err
	}
	filename := uuid.NewString() + extension
	path := filepath.Join(directory, filename)
	output, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", "", err
	}
	bytesCopied, copyErr := io.Copy(output, io.LimitReader(file, maxSize+1))
	closeErr := output.Close()
	if copyErr != nil || closeErr != nil || bytesCopied > maxSize {
		_ = os.Remove(path)
		if bytesCopied > maxSize {
			return "", "", errUploadTooLarge
		}
		return "", "", fmt.Errorf("store uploaded file: %w", copyErr)
	}
	return filename, path, nil
}

func safeExtension(filename string) string {
	extension := filepath.Ext(filename)
	if len(extension) > 10 || extension == "." {
		return ""
	}
	for _, character := range extension[1:] {
		if (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') && (character < '0' || character > '9') {
			return ""
		}
	}
	return extension
}

func avatarContentType(file io.ReadSeeker) (string, error) {
	contentType, err := detectedContentType(file)
	if err != nil {
		return "", err
	}
	if contentType != "image/jpeg" && contentType != "image/png" {
		return "", fmt.Errorf("unsupported avatar type %q", contentType)
	}
	if _, _, err := image.DecodeConfig(file); err != nil {
		return "", err
	}
	_, err = file.Seek(0, io.SeekStart)
	return contentType, err
}

func detectedContentType(file io.ReadSeeker) (string, error) {
	header := make([]byte, 512)
	read, err := file.Read(header)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	return http.DetectContentType(header[:read]), nil
}

func (controller *Controller) userID(writer http.ResponseWriter, request *http.Request) (uuid.UUID, bool) {
	userID, err := controller.service.UserIDFromRequest(request)
	if err != nil {
		writeError(writer, http.StatusUnauthorized, "invalid credentials")
		return uuid.Nil, false
	}
	return userID, true
}

func (controller *Controller) writeTokens(writer http.ResponseWriter, status int, pair TokenPair) {
	http.SetCookie(writer, &http.Cookie{Name: refreshCookieName, Value: pair.RefreshToken, Path: "/api/v1/auth", MaxAge: int(controller.refreshTTL.Seconds()), Expires: time.Now().Add(controller.refreshTTL), HttpOnly: true, Secure: controller.cookieSecure, SameSite: http.SameSiteLaxMode})
	writeJSON(writer, status, tokenResponse{AccessToken: pair.AccessToken, TokenType: "Bearer", ExpiresIn: pair.ExpiresIn, User: responseUser(pair.User)})
}

func (controller *Controller) clearRefreshCookie(writer http.ResponseWriter) {
	http.SetCookie(writer, &http.Cookie{Name: refreshCookieName, Value: "", Path: "/api/v1/auth", MaxAge: -1, Expires: time.Unix(0, 0), HttpOnly: true, Secure: controller.cookieSecure, SameSite: http.SameSiteLaxMode})
}

func (controller *Controller) writeAuthError(writer http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidRegistrationInput):
		writeError(writer, http.StatusBadRequest, "invalid registration input")
	case errors.Is(err, ErrEmailTaken):
		writeError(writer, http.StatusConflict, "email is already registered")
	case errors.Is(err, ErrInvalidCredentials), errors.Is(err, ErrInvalidRefreshToken):
		writeError(writer, http.StatusUnauthorized, "invalid credentials")
	default:
		writeError(writer, http.StatusInternalServerError, "internal server error")
	}
}

func decodeJSON(writer http.ResponseWriter, request *http.Request, target any) bool {
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

func responseUser(user models.User) userResponse {
	var avatarURL *string
	if user.AvatarPath != "" {
		value := "/uploads/avatars/" + filepath.Base(user.AvatarPath)
		avatarURL = &value
	}
	return userResponse{ID: user.ID.String(), Email: user.Email, Name: user.Name, AvatarURL: avatarURL}
}

func writeError(writer http.ResponseWriter, status int, message string) {
	writeJSON(writer, status, map[string]string{"error": message})
}

func writeJSON(writer http.ResponseWriter, status int, body any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(body)
}
