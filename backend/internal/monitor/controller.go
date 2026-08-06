package monitor

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"uptime-backend/internal/models"

	"github.com/google/uuid"
)

// Controller keeps monitor HTTP handling dependent on authentication and storage contracts.
type Controller struct {
	auth  Authenticator
	store Store
}

// Authenticator lets protected handlers share token validation without coupling them to auth storage.
type Authenticator interface {
	UserIDFromRequest(*http.Request) (uuid.UUID, error)
}

const (
	maxMonitorBodySize  = 16 << 10
	maxMonitorURLLength = 2048
	minMonitorInterval  = 1
	maxMonitorInterval  = 7 * 24 * 60 * 60
)

// NewController wires monitor HTTP handling to authentication and storage dependencies.
func NewController(authenticator Authenticator, store Store) *Controller {
	return &Controller{auth: authenticator, store: store}
}

// RegisterRoutes exposes only the monitor operations supported by this API version.
func (controller *Controller) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/monitors", controller.list)
	mux.HandleFunc("POST /api/v1/monitors", controller.create)
}

type createRequest struct {
	URL             string `json:"url"`
	IntervalSeconds int    `json:"interval_seconds"`
}

type response struct {
	ID              string `json:"id"`
	URL             string `json:"url"`
	IntervalSeconds int    `json:"interval_seconds"`
}

func (controller *Controller) create(writer http.ResponseWriter, request *http.Request) {
	userID, ok := controller.userID(writer, request)
	if !ok {
		return
	}
	request.Body = http.MaxBytesReader(writer, request.Body, maxMonitorBodySize)
	var body createRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil || !validMonitorInput(body) {
		writeError(writer, http.StatusBadRequest, "invalid monitor input")
		return
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		writeError(writer, http.StatusBadRequest, "invalid monitor input")
		return
	}
	value := models.Monitor{ID: uuid.New(), UserID: userID, URL: strings.TrimSpace(body.URL), IntervalSeconds: body.IntervalSeconds}
	if err := controller.store.Create(request.Context(), value); err != nil {
		writeError(writer, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(writer, http.StatusCreated, monitorResponse(value))
}

func (controller *Controller) list(writer http.ResponseWriter, request *http.Request) {
	userID, ok := controller.userID(writer, request)
	if !ok {
		return
	}
	monitors, err := controller.store.List(request.Context(), userID)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "internal server error")
		return
	}
	result := make([]response, 0, len(monitors))
	for _, value := range monitors {
		result = append(result, monitorResponse(value))
	}
	writeJSON(writer, http.StatusOK, result)
}

func (controller *Controller) userID(writer http.ResponseWriter, request *http.Request) (uuid.UUID, bool) {
	userID, err := controller.auth.UserIDFromRequest(request)
	if err != nil {
		writeError(writer, http.StatusUnauthorized, "invalid credentials")
		return uuid.Nil, false
	}
	return userID, true
}

func validURL(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > maxMonitorURLLength {
		return false
	}
	parsed, err := url.ParseRequestURI(strings.TrimSpace(value))
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Hostname() != "" && parsed.User == nil
}

func validMonitorInput(value createRequest) bool {
	return validURL(value.URL) && value.IntervalSeconds >= minMonitorInterval && value.IntervalSeconds <= maxMonitorInterval
}

func monitorResponse(value models.Monitor) response {
	return response{ID: value.ID.String(), URL: value.URL, IntervalSeconds: value.IntervalSeconds}
}

func writeError(writer http.ResponseWriter, status int, message string) {
	writeJSON(writer, status, map[string]string{"error": message})
}

func writeJSON(writer http.ResponseWriter, status int, body any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(body)
}
