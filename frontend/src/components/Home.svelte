<script lang="ts">
	import { api } from '../lib/api';
	import type { SetEntry, Workout } from '../lib/types';
	import { ui, ALL_DAYS, todayDay } from '../lib/store.svelte';

	let loading = $state(true);
	let workout = $state<Workout | null>(null);
	let activeDay = $state<string>('');
	let lastSets = $state<Record<string, SetEntry[]>>({});

	async function load(day?: string) {
		loading = true;
		const target = day ?? todayDay();
		if (!target) {
			workout = null;
			activeDay = '';
			loading = false;
			return;
		}
		try {
			const data = await api.workoutByDay(target);
			workout = data;
			activeDay = target;
			await refreshHistory();
		} catch {
			workout = null;
			activeDay = '';
		} finally {
			loading = false;
		}
	}

	async function refreshHistory() {
		if (!workout) return;
		const seen: Record<string, SetEntry[]> = {};
		for (const ex of workout.exercises) {
			try {
				const hist = await api.exerciseHistory(ex.id, 10);
				const seenKeys = new Set<string>();
				seen[ex.id] = hist
					.filter((h) => {
						const key = `${h.SessionID}_${h.SetNum}`;
						if (seenKeys.has(key)) return false;
						seenKeys.add(key);
						return true;
					})
					.slice(0, ex.sets);
			} catch {
				seen[ex.id] = [];
			}
		}
		lastSets = seen;
	}

	async function pickDay(day: string) {
		loading = true;
		try {
			workout = await api.workoutByDay(day);
			activeDay = day;
			await refreshHistory();
		} finally {
			loading = false;
		}
	}

	async function startWorkout() {
		if (!workout) return;
		ui.workout = workout;
		ui.exIdx = 0;
		ui.setIdx = 0;
		ui.curSide = 'left';
		ui.log = [];
		ui.startTs = Date.now();
		ui.prevSessionSets = {};
		try {
			const sessions = await api.listSessions(50);
			const prev = sessions.find((s) => s.WorkoutID === workout!.day && s.FinishedAt);
			if (prev) {
				const data = await api.getSession(prev.ID);
				const seen = new Set<string>();
				for (const s of data.sets) {
					const key = `${s.SetNum}_${s.Side || ''}`;
					if (seen.has(key)) continue;
					seen.add(key);
					if (!ui.prevSessionSets[s.ExerciseID]) ui.prevSessionSets[s.ExerciseID] = [];
					ui.prevSessionSets[s.ExerciseID].push({
						set_num: s.SetNum,
						reps: s.Reps ?? null,
						duration_sec: s.DurationSec ?? null,
						side: s.Side ?? null,
					});
				}
			}
			const resp = await api.createSession(workout!.day, workout!.name);
			ui.sessionId = resp.session_id;
			ui.screen = 'workout';
		} catch {
			alert('Failed to start workout');
		}
	}

	$effect(() => {
		load();
	});

	$effect(() => {
		if (!loading && !workout) {
			lastSets = {};
		}
	});
</script>

<div id="home-body">
	{#if loading}
		<div class="loading-txt">Loading...</div>
	{:else if !workout}
		<div class="rest-day-card">
			<div class="workout-tag">Today</div>
			<div class="workout-name">Rest day</div>
			<p style="color:var(--text2);font-size:13px;margin-top:8px">Recovery is part of training.</p>
		</div>
		<div class="section-label">Other days</div>
		<div class="day-chips">
			{#each ALL_DAYS as d (d.day)}
				<button class="day-chip" onclick={() => pickDay(d.day)}>
					<div class="day-chip-name">{d.name}</div>
					<div class="day-chip-day">{d.label}</div>
				</button>
			{/each}
		</div>
	{:else}
		<div class="workout-card">
			<div class="workout-tag">{activeDay === todayDay() ? 'Today' : 'Another day'}</div>
			<div class="workout-name">{workout.name}</div>
			<div class="ex-preview">
				{#each workout.exercises as ex (ex.id)}
					{@const hist = lastSets[ex.id] || []}
					{@const prevStr = hist.length > 0 ? hist.map((s) => (ex.is_timed ? `${s.DurationSec ?? '—'}s` : `${s.Reps ?? '—'}`)).join(' · ') : null}
					<div class="ex-preview-item">
						<div style="flex:1">
							<span class="ex-preview-name">{ex.name}</span>
							{#if ex.difficulty_level}<span class="lvl-badge">LVL {ex.difficulty_level}/{ex.difficulty_max}</span>{/if}
							<span class="ex-preview-meta">
								{ex.sets}×{ex.reps_target}
								{#if prevStr}<b style="color:var(--accent)"> ({prevStr})</b>{/if}
							</span>
						</div>
					</div>
				{/each}
			</div>
			<button class="btn btn-accent" onclick={startWorkout}>Start workout</button>
		</div>
		<div class="section-label">Other days</div>
		<div class="day-chips">
			{#each ALL_DAYS.filter((d) => d.day !== activeDay) as d (d.day)}
				<button class="day-chip" onclick={() => pickDay(d.day)}>
					<div class="day-chip-name">{d.name}</div>
					<div class="day-chip-day">{d.label}</div>
				</button>
			{/each}
		</div>
	{/if}
</div>
