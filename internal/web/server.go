package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strconv"
	"time"

	"github.com/shenghuikevin/cc-track/internal/analysis"
	"github.com/shenghuikevin/cc-track/internal/config"
	"github.com/shenghuikevin/cc-track/internal/store"
)

//go:embed static/*
var staticFiles embed.FS

// Serve starts the web dashboard on the given port.
func Serve(port int) error {
	mux := http.NewServeMux()

	// Static files
	sub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return fmt.Errorf("web: embed fs: %w", err)
	}
	mux.Handle("/", http.FileServer(http.FS(sub)))

	// API routes
	mux.HandleFunc("/api/viewport", handleViewport)
	mux.HandleFunc("/api/summary", handleSummary)
	mux.HandleFunc("/api/trend", handleTrend)
	mux.HandleFunc("/api/sessions", handleSessions)
	mux.HandleFunc("/api/session/", handleSessionDetail)
	mux.HandleFunc("/api/waste", handleWaste)
	mux.HandleFunc("/api/roi", handleROI)

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("Dashboard: http://localhost%s\n", addr)
	return http.ListenAndServe(addr, mux)
}

func openDB() (*store.Store, error) {
	dbPath, err := config.DBPath()
	if err != nil {
		return nil, err
	}
	return store.Open(dbPath)
}

func parseTimeRange(r *http.Request) (int64, int64) {
	now := time.Now()
	untilMs := now.UnixMilli()

	since := r.URL.Query().Get("since")
	if since != "" {
		t, err := time.ParseInLocation("2006-01-02", since, time.Local)
		if err == nil {
			return t.UnixMilli(), untilMs
		}
	}

	period := r.URL.Query().Get("period")
	switch period {
	case "month":
		return now.AddDate(0, 0, -30).UnixMilli(), untilMs
	case "week":
		return now.AddDate(0, 0, -7).UnixMilli(), untilMs
	default:
		return now.AddDate(0, 0, -7).UnixMilli(), untilMs
	}
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// viewportData stores the latest viewport report from the browser.
var viewportData []byte

// POST /api/viewport — browser reports its viewport dimensions
// GET  /api/viewport — returns the latest reported viewport
func handleViewport(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		body, _ := io.ReadAll(r.Body)
		viewportData = body
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if viewportData == nil {
		w.Write([]byte(`{"error":"no viewport data yet, open the dashboard in a browser first"}`))
		return
	}
	w.Write(viewportData)
}

// GET /api/summary?period=week|month&since=2026-03-01
func handleSummary(w http.ResponseWriter, r *http.Request) {
	s, err := openDB()
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	defer s.Close()

	sinceMs, untilMs := parseTimeRange(r)
	sum, err := s.QuerySummary(sinceMs, untilMs)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}

	// Add cost
	byModel, _ := s.QueryTokensByModel(sinceMs, untilMs)
	cost := &analysis.CostBreakdown{}
	for _, mt := range byModel {
		pricing := analysis.LookupPricing(mt.Model)
		c := analysis.CalculateCost(mt.TotalInputTokens, mt.TotalOutputTokens,
			mt.TotalCacheReadTokens, mt.TotalCacheCreationTokens, pricing)
		cost.InputCost += c.InputCost
		cost.OutputCost += c.OutputCost
		cost.CacheReadCost += c.CacheReadCost
		cost.CacheCreationCost += c.CacheCreationCost
		cost.TotalCost += c.TotalCost
	}

	writeJSON(w, map[string]interface{}{
		"summary": sum,
		"cost":    cost,
	})
}

// GET /api/trend?period=week|month&since=2026-03-01
func handleTrend(w http.ResponseWriter, r *http.Request) {
	s, err := openDB()
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	defer s.Close()

	sinceMs, untilMs := parseTimeRange(r)
	daily, err := s.QueryDailyStats(sinceMs, untilMs)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}

	// Add cost per day
	type dayWithCost struct {
		store.DailyStats
		Cost float64 `json:"cost"`
	}
	var result []dayWithCost
	for _, d := range daily {
		t, _ := time.ParseInLocation("2006-01-02", d.Date, time.Local)
		daySince := t.UnixMilli()
		dayUntil := t.Add(24 * time.Hour).UnixMilli()
		byModel, _ := s.QueryTokensByModel(daySince, dayUntil)
		var dayCost float64
		for _, mt := range byModel {
			pricing := analysis.LookupPricing(mt.Model)
			c := analysis.CalculateCost(mt.TotalInputTokens, mt.TotalOutputTokens,
				mt.TotalCacheReadTokens, mt.TotalCacheCreationTokens, pricing)
			dayCost += c.TotalCost
		}
		result = append(result, dayWithCost{DailyStats: d, Cost: dayCost})
	}

	writeJSON(w, result)
}

// GET /api/sessions?limit=20
func handleSessions(w http.ResponseWriter, r *http.Request) {
	s, err := openDB()
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	defer s.Close()

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	sessions, err := s.ListSessions(limit)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}

	// Add cost per session
	type sessionWithCost struct {
		store.SessionRow
		Cost float64 `json:"cost"`
	}
	var result []sessionWithCost
	for _, sess := range sessions {
		pricing := analysis.LookupPricing(sess.ModelStr)
		c := analysis.CalculateCost(sess.TotalInputTokens, sess.TotalOutputTokens,
			sess.TotalCacheReadTokens, sess.TotalCacheCreationTokens, pricing)
		result = append(result, sessionWithCost{SessionRow: sess, Cost: c.TotalCost})
	}

	writeJSON(w, result)
}

// GET /api/session/{id}
func handleSessionDetail(w http.ResponseWriter, r *http.Request) {
	s, err := openDB()
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	defer s.Close()

	id := r.URL.Path[len("/api/session/"):]
	if id == "" {
		writeError(w, 400, "session id required")
		return
	}

	tl, err := s.GetSessionTimeline(id)
	if err != nil {
		writeError(w, 404, err.Error())
		return
	}

	writeJSON(w, tl)
}

// GET /api/waste?period=week
func handleWaste(w http.ResponseWriter, r *http.Request) {
	s, err := openDB()
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	defer s.Close()

	ids, err := s.GetRecentSessionIDs(10)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}

	report, err := analysis.AnalyzeWaste(s, ids)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}

	writeJSON(w, report)
}

// GET /api/roi?period=week&since=2026-03-01
func handleROI(w http.ResponseWriter, r *http.Request) {
	s, err := openDB()
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	defer s.Close()

	sinceMs, untilMs := parseTimeRange(r)
	report, err := analysis.AnalyzeROI(s, sinceMs, untilMs, "")
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}

	writeJSON(w, report)
}
