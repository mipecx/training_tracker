<script lang="ts">
	import Chart from 'chart.js/auto';
	import { api } from '../lib/api';
	import type { CompareResult, Session, SetEntry } from '../lib/types';
	import { ui, DAYS, MONTHS } from '../lib/store.svelte';

	let loading = $state(true);
	let session = $state<Session | null>(null);
	let sets = $state<SetEntry[]>([]);
	let compare = $state<CompareResult | null>(null);
	let tab = $state<'ex' | 'chart'>('ex');
	let chart: Chart | null = null;

	async function load() {
		if (ui.selectedSessionId == null) return;
		loading = true;
		tab = 'ex';
		try {
			const [detail, comp] = await Promise.all([
				api.getSession(ui.selectedSessionId),
				api.compare(ui.selectedSessionId),
			]);
			session = detail.session;
			sets = detail.sets;
			compare = comp;
		} finally {
			loading = false;
		}
	}

	function back() {
		ui.screen = 'history';
	}

	$effect(() => {
		if (ui.screen === 'session') load();
	});

	const dedupedSets = $derived.by(() => {
		const seen = new Set<string>();
		return (sets || []).filter((s) => {
			const key = `${s.ExerciseID}_${s.SetNum}_${s.Side || ''}`;
			if (seen.has(key)) return false;
			seen.add(key);
			return true;
		});
	});

	const byEx = $derived.by(() => {
		const map: Record<string, { name: string; sets: SetEntry[] }> = {};
		for (const s of dedupedSets) {
			if (!map[s.ExerciseID]) map[s.ExerciseID] = { name: s.ExerciseName, sets: [] };
			map[s.ExerciseID].sets.push(s);
		}
		return map;
	});

	async function switchTab(t: 'ex' | 'chart') {
		tab = t;
		if (t === 'chart') {
			await new Promise<void>((r) => setTimeout(r, 0));
			renderChart();
		}
	}

	function renderChart() {
		const canvas = document.getElementById('session-chart') as HTMLCanvasElement | null;
		if (!canvas || !compare) return;
		const curr = compare.current?.volume || [];
		const prev = compare.previous?.volume || [];
		if (!curr.length) {
			document.getElementById('session-chart-wrap')!.innerHTML = '<div class="empty-state">No data</div>';
			return;
		}
		const labels = curr.map((e) => e.exercise_name.split(' ').slice(0, 2).join(' '));
		const currVol = curr.map((e) => (e.unit === 'sec' ? e.total_sec : e.sets * Math.round(e.avg_reps || 0)));
		const prevMap: Record<string, number> = {};
		prev.forEach((e) => {
			prevMap[e.exercise_id] = e.unit === 'sec' ? e.total_sec : (e.sets || 0) * Math.round(e.avg_reps || 0);
		});
		const prevVol = curr.map((e) => prevMap[e.exercise_id] || 0);

		if (chart) {
			chart.destroy();
			chart = null;
		}
		const ctx = canvas.getContext('2d');
		if (!ctx) return;
		chart = new Chart(ctx, {
			type: 'bar',
			data: {
				labels,
				datasets: [
					{
						label: 'Previous',
						data: prevVol,
						backgroundColor: '#2a2a2a',
						borderColor: '#3a3a3a',
						borderWidth: 1,
						borderRadius: 4,
					},
					{
						label: 'Current',
						data: currVol,
						backgroundColor: currVol.map((v, i) => (v >= prevVol[i] ? '#c8ff00' : '#ff4444')),
						borderColor: currVol.map((v, i) => (v >= prevVol[i] ? '#a0cc00' : '#cc2222')),
						borderWidth: 1,
						borderRadius: 4,
					},
				],
			},
				options: {
					indexAxis: 'y',
					responsive: true,
					maintainAspectRatio: false,
					plugins: {
						legend: { display: false },
						tooltip: {
							backgroundColor: '#1a1a1a',
							titleColor: '#f0f0f0',
							bodyColor: '#888',
							borderColor: '#2a2a2a',
							borderWidth: 1,
							callbacks: { label: (c) => `${c.dataset.label}: ${c.parsed.x}` },
						},
					},
				scales: {
					x: { ticks: { color: '#555', font: { family: 'JetBrains Mono', size: 10 } }, grid: { color: '#1a1a1a' }, beginAtZero: true },
					y: { ticks: { color: '#888', font: { family: 'JetBrains Mono', size: 10 }, autoSkip: false }, grid: { color: '#1a1a1a' } },
				},
			},
		});
	}
</script>

<div id="session-body">
	{#if loading}
		<div class="loading-txt">Loading...</div>
	{:else if session}
		<button class="btn btn-sm back-btn" onclick={back}>← Back</button>
		<div class="session-header-card">
			<div class="workout-tag">{DAYS[new Date(session.Date).getDay()]}, {new Date(session.Date).getDate()} {MONTHS[new Date(session.Date).getMonth()]}</div>
			<div class="workout-name">{session.WorkoutName}</div>
			<div class="session-stat-row">
				<div class="session-stat"><div class="session-stat-val">{Math.round(session.DurationSec / 60)}</div><div class="session-stat-lbl">min</div></div>
				<div class="session-stat"><div class="session-stat-val">{dedupedSets.length}</div><div class="session-stat-lbl">sets</div></div>
				<div class="session-stat"><div class="session-stat-val">{dedupedSets.reduce((a, s) => a + (s.Reps || 0), 0)}</div><div class="session-stat-lbl">reps</div></div>
			</div>
		</div>

		<div class="session-tabs">
			<button class="session-tab {tab === 'ex' ? 'active' : ''}" id="stab-ex" onclick={() => switchTab('ex')}>Exercises</button>
			<button class="session-tab {tab === 'chart' ? 'active' : ''}" id="stab-chart" onclick={() => switchTab('chart')}>Chart</button>
		</div>

		{#if tab === 'ex'}
			<div id="stab-panel-ex">
				{#each Object.entries(byEx) as [exId, { name, sets }]}
					<div class="session-ex-item">
						<div class="session-ex-header">
							<span class="session-ex-name">{name}</span>
							<span class="session-ex-vol">{sets.reduce((a, s) => a + (s.Reps || 0), 0)} reps</span>
						</div>
						<div class="set-pills">
							{#each sets as s, i (i)}
								<span class="set-pill">{i + 1}: {s.DurationSec != null ? s.DurationSec + 's' : (s.Reps ?? '—')}</span>
							{/each}
						</div>
					</div>
				{/each}
			</div>
		{:else}
			<div id="stab-panel-chart">
				<div id="session-chart-wrap" style="height:{Math.max(200, Object.keys(byEx).length * 44 + 60)}px">
					<div class="chart-legend">
						<span class="leg-item"><span class="leg-dot" style="background:#333"></span>Previous</span>
						<span class="leg-item"><span class="leg-dot" style="background:#c8ff00"></span>Current</span>
					</div>
					<div style="position:relative;width:100%;height:100%;margin-top:12px">
						<canvas id="session-chart" aria-label="Exercise volume comparison"></canvas>
					</div>
				</div>
			</div>
		{/if}
	{:else}
		<div class="empty-state">Failed to load</div>
	{/if}
</div>
