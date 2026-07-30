package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"uptime-backend/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const refreshCookieName = "refresh_token"

// HTTPConfig contains transport settings for the authentication controller.
type HTTPConfig struct {
	RefreshTokenTTL time.Duration
	CookieSecure    bool
}

// Controller exposes authentication operations over HTTP.
type Controller struct {
	service      *Service
	refreshTTL   time.Duration
	cookieSecure bool
}

// NewController creates an authentication HTTP controller.
func NewController(service *Service, config HTTPConfig) *Controller {
	return &Controller{service: service, refreshTTL: config.RefreshTokenTTL, cookieSecure: config.CookieSecure}
}

// RegisterRoutes adds authentication routes to the provided mux.
func (controller *Controller) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/register", controller.register)
	mux.HandleFunc("POST /api/v1/auth/login", controller.login)
	mux.HandleFunc("POST /api/v1/auth/refresh", controller.refresh)
	mux.HandleFunc("POST /api/v1/auth/logout", controller.logout)
	mux.HandleFunc("GET /api/v1/profile", controller.profile)
	mux.HandleFunc("PATCH /api/v1/profile", controller.updateProfile)
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
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type tokenResponse struct {
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type"`
	ExpiresIn   int64        `json:"expires_in"`
	User        userResponse `json:"user"`
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

func (controller *Controller) userID(writer http.ResponseWriter, request *http.Request) (uuid.UUID, bool) {
	const prefix = "Bearer "
	header := request.Header.Get("Authorization")
	if !strings.HasPrefix(header, prefix) {
		writeError(writer, http.StatusUnauthorized, "invalid credentials")
		return uuid.Nil, false
	}
	token, err := jwt.Parse(strings.TrimSpace(strings.TrimPrefix(header, prefix)), func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return controller.service.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		writeError(writer, http.StatusUnauthorized, "invalid credentials")
		return uuid.Nil, false
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["token_type"] != "access" {
		writeError(writer, http.StatusUnauthorized, "invalid credentials")
		return uuid.Nil, false
	}
	sub, ok := claims["sub"].(string)
	userID, err := uuid.Parse(sub)
	if !ok || err != nil || userID == uuid.Nil {
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
	return userResponse{ID: user.ID.String(), Email: user.Email, Name: user.Name}
}

func writeError(writer http.ResponseWriter, status int, message string) {
	writeJSON(writer, status, map[string]string{"error": message})
}

func writeJSON(writer http.ResponseWriter, status int, body any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(body)
}
