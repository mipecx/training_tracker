package db

import (
	"database/sql"
	"log"
	_ "modernc.org/sqlite"
	"strings"
	"time"
)

const schema = `
CREATE TABLE IF NOT EXISTS sessions (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    workout_id   TEXT    NOT NULL,
    workout_name TEXT    NOT NULL,
    date         TEXT    NOT NULL,
    started_at   TEXT    NOT NULL,
    finished_at  TEXT,
    duration_sec INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS sets_log (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id   INTEGER NOT NULL REFERENCES sessions(id),
    exercise_id  TEXT    NOT NULL,
    exercise_name TEXT   NOT NULL,
    set_num      INTEGER NOT NULL,
    side         TEXT,
    reps         INTEGER,
    duration_sec INTEGER,
    notes        TEXT,
    logged_at    TEXT    NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sets_session ON sets_log(session_id);
CREATE INDEX IF NOT EXISTS idx_sets_exercise ON sets_log(exercise_id);

CREATE TABLE IF NOT EXISTS weekly_metrics (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    week_date    TEXT    NOT NULL UNIQUE,
    weight_kg    REAL,
    height_cm    REAL,
    waist_cm     REAL,
    neck_cm      REAL,
    notes        TEXT,
    created_at   TEXT    NOT NULL
);
`

type DB struct {
	conn *sql.DB
}

func Open(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, err
	}
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(schema); err != nil {
		return nil, err
	}
	if err := migrate(conn); err != nil {
		return nil, err
	}
	log.Printf("db: opened %s", path)
	return &DB{conn: conn}, nil
}

func migrate(conn *sql.DB) error {
	type colDef struct {
		table string
		name  string
		typ   string
	}
	want := []colDef{
		{"sets_log", "duration_sec", "INTEGER"},
		{"weekly_metrics", "height_cm", "REAL"},
		{"weekly_metrics", "waist_cm", "REAL"},
		{"weekly_metrics", "neck_cm", "REAL"},
	}
	seen := map[string]bool{}
	for _, c := range want {
		if seen[c.table+"."+c.name] {
			continue
		}
		seen[c.table+"."+c.name] = true

		has, err := hasColumn(conn, c.table, c.name)
		if err != nil {
			return err
		}
		if !has {
			if _, err := conn.Exec(`ALTER TABLE ` + c.table + ` ADD COLUMN ` + c.name + ` ` + c.typ); err != nil {
				return err
			}
			log.Printf("db: migrated %s +%s", c.table, c.name)
		}
	}
	return nil
}

func hasColumn(conn *sql.DB, table, column string) (bool, error) {
	rows, err := conn.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notnull, pk int
		var name, ctype string
		var dflt any
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

func (d *DB) Close() { d.conn.Close() }

type Session struct {
	ID          int64
	WorkoutID   string
	WorkoutName string
	Date        string
	StartedAt   time.Time
	FinishedAt  *time.Time
	DurationSec int
}

func (d *DB) CreateSession(workoutID, workoutName string) (int64, error) {
	now := time.Now()
	res, err := d.conn.Exec(`
		INSERT INTO sessions (workout_id, workout_name, date, started_at)
		VALUES (?, ?, ?, ?)`,
		workoutID,
		workoutName,
		now.Format("2006-01-02"),
		now.Format(time.RFC3339),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) FinishSession(sessionID int64, durationSec int) error {
	_, err := d.conn.Exec(`
		UPDATE sessions SET finished_at = ?, duration_sec = ? WHERE id = ?`,
		time.Now().Format(time.RFC3339),
		durationSec,
		sessionID,
	)
	return err
}

func (d *DB) GetSession(id int64) (*Session, error) {
	row := d.conn.QueryRow(`SELECT id, workout_id, workout_name, date, started_at, finished_at, duration_sec FROM sessions WHERE id = ?`, id)
	return scanSession(row)
}

func (d *DB) ListSessions(limit int) ([]Session, error) {
	rows, err := d.conn.Query(`
		SELECT id, workout_id, workout_name, date, started_at, finished_at, duration_sec
		FROM sessions ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Session
	for rows.Next() {
		s, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanSession(r scanner) (*Session, error) {
	var s Session
	var startedStr, finishedStr sql.NullString
	if err := r.Scan(&s.ID, &s.WorkoutID, &s.WorkoutName, &s.Date, &startedStr, &finishedStr, &s.DurationSec); err != nil {
		return nil, err
	}
	if startedStr.Valid {
		t, _ := time.Parse(time.RFC3339, startedStr.String)
		s.StartedAt = t
	}
	if finishedStr.Valid {
		t, _ := time.Parse(time.RFC3339, finishedStr.String)
		s.FinishedAt = &t
	}
	return &s, nil
}

type SetEntry struct {
	ID           int64
	SessionID    int64
	ExerciseID   string
	ExerciseName string
	SetNum       int
	Side         *string
	Reps         *int
	DurationSec  *int
	Notes        string
	LoggedAt     time.Time
}

func (d *DB) LogSet(sessionID int64, exID, exName string, setNum int, side *string, reps *int, durationSec *int, notes string) error {
	_, err := d.conn.Exec(`
		INSERT INTO sets_log (session_id, exercise_id, exercise_name, set_num, side, reps, duration_sec, notes, logged_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sessionID, exID, exName, setNum, side, reps, durationSec, notes,
		time.Now().Format(time.RFC3339),
	)
	return err
}

func (d *DB) MigrateTimedSets(timedIDs []string) error {
	if len(timedIDs) == 0 {
		return nil
	}
	ph := strings.Repeat("?,", len(timedIDs))
	ph = ph[:len(ph)-1]
	args := make([]any, 0, len(timedIDs))
	for _, id := range timedIDs {
		args = append(args, id)
	}
	_, err := d.conn.Exec(`
		UPDATE sets_log SET duration_sec = reps, reps = NULL
		WHERE exercise_id IN (`+ph+`) AND duration_sec IS NULL`, args...)
	if err == nil {
		log.Printf("db: migrated %d timed exercises to duration_sec", len(timedIDs))
	}
	return err
}

func (d *DB) GetSessionSets(sessionID int64) ([]SetEntry, error) {
	rows, err := d.conn.Query(`
		SELECT id, session_id, exercise_id, exercise_name, set_num, side, reps, duration_sec, notes, logged_at
		FROM sets_log WHERE session_id = ? ORDER BY id`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SetEntry
	for rows.Next() {
		var e SetEntry
		var side sql.NullString
		var reps sql.NullInt64
		var dur sql.NullInt64
		var loggedStr string
		if err := rows.Scan(&e.ID, &e.SessionID, &e.ExerciseID, &e.ExerciseName,
			&e.SetNum, &side, &reps, &dur, &e.Notes, &loggedStr); err != nil {
			return nil, err
		}
		if side.Valid {
			e.Side = &side.String
		}
		if reps.Valid {
			r := int(reps.Int64)
			e.Reps = &r
		}
		if dur.Valid {
			d := int(dur.Int64)
			e.DurationSec = &d
		}
		e.LoggedAt, _ = time.Parse(time.RFC3339, loggedStr)
		out = append(out, e)
	}
	return out, rows.Err()
}

func (d *DB) ExerciseHistory(exerciseID string, limit int) ([]SetEntry, error) {
	rows, err := d.conn.Query(`
		SELECT sl.id, sl.session_id, sl.exercise_id, sl.exercise_name,
		       sl.set_num, sl.side, sl.reps, sl.duration_sec, sl.notes, sl.logged_at
		FROM sets_log sl
		JOIN sessions s ON s.id = sl.session_id
		WHERE sl.exercise_id = ?
		ORDER BY s.date DESC, sl.set_num ASC
		LIMIT ?`, exerciseID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SetEntry
	for rows.Next() {
		var e SetEntry
		var side sql.NullString
		var reps sql.NullInt64
		var dur sql.NullInt64
		var loggedStr string
		if err := rows.Scan(&e.ID, &e.SessionID, &e.ExerciseID, &e.ExerciseName,
			&e.SetNum, &side, &reps, &dur, &e.Notes, &loggedStr); err != nil {
			return nil, err
		}
		if side.Valid {
			e.Side = &side.String
		}
		if reps.Valid {
			r := int(reps.Int64)
			e.Reps = &r
		}
		if dur.Valid {
			d := int(dur.Int64)
			e.DurationSec = &d
		}
		e.LoggedAt, _ = time.Parse(time.RFC3339, loggedStr)
		out = append(out, e)
	}
	return out, rows.Err()
}

type WeeklyMetric struct {
	ID        int64    `json:"id"`
	WeekDate  string   `json:"week_date"`
	WeightKg  *float64 `json:"weight_kg"`
	HeightCm  *float64 `json:"height_cm"`
	WaistCm   *float64 `json:"waist_cm"`
	NeckCm    *float64 `json:"neck_cm"`
	Notes     string   `json:"notes"`
	CreatedAt string   `json:"created_at"`
}

func (d *DB) SaveWeeklyMetric(m WeeklyMetric) error {
	_, err := d.conn.Exec(`
		INSERT INTO weekly_metrics (week_date, weight_kg, height_cm, waist_cm, neck_cm, notes, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(week_date) DO UPDATE SET
			weight_kg = excluded.weight_kg,
			height_cm = excluded.height_cm,
			waist_cm = excluded.waist_cm,
			neck_cm = excluded.neck_cm,
			notes = excluded.notes,
			created_at = excluded.created_at`,
		m.WeekDate, m.WeightKg, m.HeightCm, m.WaistCm, m.NeckCm, m.Notes,
		time.Now().Format(time.RFC3339),
	)
	return err
}

func (d *DB) ListWeeklyMetrics(limit int) ([]WeeklyMetric, error) {
	rows, err := d.conn.Query(`
		SELECT id, week_date, weight_kg, height_cm, waist_cm, neck_cm, notes, created_at
		FROM weekly_metrics ORDER BY week_date ASC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []WeeklyMetric
	for rows.Next() {
		var m WeeklyMetric
		var wt, ht, ws, nk sql.NullFloat64
		if err := rows.Scan(&m.ID, &m.WeekDate, &wt, &ht, &ws, &nk, &m.Notes, &m.CreatedAt); err != nil {
			return nil, err
		}
		if wt.Valid {
			v := wt.Float64
			m.WeightKg = &v
		}
		if ht.Valid {
			v := ht.Float64
			m.HeightCm = &v
		}
		if ws.Valid {
			v := ws.Float64
			m.WaistCm = &v
		}
		if nk.Valid {
			v := nk.Float64
			m.NeckCm = &v
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (d *DB) ExerciseProgress(exerciseID string) ([]map[string]interface{}, error) {
	rows, err := d.conn.Query(`
		SELECT s.date,
		       MAX(sl.reps) as max_reps, AVG(sl.reps) as avg_reps, COUNT(*) as sets_done,
		       MAX(sl.duration_sec) as max_sec, SUM(sl.duration_sec) as total_sec
		FROM sets_log sl
		JOIN sessions s ON s.id = sl.session_id
		WHERE sl.exercise_id = ? AND s.finished_at IS NOT NULL
		  AND (sl.reps IS NOT NULL OR sl.duration_sec IS NOT NULL)
		GROUP BY s.date
		ORDER BY s.date ASC`, exerciseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []map[string]interface{}
	for rows.Next() {
		var date string
		var maxR, sets, maxSec, totalSec sql.NullInt64
		var avgR sql.NullFloat64
		if err := rows.Scan(&date, &maxR, &avgR, &sets, &maxSec, &totalSec); err != nil {
			return nil, err
		}
		out = append(out, map[string]interface{}{
			"date":      date,
			"max_reps":  orZero(maxR),
			"avg_reps":  avgR.Float64,
			"sets":      int(sets.Int64),
			"max_sec":   orZero(maxSec),
			"total_sec": orZero(totalSec),
		})
	}
	return out, rows.Err()
}

func orZero(v sql.NullInt64) int {
	if !v.Valid {
		return 0
	}
	return int(v.Int64)
}

func (d *DB) AllExerciseSummary() ([]map[string]interface{}, error) {
	rows, err := d.conn.Query(`
		SELECT sl.exercise_id, sl.exercise_name,
		       MAX(sl.reps) as all_time_max,
		       MAX(sl.duration_sec) as all_time_max_sec,
		       COUNT(DISTINCT s.id) as sessions_count,
		       MAX(s.date) as last_date,
		       CASE WHEN MAX(sl.duration_sec) IS NOT NULL THEN 'sec' ELSE 'reps' END as unit
		FROM sets_log sl
		JOIN sessions s ON s.id = sl.session_id
		WHERE s.finished_at IS NOT NULL AND (sl.reps IS NOT NULL OR sl.duration_sec IS NOT NULL)
		GROUP BY sl.exercise_id
		ORDER BY last_date DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []map[string]interface{}
	for rows.Next() {
		var exID, exName, lastDate, unit string
		var allTimeMax, allTimeMaxSec, sessionsCount sql.NullInt64
		if err := rows.Scan(&exID, &exName, &allTimeMax, &allTimeMaxSec, &sessionsCount, &lastDate, &unit); err != nil {
			return nil, err
		}
		out = append(out, map[string]interface{}{
			"exercise_id": exID, "exercise_name": exName,
			"all_time_max":     orZero(allTimeMax),
			"all_time_max_sec": orZero(allTimeMaxSec),
			"sessions_count":   int(sessionsCount.Int64), "last_date": lastDate, "unit": unit,
		})
	}
	return out, rows.Err()
}

func (d *DB) SessionVolume(sessionID int64) ([]map[string]interface{}, error) {
	rows, err := d.conn.Query(`
		SELECT exercise_id, exercise_name,
		       COUNT(DISTINCT set_num) as sets_done,
		       COALESCE(SUM(reps), 0) as total_reps,
		       COALESCE(MAX(reps), 0) as max_reps,
		       COALESCE(AVG(reps), 0) as avg_reps,
		       COALESCE(SUM(duration_sec), 0) as total_sec,
		       COALESCE(MAX(duration_sec), 0) as max_sec,
		       CASE WHEN MAX(duration_sec) IS NOT NULL THEN 'sec' ELSE 'reps' END as unit
		FROM (
			SELECT exercise_id, exercise_name, set_num, side,
			       MAX(reps) as reps, MAX(duration_sec) as duration_sec
			FROM sets_log WHERE session_id = ?
			GROUP BY exercise_id, set_num, side
		)
		GROUP BY exercise_id`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var exID, exName, unit string
		var sets, totalReps, maxReps, totalSec, maxSec int
		var avgReps float64
		if err := rows.Scan(&exID, &exName, &sets, &totalReps, &maxReps, &avgReps, &totalSec, &maxSec, &unit); err != nil {
			return nil, err
		}
		out = append(out, map[string]interface{}{
			"exercise_id": exID, "exercise_name": exName,
			"sets": sets, "total_reps": totalReps,
			"max_reps": maxReps, "avg_reps": avgReps,
			"total_sec": totalSec, "max_sec": maxSec, "unit": unit,
			"volume": volumeValue(sets, totalReps, totalSec, unit),
		})
	}
	return out, rows.Err()
}

func volumeValue(sets, totalReps, totalSec int, unit string) int {
	if unit == "sec" {
		return totalSec
	}
	if sets == 0 {
		return 0
	}
	return totalReps
}

func (d *DB) PrevSessionOfType(workoutID string, beforeSessionID int64) (*Session, error) {
	row := d.conn.QueryRow(`
		SELECT id, workout_id, workout_name, date, started_at, finished_at, duration_sec
		FROM sessions
		WHERE workout_id = ? AND id < ? AND finished_at IS NOT NULL
		ORDER BY id DESC LIMIT 1`, workoutID, beforeSessionID)
	return scanSession(row)
}
