<script lang="ts">
	import { api } from '../lib/api';
	import type { Session } from '../lib/types';
	import { ui, DAYS, MONTHS } from '../lib/store.svelte';
	import AiReview from './AiReview.svelte';

	let loading = $state(true);
	let sessions = $state<Session[]>([]);
	const byMonth = $derived.by(() => {
		const map: Record<string, { label: string; sessions: Session[] }> = {};
		for (const s of sessions.filter((s) => s.FinishedAt)) {
			const d = new Date(s.Date);
			const key = `${d.getFullYear()}-${d.getMonth()}`;
			const label = `${['January', 'February', 'March', 'April', 'May', 'June', 'July', 'August', 'September', 'October', 'November', 'December'][d.getMonth()]} ${d.getFullYear()}`;
			if (!map[key]) map[key] = { label, sessions: [] };
			map[key].sessions.push(s);
		}
		return map;
	});

	const typeColors: Record<string, string> = {
		monday: '#4ade80',
		wednesday: '#60a5fa',
		friday: '#fbbf24',
		sunday: '#f472b6',
	};

	async function load() {
		loading = true;
		try {
			sessions = await api.listSessions(100);
		} finally {
			loading = false;
		}
	}

	function openSession(id: number) {
		ui.selectedSessionId = id;
		ui.screen = 'session';
	}

	$effect(() => {
		if (ui.screen === 'history') load();
	});
</script>

<div id="history-body">
	<AiReview />

	{#if loading}
		<div class="loading-txt">Loading...</div>
	{:else if sessions.filter((s) => s.FinishedAt).length === 0}
		<div class="empty-state">No completed workouts</div>
	{:else}
		{#each Object.entries(byMonth) as [key, group]}
			<div class="section-label">{group.label}</div>
			{#each group.sessions as s (s.ID)}
				<div class="history-item" role="button" tabindex="0" onclick={() => openSession(s.ID)} onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') openSession(s.ID); }}>
					<div class="history-dot" style="background:{typeColors[s.WorkoutID] || '#888'}"></div>
					<div class="history-info">
						<div class="history-name">{s.WorkoutName}</div>
						<div class="history-date">{DAYS[new Date(s.Date).getDay()]}, {new Date(s.Date).getDate()} {MONTHS[new Date(s.Date).getMonth()]}</div>
					</div>
					<div class="history-meta">
						<div class="history-dur">{Math.round(s.DurationSec / 60)} min</div>
						<svg width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><polyline points="9 18 15 12 9 6"/></svg>
					</div>
				</div>
			{/each}
		{/each}
	{/if}
</div>
