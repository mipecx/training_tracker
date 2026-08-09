import type { Exercise, Workout } from './types';

export type Screen = 'home' | 'workout' | 'rest' | 'finish' | 'history' | 'session' | 'progress';

export const ALL_DAYS = [
	{ day: 'monday', label: 'Mon', name: 'PUSH' },
	{ day: 'wednesday', label: 'Wed', name: 'PULL' },
	{ day: 'friday', label: 'Fri', name: 'FULL' },
	{ day: 'sunday', label: 'Sun', name: 'LEGS+CORE' },
];

export const DAYS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
export const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];

function todayDay(): string {
	const map: Record<number, string> = { 1: 'monday', 3: 'wednesday', 5: 'friday', 0: 'sunday' };
	return map[new Date().getDay()] ?? '';
}

export const ui = $state({
	screen: 'home' as Screen,
	workout: null as Workout | null,
	exIdx: 0,
	setIdx: 0,
	curSide: 'left' as 'left' | 'right',
	log: [] as Array<{ exercise_id: string; exercise_name: string; set_num: number; side: string | null; reps: number | null; duration_sec: number | null }>,
	startTs: null as number | null,
	sessionId: null as number | null,
	prevSessionSets: {} as Record<string, Array<{ set_num: number; reps: number | null; duration_sec: number | null; side: string | null }>>,
	historyForWorkout: {} as Record<string, SetEntryShort[]>,
	selectedSessionId: null as number | null,
	finish: null as null | {
		durationSec: number;
		totalReps: number;
		setsCount: number;
		byEx: Record<string, Array<{ set_num: number; side: string | null; reps: number | null; duration_sec: number | null }>>;
	},
});

export interface SetEntryShort {
	SessionID: number;
	SetNum: number;
	Reps?: number | null;
	DurationSec?: number | null;
	Side?: string | null;
	ExerciseID: string;
}

export function currentExercise(state: typeof ui): Exercise | null {
	if (!state.workout) return null;
	return state.workout.exercises[state.exIdx] ?? null;
}

export function getMondayDate(): string {
	const dt = new Date();
	const day = dt.getDay();
	const diff = dt.getDate() - day + (day === 0 ? -6 : 1);
	dt.setDate(diff);
	return dt.toISOString().split('T')[0];
}

export function fmtDateLabel(str: string): string {
	const d = new Date(str);
	return `${d.getDate()} ${MONTHS[d.getMonth()]}`;
}

export function fmtRest(sec: number): string {
	const m = Math.floor(sec / 60);
	const s = sec % 60;
	return s ? `${m}m${s}s` : `${m} min`;
}

export function fmtDuration(sec: number): string {
	const m = Math.floor(sec / 60);
	const s = sec % 60;
	return s ? `${m}:${s.toString().padStart(2, '0')}` : `${m}m`;
}

export { todayDay };

export type { Workout, Exercise };
