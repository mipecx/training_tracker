# training_tracker

A calisthenics workout tracker: program planner, set log, and exercise progress tracking.

## Features

- Ready-made programs by day (push / pull / legs / full) in `workouts/*.json`
- Workout sessions: create, finish, set log
- History and progress per exercise (charts)
- Body metrics (weight, measurements)
- Analytics: session volume, workout comparison
- AI review of training history (optional, via Cloudflare Worker)

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

The server starts on the port from `GYMTRACKER_ADDR` (default `:8080`). If a
sibling `plan-app` checkout exists, it is served at `/` and this app at `/gym/`
(see [Serve plan-app alongside](#serve-plan-app-alongside)).

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
| GET | `/api/plan/training` | Compact snapshot for plan-app (CORS enabled) |

## Serve plan-app alongside

The server can also serve a sibling `plan-app` checkout as the main page:

```bash
GYMTRACKER_ADDR=:8000 \
PLAN_APP_DIR=../plan-app \
PLAN_TRAINING_JSON=../plan-app/data/training.json \
go run main.go
```

- `http://localhost:8000/` — plan-app (main page)
- `http://localhost:8000/gym/` — this app
- `GET /api/plan/training` — compact training snapshot for plan-app

Old `/plan/` links redirect to `/`. Using port `8000` keeps plan-app on the same
origin as before, so its localStorage is preserved. `PLAN_TRAINING_JSON` is optional: when set, the
snapshot file is rewritten after a session, case metric or workout program change.
API responses under `/api/` send CORS headers, so plan-app on Cloudflare Pages
(via an https tunnel) can read and edit workouts with `GET/PUT /api/workout/{day}`.
The plan-app repo ships `scripts/host.sh`, which starts this server on `:8000`,
opens a `cloudflared` quick tunnel and stores its address in a Cloudflare Worker
KV. Plan-app on Pages then talks to the stable proxy
`https://plan-tunnel.mipec404.workers.dev` instead of the changing quick-tunnel
hostname.

For plan-app on GitHub Pages over https, live data needs an https tunnel to this
server (for example `cloudflared tunnel --url http://localhost:8000`). Put the
tunnel URL into plan-app's **Settings → Training**.

## AI review

The History screen has an **AI review** card. It sends the last 8 finished
sessions, exercise summaries and metrics to the same Cloudflare Worker as
`plan-app` and shows a Coach-style JSON review.

Set the Worker URL/model/token in the card's **AI settings** block. They are
stored in `localStorage`:

- `gym_ai_worker` — default `https://plan-llm-proxy.mipec404.workers.dev`
- `gym_ai_model` — default `deepseek-chat`
- `gym_ai_token` — optional `x-app-token`

After changing the Svelte source, rebuild the served UI:

```bash
cd frontend && npm run build && cd ..
go run main.go
```
