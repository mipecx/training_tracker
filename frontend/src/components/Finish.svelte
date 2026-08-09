<script lang="ts">
	import { ui } from '../lib/store.svelte';

	const f = $derived(ui.finish);
	const mins = $derived(f ? Math.round(f.durationSec / 60) || 0 : 0);
	const workoutName = $derived(ui.workout?.name ?? '');

	function goHome() {
		ui.workout = null;
		ui.exIdx = 0;
		ui.setIdx = 0;
		ui.log = [];
		ui.startTs = null;
		ui.finish = null;
		ui.screen = 'home';
	}
</script>

<div id="finish-body">
	{#if f}
		<div class="finish-card">
			<div class="finish-title">Done</div>
			<div class="finish-sub">{workoutName}</div>
			<div class="finish-stats">
				<div class="finish-stat"><div class="finish-stat-val">{mins}</div><div class="finish-stat-lbl">min</div></div>
				<div class="finish-stat"><div class="finish-stat-val">{f.setsCount}</div><div class="finish-stat-lbl">sets</div></div>
				<div class="finish-stat"><div class="finish-stat-val">{f.totalReps}</div><div class="finish-stat-lbl">reps</div></div>
			</div>
			{#each Object.entries(f.byEx) as [name, entries]}
				<div class="summary-item">
					<span>{name}</span>
					<span class="summary-sets">
						{entries.map((e) => `${e.set_num + 1}${e.side ? ` (${e.side === 'left' ? 'L' : 'R'})` : ''}: ${e.duration_sec != null ? e.duration_sec + 's' : (e.reps ?? '—')}`).join(' · ')}
					</span>
				</div>
			{/each}
		</div>
	{/if}
</div>
<button class="btn btn-accent" onclick={goHome} style="margin-top:8px">← Home</button>
