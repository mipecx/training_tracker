export interface Technique {
	text: string;
	image: string;
}

export interface ProgressionLevel {
	level: number;
	name: string;
	description: string;
}

export interface Exercise {
	id: string;
	name: string;
	sets: number;
	reps_target: string;
	rest_sec: number;
	notes?: string;
	per_side?: boolean;
	is_timed?: boolean;
	difficulty_level?: number;
	difficulty_max?: number;
	difficulty_threshold?: number;
	progression?: ProgressionLevel[];
	technique?: Technique;
}

export interface Workout {
	id: string;
	name: string;
	day: string;
	exercises: Exercise[];
}

export interface SetEntry {
	ID: number;
	SessionID: number;
	ExerciseID: string;
	ExerciseName: string;
	SetNum: number;
	Side?: string | null;
	Reps?: number | null;
	DurationSec?: number | null;
	Notes: string;
	LoggedAt: string;
}

export interface Session {
	ID: number;
	WorkoutID: string;
	WorkoutName: string;
	Date: string;
	StartedAt: string;
	FinishedAt?: string | null;
	DurationSec: number;
}

export interface WeeklyMetric {
	id: number;
	week_date: string;
	weight_kg?: number | null;
	height_cm?: number | null;
	waist_cm?: number | null;
	neck_cm?: number | null;
	notes?: string;
	created_at: string;
}

export interface ExerciseSummary {
	exercise_id: string;
	exercise_name: string;
	all_time_max: number;
	all_time_max_sec: number;
	sessions_count: number;
	last_date: string;
	unit: 'reps' | 'sec';
}

export interface ExerciseProgressPoint {
	date: string;
	max_reps: number;
	avg_reps: number;
	sets: number;
	max_sec: number;
	total_sec: number;
}

export interface VolumeEntry {
	exercise_id: string;
	exercise_name: string;
	sets: number;
	total_reps: number;
	max_reps: number;
	avg_reps: number;
	total_sec: number;
	max_sec: number;
	unit: 'reps' | 'sec';
	volume: number;
}

export interface CompareResult {
	current: { session: Session | null; volume: VolumeEntry[] };
	previous: { session: Session | null; volume: VolumeEntry[] };
}
