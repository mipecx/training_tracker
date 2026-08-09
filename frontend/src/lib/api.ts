import type {
	CompareResult,
	ExerciseProgressPoint,
	ExerciseSummary,
	Session,
	SetEntry,
	VolumeEntry,
	WeeklyMetric,
	Workout,
} from './types';

async function req<T>(url: string, init?: RequestInit): Promise<T> {
	const res = await fetch(url, init);
	if (!res.ok) {
		throw new Error(`HTTP ${res.status} ${url}`);
	}
	return res.json() as Promise<T>;
}

export const api = {
	today: () => req<Workout | { message: string }>('/api/today'),
	workoutByDay: (day: string) => req<Workout>(`/api/workout/${day}`),
	saveWorkout: (day: string, workout: Workout) =>
		req<Workout>(`/api/workout/${day}`, {
			method: 'PUT',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(workout),
		}),

	createSession: (workoutId: string, workoutName: string) =>
		req<{ session_id: number }>('/api/sessions', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ workout_id: workoutId, workout_name: workoutName }),
		}),

	finishSession: (id: number, durationSec: number) =>
		req<{ status: string }>(`/api/sessions/${id}/finish`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ duration_sec: durationSec }),
		}),

	logSet: (sessionId: number, body: {
		exercise_id: string;
		exercise_name: string;
		set_num: number;
		side?: string | null;
		reps?: number | null;
		duration_sec?: number | null;
		notes?: string;
	}) =>
		req<{ status: string }>(`/api/sessions/${sessionId}/sets`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(body),
		}),

	listSessions: (limit = 100) =>
		req<Session[]>(`/api/sessions?limit=${limit}`),

	getSession: (id: number) =>
		req<{ session: Session; sets: SetEntry[] }>(`/api/sessions/${id}`),

	compare: (id: number) =>
		req<CompareResult>(`/api/sessions/${id}/compare`),

	exerciseHistory: (exerciseId: string, limit = 10) =>
		req<SetEntry[]>(`/api/history/${exerciseId}?limit=${limit}`),

	allExerciseSummary: () =>
		req<ExerciseSummary[]>('/api/progress'),

	exerciseProgress: (exerciseId: string) =>
		req<ExerciseProgressPoint[]>(`/api/progress/${exerciseId}`),

	listMetrics: (limit = 50) =>
		req<WeeklyMetric[]>(`/api/metrics?limit=${limit}`),

	saveMetric: (m: Partial<WeeklyMetric>) =>
		req<{ status: string }>('/api/metrics', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(m),
		}),
};

export type { VolumeEntry };
