package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
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

	mux.Handle("/", http.FileServer(http.Dir("frontend/dist")))

	mux.HandleFunc("GET /api/today", handleToday)
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
	log.Fatal(http.ListenAndServe(addr, mux))
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
