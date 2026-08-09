<script lang="ts">
	import { api } from '../lib/api';
	import type { Exercise } from '../lib/types';
	import { ui, currentExercise } from '../lib/store.svelte';

	let timedRunning = $state(false);
	let timedElapsed = $state(0);
	let timedIv: ReturnType<typeof setInterval> | null = null;
	let timedStartTs = $state(0);

	const ex = $derived(currentExercise(ui));
	const totalSets = $derived(ui.workout ? ui.workout.exercises.reduce((s, e) => s + e.sets, 0) : 0);
	const doneSets = $derived(
		ui.workout ? ui.workout.exercises.slice(0, ui.exIdx).reduce((s, e) => s + e.sets, 0) + ui.setIdx : 0
	);
	const pct = $derived(totalSets ? (doneSets / totalSets) * 100 : 0);
	const lastSet = $derived(ex ? ui.setIdx + 1 >= ex.sets : true);
	const lastEx = $derived(ex ? ui.exIdx + 1 >= (ui.workout?.exercises.length ?? 0) : true);

	function fmtRest(sec: number) {
		const m = Math.floor(sec / 60);
		const s = sec % 60;
		return s ? `${m}m${s}s` : `${m} min`;
	}

	function startTimed() {
		if (timedRunning) return;
		timedRunning = true;
		timedStartTs = Date.now();
		timedIv = setInterval(() => {
			timedElapsed = Math.round((Date.now() - timedStartTs) / 1000);
		}, 250);
	}

	function stopTimed() {
		if (!timedRunning) return;
		timedRunning = false;
		if (timedIv) clearInterval(timedIv);
		timedIv = null;
		timedElapsed = Math.round((Date.now() - timedStartTs) / 1000);
		if (ex) {
			saveSet(ex, timedElapsed, true);
		}
	}

	function resetTimed() {
		stopTimed();
		timedElapsed = 0;
	}

	function timedElapsedStr() {
		const m = Math.floor(timedElapsed / 60);
		const s = timedElapsed % 60;
		return `${m}:${s.toString().padStart(2, '0')}`;
	}

	function logKey(ex: Exercise) {
		return ex.per_side ? `${ex.id}_${ui.setIdx}_${ui.curSide}` : `${ex.id}_${ui.setIdx}`;
	}

	async function saveSet(ex: Exercise, value: number, isTimed: boolean) {
		if (!ui.sessionId) return;
		const entry = {
			exercise_id: ex.id,
			exercise_name: ex.name,
			set_num: ui.setIdx,
			side: ex.per_side ? ui.curSide : null,
			reps: isTimed ? null : value,
			duration_sec: isTimed ? value : null,
			notes: '',
		};
		try {
			await api.logSet(ui.sessionId, entry);
		} catch {
		}
		ui.log.push(entry);
	}

	async function onRepsChange(i: number, val: string) {
		if (!ex) return;
		const v = parseInt(val, 10);
		if (!isNaN(v)) await saveSet(ex, v, false);
	}

	async function doneSet() {
		if (!ex) return;
		if (ex.is_timed) {
			stopTimed();
		} else {
			const inp = document.getElementById(`si-${ui.setIdx}`) as HTMLInputElement | null;
			if (inp?.value) await onRepsChange(ui.setIdx, inp.value);
		}
		if (lastSet && lastEx) {
			showFinish();
			return;
		}
		if (lastSet) {
			ui.exIdx++;
			ui.setIdx = 0;
			ui.curSide = 'left';
		} else {
			ui.setIdx++;
		}
		ui.screen = 'rest';
		startRest(ex.rest_sec);
		timedElapsed = 0;
	}

	let restRem = $state(0);
	let restTotal = $state(0);
	let restIv: ReturnType<typeof setInterval> | null = null;
	let restNextLabel = $state('');

	function startRest(sec: number) {
		restRem = sec;
		restTotal = sec;
		const next = ui.workout?.exercises[ui.exIdx];
		restNextLabel = next ? `Next: <b>${next.name}</b>` : '';
		clearInterval(restIv ?? undefined);
		restIv = setInterval(() => {
			restRem--;
			if (restRem <= 0) {
				clearInterval(restIv!);
				setTimeout(() => nextEx(), 300);
			}
		}, 1000);
	}

	function nextEx() {
		if (restIv) clearInterval(restIv);
		ui.screen = 'workout';
	}

	function adjRest(d: number) {
		restRem = Math.max(0, restRem + d);
		restTotal = Math.max(restTotal, restRem);
	}

	function skipRest() {
		if (restIv) clearInterval(restIv);
		nextEx();
	}

	function restElapsedStr() {
		const m = Math.floor(restRem / 60);
		const s = restRem % 60;
		return `${m}:${s.toString().padStart(2, '0')}`;
	}

	async function showFinish() {
		if (restIv) clearInterval(restIv);
		const durationSec = ui.startTs ? Math.round((Date.now() - ui.startTs) / 1000) : 0;
		try {
			await api.finishSession(ui.sessionId!, durationSec);
		} catch {
		}
		const byEx: typeof ui.finish extends null ? never : NonNullable<typeof ui.finish>['byEx'] = {};
		for (const l of ui.log) {
			(byEx[l.exercise_name] ??= []).push({ set_num: l.set_num, side: l.side, reps: l.reps, duration_sec: l.duration_sec });
		}
		ui.finish = {
			durationSec,
			totalReps: ui.log.reduce((s, l) => s + (l.reps ?? 0), 0),
			setsCount: ui.log.length,
			byEx,
		};
		ui.screen = 'finish';
	}

	function skipEx() {
		if (ui.exIdx + 1 >= (ui.workout?.exercises.length ?? 0)) {
			showFinish();
			return;
		}
		ui.exIdx++;
		ui.setIdx = 0;
		ui.curSide = 'left';
		timedElapsed = 0;
	}

	function setSide(s: 'left' | 'right') {
		ui.curSide = s;
	}

	const prevSets = $derived(ex ? ui.prevSessionSets[ex.id] || [] : []);
</script>

{#if ui.screen === 'workout'}
	{#if ex}
		<div class="progress-row">
			<div class="progress-bar"><div class="progress-fill" style="width:{pct}%"></div></div>
			<div class="progress-txt">{ui.exIdx + 1}/{ui.workout?.exercises.length}</div>
		</div>

		<div id="ex-wrap">
			<div class="ex-card">
				<div class="ex-counter">Ex {ui.exIdx + 1}/{ui.workout?.exercises.length} · Set {ui.setIdx + 1}/{ex.sets}</div>
				<div class="ex-name">{ex.name}</div>
				<div class="meta-row">
					<div class="meta-item"><div class="meta-label">Reps</div><div class="meta-value">{ex.reps_target}</div></div>
					<div class="meta-item"><div class="meta-label">Rest</div><div class="meta-value">{fmtRest(ex.rest_sec)}</div></div>
					<div class="meta-item"><div class="meta-label">Sets</div><div class="meta-value">{ex.sets}</div></div>
				</div>
				{#if ex.notes}<div class="notes-box">{ex.notes}</div>{/if}

				{#if prevSets.length > 0}
					<div class="prev-session-row">
						<span class="prev-session-lbl">Last time:</span>
						{#each prevSets as s, i (i)}
							<span class="prev-set-tag">{i + 1}: <b>{ex.is_timed ? (s.duration_sec ?? '—') + 's' : (s.reps ?? '—')}</b></span>
						{/each}
					</div>
				{/if}

				{#if ex.per_side}
					<div class="side-toggle">
						<button class="side-btn {ui.curSide === 'left' ? 'active' : ''}" onclick={() => setSide('left')}>Left</button>
						<button class="side-btn {ui.curSide === 'right' ? 'active' : ''}" onclick={() => setSide('right')}>Right</button>
					</div>
				{/if}

				{#if ex.is_timed}
					<div class="timed-block">
						<div class="sets-label">Set time</div>
						<div class="timed-display">{timedElapsedStr()}</div>
						{#if !timedRunning}
							<button class="btn btn-accent" onclick={startTimed}>▶ Start</button>
						{:else}
							<button class="btn btn-danger" onclick={stopTimed}>⏹ Stop ({timedElapsedStr()})</button>
						{/if}
					</div>
				{:else}
					<div class="sets-label">Reps</div>
					<div class="sets-row">
						{#each Array.from({ length: ex.sets }, (_, i) => i) as i (i)}
							{@const done = i < ui.setIdx}
							{@const current = i === ui.setIdx}
							{@const entry = ui.log.find((l) => l.exercise_id === ex.id && l.set_num === i && (!ex.per_side || l.side === ui.curSide))}
							<div class="set-col">
								<div class="set-num">{i + 1}</div>
								<input
									class="set-inp {done ? 'done' : ''}"
									id="si-{i}"
									type="number"
									inputmode="numeric"
									placeholder={ex.reps_target}
									value={entry?.reps ?? ''}
									readonly={!current}
									onchange={(e) => onRepsChange(i, (e.target as HTMLInputElement).value)}
								/>
							</div>
						{/each}
					</div>
				{/if}

				{#if ex.difficulty_level && ex.progression}
					{@const lvl = ex.difficulty_level}
					{@const lvlMax = ex.difficulty_max || ex.progression.length}
					{@const threshold = ex.difficulty_threshold}
					{@const nextProg = ex.progression.find((p) => p.level === lvl + 1)}
					{@const pctDiff = threshold ? Math.min(100, Math.round((parseInt(ex.reps_target) || threshold) / threshold * 100)) : 0}
					<div class="diff-block">
						<div class="diff-header">
							<span class="diff-label">Level {lvl}/{lvlMax}</span>
							<span class="diff-name">{ex.progression.find((p) => p.level === lvl)?.name || ex.name}</span>
						</div>
						{#if threshold}
							<div class="diff-progress">
								<div class="diff-prog-track"><div class="diff-prog-fill" style="width:{pctDiff}%"></div></div>
								<span class="diff-prog-lbl">Next level at: {threshold} reps</span>
							</div>
						{/if}
						{#if nextProg}
							<div class="next-lvl-hint">→ LVL {lvl + 1}: <b>{nextProg.name}</b> — {nextProg.description}</div>
						{:else}
							<div class="next-lvl-hint max-lvl">🏆 Max level</div>
						{/if}
					</div>
				{/if}

				{#if ex.technique?.text}
					<details class="tech-toggle">
						<summary><span class="tech-icon">?</span> How to perform</summary>
						<div class="tech-box">{ex.technique.text}</div>
					</details>
				{/if}
			</div>
		</div>

		<div class="action-row" style="margin-top:8px">
			<button class="btn btn-sm" onclick={skipEx}>Skip</button>
			<button class="btn btn-accent" style="flex:1" onclick={doneSet}>
				{ex.is_timed ? 'Done → Rest' : 'Done → Rest'}
			</button>
		</div>
	{/if}
{:else if ui.screen === 'rest'}
	<div class="rest-wrap">
		<div class="rest-lbl">Rest</div>
		<div class="arc-wrap">
			<svg class="arc-svg" viewBox="0 0 180 180">
				<circle class="arc-bg" cx="90" cy="90" r="78"/>
				<circle class="arc-fill" id="arc-fill" cx="90" cy="90" r="78"
					style="stroke-dasharray:{2 * Math.PI * 78};stroke-dashoffset:{2 * Math.PI * 78 * (1 - (restTotal ? restRem / restTotal : 0))}" />
			</svg>
			<div class="arc-center">
				<div class="rest-time {restRem <= 10 && restRem > 0 ? 'urgent' : ''}" id="rest-time">{restElapsedStr()}</div>
			</div>
		</div>
		<div class="rest-next">{@html restNextLabel}</div>
		<div class="rest-actions">
			<button class="btn btn-sm" onclick={() => adjRest(30)}>+30s</button>
			<button class="btn btn-accent" style="flex:1" onclick={skipRest}>Next →</button>
			<button class="btn btn-sm" onclick={() => adjRest(-15)}>−15s</button>
		</div>
	</div>
{/if}

<svelte:window onbeforeunload={() => { if (ui.screen === 'workout') return true; }} />
