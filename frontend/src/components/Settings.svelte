<script lang="ts">
	import { api } from '../lib/api';
	import type { Exercise, Workout } from '../lib/types';
	import { ALL_DAYS } from '../lib/store.svelte';

	let loading = $state(false);
	let saving = $state(false);
	let workout = $state<Workout | null>(null);
	let activeDay = $state<string>('monday');
	let status = $state('');
	let error = $state('');

	function emptyExercise(): Exercise {
		return {
			id: `ex_${Date.now()}`,
			name: 'New exercise',
			sets: 3,
			reps_target: '10',
			rest_sec: 120,
			notes: '',
			per_side: false,
			is_timed: false,
			difficulty_level: 1,
			difficulty_max: 1,
			technique: { text: '', image: '' },
		};
	}

	async function load(day: string) {
		loading = true;
		error = '';
		status = '';
		activeDay = day;
		try {
			workout = await api.workoutByDay(day);
			for (const ex of workout.exercises) {
				ex.technique ??= { text: '', image: '' };
				ex.difficulty_level ??= 1;
				ex.difficulty_max ??= 1;
			}
		} catch {
			error = 'Failed to load workout';
			workout = null;
		} finally {
			loading = false;
		}
	}

	function addExercise() {
		if (!workout) return;
		workout.exercises.push(emptyExercise());
	}

	function removeExercise(i: number) {
		if (!workout) return;
		workout.exercises.splice(i, 1);
	}

	function moveExercise(i: number, dir: -1 | 1) {
		if (!workout) return;
		const j = i + dir;
		if (j < 0 || j >= workout.exercises.length) return;
		const arr = workout.exercises;
		[arr[i], arr[j]] = [arr[j], arr[i]];
	}

	function changeLevel(i: number, dir: -1 | 1) {
		if (!workout) return;
		const ex = workout.exercises[i];
		const max = ex.difficulty_max ?? 1;
		const cur = ex.difficulty_level ?? 1;
		const next = Math.min(max, Math.max(1, cur + dir));
		ex.difficulty_level = next;
		const prog = ex.progression?.find((p) => p.level === next);
		if (prog) ex.name = prog.name;
	}

	function lvlName(ex: Exercise): string {
		const prog = ex.progression?.find((p) => p.level === (ex.difficulty_level ?? 1));
		return prog ? prog.name : (ex.name || '');
	}

	function genId(name: string) {
		const base = name.toLowerCase().replace(/[^a-z0-9]+/g, '_').replace(/^_+|_+$/g, '');
		return base || `ex_${Date.now()}`;
	}

	async function save() {
		if (!workout) return;
		saving = true;
		status = '';
		error = '';
		const used = new Set<string>();
		for (const ex of workout.exercises) {
			if (!ex.id || used.has(ex.id)) {
				ex.id = genId(ex.name) || `ex_${Date.now()}_${Math.random().toString(36).slice(2, 7)}`;
			}
			used.add(ex.id);
		}
		try {
			const saved = await api.saveWorkout(activeDay, workout);
			workout = saved;
			for (const ex of workout.exercises) {
				ex.technique ??= { text: '', image: '' };
				ex.difficulty_level ??= 1;
				ex.difficulty_max ??= 1;
			}
			status = 'Saved';
			setTimeout(() => (status = ''), 2000);
		} catch {
			error = 'Save error';
		} finally {
			saving = false;
		}
	}

	$effect(() => {
		load('monday');
	});
</script>

<div id="settings-body">
	<div class="section-label">Day of week</div>
	<div class="day-chips">
		{#each ALL_DAYS as d (d.day)}
			<button class="day-chip {d.day === activeDay ? 'day-chip-active' : ''}" onclick={() => load(d.day)}>
				<div class="day-chip-name">{d.name}</div>
				<div class="day-chip-day">{d.label}</div>
			</button>
		{/each}
	</div>

	{#if loading}
		<div class="loading-txt">Loading...</div>
	{:else if error && !workout}
		<div class="empty-state">{error}</div>
	{:else if workout}
		<div class="settings-header">
			<div class="workout-tag">Editing</div>
			<input class="metric-inp workout-name-input" placeholder="Workout name" bind:value={workout.name} />
		</div>

		{#each workout.exercises as ex, i (ex.id)}
			<div class="ex-edit-card">
				<div class="ex-edit-header">
					<span class="ex-edit-num">{i + 1}</span>
					<input class="ex-edit-name metric-inp" placeholder="Exercise name" bind:value={ex.name} />
					<div class="ex-edit-arrows">
						<button class="mini-btn" onclick={() => moveExercise(i, -1)} disabled={i === 0}>↑</button>
						<button class="mini-btn" onclick={() => moveExercise(i, 1)} disabled={i === workout.exercises.length - 1}>↓</button>
						<button class="mini-btn mini-del" onclick={() => removeExercise(i)}>✕</button>
					</div>
				</div>

				<div class="ex-edit-grid">
					<div class="m-field">
						<label class="metric-label" for="id-{i}">ID</label>
						<input class="metric-inp" id="id-{i}" bind:value={ex.id} placeholder="unique_id" />
					</div>
					<div class="m-field">
						<label class="metric-label" for="sets-{i}">Sets</label>
						<input class="metric-inp" id="sets-{i}" type="number" min="1" bind:value={ex.sets} />
					</div>
					<div class="m-field">
						<label class="metric-label" for="reps-{i}">Reps / target</label>
						<input class="metric-inp" id="reps-{i}" bind:value={ex.reps_target} placeholder="8–12" />
					</div>
					<div class="m-field">
						<label class="metric-label" for="rest-{i}">Rest (sec)</label>
						<input class="metric-inp" id="rest-{i}" type="number" min="0" step="15" bind:value={ex.rest_sec} />
					</div>
					<div class="m-field">
						<label class="metric-label" for="lvlmax-{i}">Max level</label>
						<input class="metric-inp" id="lvlmax-{i}" type="number" min="1" bind:value={ex.difficulty_max} />
					</div>
				</div>

				<div class="lvl-block">
					<div class="metric-label">Difficulty level</div>
					<div class="lvl-switch">
						<button class="lvl-btn" onclick={() => changeLevel(i, -1)} disabled={(ex.difficulty_level ?? 1) <= 1}>−</button>
						<div class="lvl-indicator">
							<span class="lvl-num">LVL {ex.difficulty_level ?? 1}/{ex.difficulty_max ?? 1}</span>
							<span class="lvl-name">{lvlName(ex)}</span>
						</div>
						<button class="lvl-btn" onclick={() => changeLevel(i, 1)} disabled={(ex.difficulty_level ?? 1) >= (ex.difficulty_max ?? 1)}>+</button>
					</div>
				</div>

				<div class="m-field" style="margin-top:8px">
					<label class="metric-label" for="notes-{i}">Notes</label>
					<textarea class="metric-inp metric-notes" id="notes-{i}" placeholder="How to perform..." bind:value={ex.notes}></textarea>
				</div>
				<div class="m-field" style="margin-top:8px">
					<label class="metric-label" for="tech-{i}">Technique</label>
					<textarea class="metric-inp metric-notes" id="tech-{i}" placeholder="Execution technique..." bind:value={ex.technique!.text}></textarea>
				</div>

				<div class="ex-flags">
					<label class="flag-item"><input type="checkbox" bind:checked={ex.per_side} /> Per side</label>
					<label class="flag-item"><input type="checkbox" bind:checked={ex.is_timed} /> Timed (sec)</label>
				</div>
			</div>
		{/each}

		<button class="btn btn-sm add-ex-btn" onclick={addExercise}>+ Add exercise</button>

		{#if error}<div class="empty-state" style="color:var(--red);padding:12px">{error}</div>{/if}

		<button class="btn btn-accent" style="margin-top:14px" onclick={save} disabled={saving}>
			{saving ? 'Saving...' : 'Save'}
		</button>
		{#if status}<div style="text-align:center;font-size:12px;color:var(--accent);margin-top:8px">{status}</div>{/if}
	{/if}
</div>
