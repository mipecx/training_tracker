# training_tracker

A calisthenics workout tracker: program planner, set log, and exercise progress tracking.

## Features

- Ready-made programs by day (push / pull / legs / full) in `workouts/*.json`
- Workout sessions: create, finish, set log
- History and progress per exercise (charts)
- Body metrics (weight, measurements)
- Analytics: session volume, workout comparison

## Tech stack

| Layer | Technologies |
|---|---|
| Backend | Go, SQLite (pure-Go `modernc.org/sqlite`) |
| Frontend | Svelte + TypeScript + Vite, Chart.js |
| Legacy UI | `static/` — simple HTML/JS version |

## Structure

```
training_tracker/
├── main.go            # HTTP server and REST API
├── db/db.go           # SQLite schema and queries
├── workouts/          # Workout programs (JSON)
├── frontend/          # Svelte app (Vite)
├── static/            # Simple HTML/JS version of the UI
└── data/              # Local SQLite database (not committed)
```

## Running

### Backend

```bash
go run main.go
```

The server starts on the port from `HTTP_ADDR` (default `:8080`).

### Frontend (dev)

```bash
cd frontend
npm install
npm run dev
```

### Production build

```bash
cd frontend && npm run build
cd ..
go run main.go   # serves frontend/dist
```

## API (main endpoints)

| Method | Path | Description |
|---|---|---|
| GET | `/api/today` | Today's program |
| GET/PUT | `/api/workout/{day}` | Get / save the day's program |
| POST | `/api/sessions` | Create a workout session |
| GET | `/api/sessions` | List sessions |
| GET | `/api/sessions/{id}` | Session details |
| POST | `/api/sessions/{id}/finish` | Finish a session |
| POST | `/api/sessions/{id}/sets` | Log a set |
| GET | `/api/history/{exercise_id}` | Exercise history |
| GET | `/api/progress` | Summary across all exercises |
| GET/POST | `/api/metrics` | Body metrics |
| GET | `/api/sessions/{id}/volume` | Session volume |
| GET | `/api/sessions/{id}/compare` | Compare with the previous workout |
