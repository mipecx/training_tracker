package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mipecx/training_tracker/db"
)

type Technique struct {
	Text  string `json:"text"`
	Image string `json:"image"`
}

type ProgressionLevel struct {
	Level       int    `json:"level"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Exercise struct {
	ID                  string             `json:"id"`
	Name                string             `json:"name"`
	Sets                int                `json:"sets"`
	RepsTarget          string             `json:"reps_target"`
	RestSec             int                `json:"rest_sec"`
	Notes               string             `json:"notes"`
	PerSide             bool               `json:"per_side,omitempty"`
	IsTimed             bool               `json:"is_timed,omitempty"`
	DifficultyLevel     int                `json:"difficulty_level,omitempty"`
	DifficultyMax       int                `json:"difficulty_max,omitempty"`
	DifficultyThreshold *int               `json:"difficulty_threshold,omitempty"`
	Progression         []ProgressionLevel `json:"progression,omitempty"`
	Technique           Technique          `json:"technique"`
}

type Workout struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Day       string     `json:"day"`
	Exercises []Exercise `json:"exercises"`
}

var workouts = map[string]Workout{}
var database *db.DB

var workoutFiles = map[string]string{
	"monday":    "workouts/monday_push.json",
	"wednesday": "workouts/wednesday_pull.json",
	"friday":    "workouts/friday_full.json",
	"sunday":    "workouts/sunday_legs_core.json",
}

var dayMap = map[time.Weekday]string{
	time.Monday:    "monday",
	time.Wednesday: "wednesday",
	time.Friday:    "friday",
	time.Sunday:    "sunday",
}

func loadWorkouts() {
	files := []string{
		"workouts/monday_push.json",
		"workouts/wednesday_pull.json",
		"workouts/friday_full.json",
		"workouts/sunday_legs_core.json",
	}
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			log.Fatalf("cannot read %s: %v", path, err)
		}
		var w Workout
		if err := json.Unmarshal(data, &w); err != nil {
			log.Fatalf("cannot parse %s: %v", path, err)
		}
		workouts[w.Day] = w
		log.Printf("loaded: %s (%d exercises)", w.Name, len(w.Exercises))
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func errJSON(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func handleToday(w http.ResponseWriter, r *http.Request) {
	day := dayMap[time.Now().Weekday()]
	workout, ok := workouts[day]
	if !ok {
		writeJSON(w, http.StatusOK, map[string]string{"message": "rest day"})
		return
	}
	writeJSON(w, http.StatusOK, workout)
}

func handleWorkoutByDay(w http.ResponseWriter, r *http.Request) {
	day := r.PathValue("day")
	workout, ok := workouts[day]
	if !ok {
		errJSON(w, http.StatusNotFound, "no workout for: "+day)
		return
	}
	writeJSON(w, http.StatusOK, workout)
}

func handleSaveWorkout(w http.ResponseWriter, r *http.Request) {
	day := r.PathValue("day")
	path, ok := workoutFiles[day]
	if !ok {
		errJSON(w, http.StatusNotFound, "no workout for: "+day)
		return
	}
	var wk Workout
	if err := json.NewDecoder(r.Body).Decode(&wk); err != nil {
		errJSON(w, http.StatusBadRequest, "invalid json")
		return
	}
	wk.Day = day
	data, err := json.MarshalIndent(wk, "", "  ")
	if err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	workouts[day] = wk
	maybeWritePlanSnapshot()
	log.Printf("saved: %s (%d exercises)", wk.Name, len(wk.Exercises))
	writeJSON(w, http.StatusOK, wk)
}

func handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var body struct {
		WorkoutID   string `json:"workout_id"`
		WorkoutName string `json:"workout_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errJSON(w, http.StatusBadRequest, "invalid json")
		return
	}
	if body.WorkoutID == "" {
		errJSON(w, http.StatusBadRequest, "workout_id required")
		return
	}
	id, err := database.CreateSession(body.WorkoutID, body.WorkoutName)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int64{"session_id": id})
}

func handleFinishSession(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		errJSON(w, http.StatusBadRequest, "invalid session id")
		return
	}
	var body struct {
		DurationSec int `json:"duration_sec"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if err := database.FinishSession(id, body.DurationSec); err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	maybeWritePlanSnapshot()
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleLogSet(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		errJSON(w, http.StatusBadRequest, "invalid session id")
		return
	}
	var body struct {
		ExerciseID   string  `json:"exercise_id"`
		ExerciseName string  `json:"exercise_name"`
		SetNum       int     `json:"set_num"`
		Side         *string `json:"side"`
		Reps         *int    `json:"reps"`
		DurationSec  *int    `json:"duration_sec"`
		Notes        string  `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errJSON(w, http.StatusBadRequest, "invalid json")
		return
	}
	if body.Reps == nil && body.DurationSec == nil {
		errJSON(w, http.StatusBadRequest, "reps or duration_sec required")
		return
	}
	if err := database.LogSet(id, body.ExerciseID, body.ExerciseName, body.SetNum, body.Side, body.Reps, body.DurationSec, body.Notes); err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
}

func handleListSessions(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	sessions, err := database.ListSessions(limit)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	if sessions == nil {
		sessions = []db.Session{}
	}
	writeJSON(w, http.StatusOK, sessions)
}

func handleGetSession(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		errJSON(w, http.StatusBadRequest, "invalid session id")
		return
	}
	session, err := database.GetSession(id)
	if err != nil {
		errJSON(w, http.StatusNotFound, "session not found")
		return
	}
	sets, err := database.GetSessionSets(id)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	if sets == nil {
		sets = []db.SetEntry{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"session": session, "sets": sets})
}

func handleExerciseHistory(w http.ResponseWriter, r *http.Request) {
	exID := r.PathValue("exercise_id")
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	entries, err := database.ExerciseHistory(exID, limit)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	if entries == nil {
		entries = []db.SetEntry{}
	}
	writeJSON(w, http.StatusOK, entries)
}

func handleAllExerciseSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := database.AllExerciseSummary()
	if err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	if summary == nil {
		summary = []map[string]interface{}{}
	}
	writeJSON(w, http.StatusOK, summary)
}

func handleExerciseProgress(w http.ResponseWriter, r *http.Request) {
	exID := r.PathValue("exercise_id")
	data, err := database.ExerciseProgress(exID)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	if data == nil {
		data = []map[string]interface{}{}
	}
	writeJSON(w, http.StatusOK, data)
}

func handleListMetrics(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	metrics, err := database.ListWeeklyMetrics(limit)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	if metrics == nil {
		metrics = []db.WeeklyMetric{}
	}
	writeJSON(w, http.StatusOK, metrics)
}

func handleSaveMetric(w http.ResponseWriter, r *http.Request) {
	var m db.WeeklyMetric
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		errJSON(w, http.StatusBadRequest, "invalid json")
		return
	}
	if m.WeekDate == "" {
		errJSON(w, http.StatusBadRequest, "week_date required")
		return
	}
	if err := database.SaveWeeklyMetric(m); err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	maybeWritePlanSnapshot()
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func migrateTimedSets() {
	var timedIDs []string
	for _, w := range workouts {
		for _, ex := range w.Exercises {
			if ex.IsTimed {
				timedIDs = append(timedIDs, ex.ID)
			}
		}
	}
	if err := database.MigrateTimedSets(timedIDs); err != nil {
		log.Printf("migrate timed sets: %v", err)
	}
}

func main() {
	loadWorkouts()

	if err := os.MkdirAll("data", 0755); err != nil {
		log.Fatal(err)
	}
	var err error
	database, err = db.Open("data/gymtracker.db")
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer database.Close()

	migrateTimedSets()

	mux := http.NewServeMux()

	planAppDir := os.Getenv("PLAN_APP_DIR")
	if planAppDir == "" {
		planAppDir = "../plan-app"
	}

	planAvailable := false
	if info, err := os.Stat(planAppDir); err == nil && info.IsDir() {
		planAvailable = true
	}

	if planAvailable {
		mux.Handle("/", http.FileServer(http.Dir(planAppDir)))
		redirectRoot := func(w http.ResponseWriter, r *http.Request) {
			target := "/"
			if r.URL.RawQuery != "" {
				target += "?" + r.URL.RawQuery
			}
			http.Redirect(w, r, target, http.StatusFound)
		}
		mux.HandleFunc("/plan", redirectRoot)
		mux.HandleFunc("/plan/", redirectRoot)
		log.Printf("plan-app → / (%s)", planAppDir)
	} else {
		mux.Handle("/", http.FileServer(http.Dir("frontend/dist")))
		log.Printf("plan-app: %s not found, serving training_tracker at /", planAppDir)
	}

	mux.Handle("/gym/", http.StripPrefix("/gym/", http.FileServer(http.Dir("frontend/dist"))))
	mux.HandleFunc("/gym", func(w http.ResponseWriter, r *http.Request) {
		target := "/gym/"
		if r.URL.RawQuery != "" {
			target += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, target, http.StatusFound)
	})
	log.Printf("training_tracker → /gym/")

	mux.HandleFunc("GET /api/today", handleToday)
	mux.HandleFunc("/api/plan/training", handlePlanTraining)
	mux.HandleFunc("GET /api/workout/{day}", handleWorkoutByDay)
	mux.HandleFunc("PUT /api/workout/{day}", handleSaveWorkout)

	mux.HandleFunc("POST /api/sessions", handleCreateSession)
	mux.HandleFunc("GET /api/sessions", handleListSessions)
	mux.HandleFunc("GET /api/sessions/{id}", handleGetSession)
	mux.HandleFunc("POST /api/sessions/{id}/finish", handleFinishSession)
	mux.HandleFunc("POST /api/sessions/{id}/sets", handleLogSet)

	mux.HandleFunc("GET /api/history/{exercise_id}", handleExerciseHistory)

	mux.HandleFunc("GET /api/progress", handleAllExerciseSummary)
	mux.HandleFunc("GET /api/progress/{exercise_id}", handleExerciseProgress)
	mux.HandleFunc("GET /api/metrics", handleListMetrics)
	mux.HandleFunc("POST /api/metrics", handleSaveMetric)

	mux.HandleFunc("GET /api/sessions/{id}/volume", handleSessionVolume)
	mux.HandleFunc("GET /api/sessions/{id}/compare", handleSessionCompare)

	addr := ":8080"
	if v := os.Getenv("GYMTRACKER_ADDR"); v != "" {
		addr = v
	}
	log.Printf("gymtracker → http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, withCORS(mux)))
}

// withCORS allows plan-app on GitHub Pages (possibly through an https tunnel)
// to read and edit workouts on this server.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "content-type")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func handleSessionVolume(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		errJSON(w, http.StatusBadRequest, "invalid session id")
		return
	}
	vol, err := database.SessionVolume(id)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	if vol == nil {
		vol = []map[string]interface{}{}
	}
	writeJSON(w, http.StatusOK, vol)
}

func handleSessionCompare(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		errJSON(w, http.StatusBadRequest, "invalid session id")
		return
	}
	session, err := database.GetSession(id)
	if err != nil {
		errJSON(w, http.StatusNotFound, "session not found")
		return
	}
	currVol, err := database.SessionVolume(id)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	if currVol == nil {
		currVol = []map[string]interface{}{}
	}

	prev, _ := database.PrevSessionOfType(session.WorkoutID, id)
	var prevVol []map[string]interface{}
	if prev != nil {
		prevVol, _ = database.SessionVolume(prev.ID)
	}
	if prevVol == nil {
		prevVol = []map[string]interface{}{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"current":  map[string]any{"session": session, "volume": currVol},
		"previous": map[string]any{"session": prev, "volume": prevVol},
	})
}

type planSnapshot struct {
	Version    int               `json:"version"`
	Source     string            `json:"source"`
	ExportedAt string            `json:"exportedAt"`
	Programs   []map[string]any  `json:"programs"`
	Sessions   []map[string]any  `json:"sessions"`
	Metrics    []db.WeeklyMetric `json:"metrics"`
}

func handlePlanTraining(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "content-type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		errJSON(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	snap, err := buildPlanSnapshot()
	if err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, snap)
}

func buildPlanSnapshot() (*planSnapshot, error) {
	sessions, err := database.ListSessions(1000)
	if err != nil {
		return nil, err
	}
	sort.Slice(sessions, func(i, j int) bool {
		if sessions[i].Date == sessions[j].Date {
			return sessions[i].ID < sessions[j].ID
		}
		return sessions[i].Date < sessions[j].Date
	})

	sessionRows := make([]map[string]any, 0, len(sessions))
	for _, s := range sessions {
		sets, err := database.GetSessionSets(s.ID)
		if err != nil {
			return nil, err
		}

		type setKey struct {
			exercise string
			num      int
			side     string
		}
		dedup := map[setKey]db.SetEntry{}
		for _, set := range sets {
			side := ""
			if set.Side != nil {
				side = *set.Side
			}
			dedup[setKey{set.ExerciseID, set.SetNum, side}] = set
		}

		ordered := make([]db.SetEntry, 0, len(dedup))
		for _, set := range dedup {
			ordered = append(ordered, set)
		}
		sort.Slice(ordered, func(i, j int) bool {
			a, b := ordered[i], ordered[j]
			if a.ExerciseID != b.ExerciseID {
				return a.ExerciseID < b.ExerciseID
			}
			if a.SetNum != b.SetNum {
				return a.SetNum < b.SetNum
			}
			as, bs := "", ""
			if a.Side != nil {
				as = *a.Side
			}
			if b.Side != nil {
				bs = *b.Side
			}
			return as < bs
		})

		setRows := make([]map[string]any, 0, len(ordered))
		for _, set := range ordered {
			setRows = append(setRows, map[string]any{
				"exerciseId":   set.ExerciseID,
				"exerciseName": set.ExerciseName,
				"setNum":       set.SetNum,
				"side":         set.Side,
				"reps":         set.Reps,
				"sec":          set.DurationSec,
			})
		}

		var finished any
		if s.FinishedAt != nil {
			finished = s.FinishedAt.Format(time.RFC3339)
		}
		started := ""
		if !s.StartedAt.IsZero() {
			started = s.StartedAt.Format(time.RFC3339)
		}

		sessionRows = append(sessionRows, map[string]any{
			"id":          s.ID,
			"workoutId":   s.WorkoutID,
			"workoutName": s.WorkoutName,
			"date":        s.Date,
			"startedAt":   started,
			"finishedAt":  finished,
			"durationSec": s.DurationSec,
			"sets":        setRows,
		})
	}

	metrics, err := database.ListWeeklyMetrics(1000)
	if err != nil {
		return nil, err
	}
	if metrics == nil {
		metrics = []db.WeeklyMetric{}
	}

	return &planSnapshot{
		Version:    1,
		Source:     "training_tracker",
		ExportedAt: time.Now().Format(time.RFC3339),
		Programs:   planPrograms(),
		Sessions:   sessionRows,
		Metrics:    metrics,
	}, nil
}

func planPrograms() []map[string]any {
	out := make([]map[string]any, 0, len(workouts))
	for _, day := range []string{"monday", "wednesday", "friday", "sunday"} {
		w, ok := workouts[day]
		if !ok {
			continue
		}
		exercises := make([]map[string]any, 0, len(w.Exercises))
		for _, ex := range w.Exercises {
			exercises = append(exercises, map[string]any{
				"id":                   ex.ID,
				"name":                 ex.Name,
				"sets":                 ex.Sets,
				"reps_target":          ex.RepsTarget,
				"rest_sec":             ex.RestSec,
				"per_side":             ex.PerSide,
				"is_timed":             ex.IsTimed,
				"difficulty_level":     ex.DifficultyLevel,
				"difficulty_max":       ex.DifficultyMax,
				"difficulty_threshold": ex.DifficultyThreshold,
			})
		}
		out = append(out, map[string]any{
			"id":        w.ID,
			"name":      w.Name,
			"day":       w.Day,
			"exercises": exercises,
		})
	}
	return out
}

func maybeWritePlanSnapshot() {
	path := os.Getenv("PLAN_TRAINING_JSON")
	if path == "" {
		return
	}
	if err := writePlanSnapshot(path); err != nil {
		log.Printf("plan snapshot: %v", err)
		return
	}
	log.Printf("plan snapshot: %s", path)
}

func writePlanSnapshot(path string) error {
	snap, err := buildPlanSnapshot()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, data, 0644)
}
