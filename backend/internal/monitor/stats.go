package monitor

import (
	"context"
	"net/http"
	"time"

	"uptime-backend/internal/models"

	"github.com/google/uuid"
)

type statisticsStore interface {
	List(context.Context, uuid.UUID) ([]models.Monitor, error)
	readCounts(context.Context, uuid.UUID, statsWindow) ([]minuteCount, error)
}

type statsWindow struct {
	From, To time.Time
	Step     time.Duration
}

type statsPoint struct {
	Start         time.Time `json:"start"`
	End           time.Time `json:"end"`
	Successes     int64     `json:"successes"`
	Failures      int64     `json:"failures"`
	UptimePercent *float64  `json:"uptime_percent"`
}

type monitorStats struct {
	MonitorID     uuid.UUID    `json:"monitor_id"`
	Successes     int64        `json:"successes"`
	Failures      int64        `json:"failures"`
	UptimePercent *float64     `json:"uptime_percent"`
	Points        []statsPoint `json:"points"`
}

type statsResponse struct {
	From          time.Time      `json:"from"`
	To            time.Time      `json:"to"`
	BucketSeconds int            `json:"bucket_seconds"`
	Monitors      []monitorStats `json:"monitors"`
}

func periodWindow(period string, now time.Time) (statsWindow, bool) {
	var duration, step time.Duration
	switch period {
	case "1h":
		duration, step = time.Hour, time.Minute
	case "", "24h":
		duration, step = 24*time.Hour, 15*time.Minute
	case "7d":
		duration, step = 7*24*time.Hour, time.Hour
	case "30d":
		duration, step = 30*24*time.Hour, 6*time.Hour
	default:
		return statsWindow{}, false
	}
	// Exclude the partially expired oldest minute: its individual observations no longer exist.
	from := now.UTC().Add(-duration)
	if from != from.Truncate(time.Minute) {
		from = from.Truncate(time.Minute).Add(time.Minute)
	}
	return statsWindow{From: from, To: now.UTC(), Step: step}, true
}

func percentage(successes, failures int64) *float64 {
	if successes+failures == 0 {
		return nil
	}
	value := 100 * float64(successes) / float64(successes+failures)
	return &value
}

func buildStats(monitors []models.Monitor, counts []minuteCount, window statsWindow) statsResponse {
	indexed := map[uuid.UUID]map[int64]minuteCount{}
	for _, count := range counts {
		if indexed[count.MonitorID] == nil {
			indexed[count.MonitorID] = map[int64]minuteCount{}
		}
		indexed[count.MonitorID][count.BucketStart.Unix()] = count
	}
	result := statsResponse{
		From: window.From, To: window.To, BucketSeconds: int(window.Step.Seconds()), Monitors: []monitorStats{},
	}
	for _, monitor := range monitors {
		stats := monitorStats{MonitorID: monitor.ID, Points: []statsPoint{}}
		for start := window.From.Truncate(window.Step); start.Before(window.To); start = start.Add(window.Step) {
			count := indexed[monitor.ID][start.Unix()]
			end := start.Add(window.Step)
			if end.After(window.To) {
				end = window.To
			}
			pointStart := start
			if pointStart.Before(window.From) {
				pointStart = window.From
			}
			stats.Points = append(stats.Points, statsPoint{
				Start: pointStart, End: end, Successes: count.Successes, Failures: count.Failures,
				UptimePercent: percentage(count.Successes, count.Failures),
			})
			stats.Successes += count.Successes
			stats.Failures += count.Failures
		}
		stats.UptimePercent = percentage(stats.Successes, stats.Failures)
		result.Monitors = append(result.Monitors, stats)
	}
	return result
}

// stats returns owner-scoped observation ratios, with null percentages for missing data.
//
// @Summary Get monitor availability history
// @Description Percentages count completed checks, not elapsed uptime. UTC minute precision; the partially expired oldest minute is excluded. Empty buckets have null uptime_percent.
// @Tags monitors
// @Produce json
// @Security BearerAuth
// @Param period query string false "History period" Enums(1h,24h,7d,30d) default(24h)
// @Success 200 {object} statsResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/monitors/stats [get]
func (controller *Controller) stats(writer http.ResponseWriter, request *http.Request) {
	userID, ok := controller.userID(writer, request)
	if !ok {
		return
	}
	window, ok := periodWindow(request.URL.Query().Get("period"), time.Now())
	if !ok {
		writeError(writer, http.StatusBadRequest, "invalid period")
		return
	}
	if controller.statistics == nil {
		writeError(writer, http.StatusInternalServerError, "internal server error")
		return
	}
	monitors, err := controller.statistics.List(request.Context(), userID)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "internal server error")
		return
	}
	counts, err := controller.statistics.readCounts(request.Context(), userID, window)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(writer, http.StatusOK, buildStats(monitors, counts, window))
}
