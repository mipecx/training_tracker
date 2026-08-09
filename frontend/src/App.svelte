<script lang="ts">
	import { ui, DAYS, MONTHS } from './lib/store.svelte';
	import Home from './components/Home.svelte';
	import Workout from './components/Workout.svelte';
	import Finish from './components/Finish.svelte';
	import History from './components/History.svelte';
	import SessionDetail from './components/SessionDetail.svelte';
	import Settings from './components/Settings.svelte';

	const now = new Date();
	const hdr = $derived(`${DAYS[now.getDay()]}, ${now.getDate()} ${MONTHS[now.getMonth()]}`);

	function navTo(section: string) {
		ui.screen = section as typeof ui.screen;
	}
</script>

<div id="app">
	<div class="header">
		<div class="logo">GYMTRACKER</div>
		<div class="day-badge" id="hdr-date">{hdr}</div>
	</div>

	{#if ui.screen === 'home'}
		<div id="screen-home" class="screen active"><Home /></div>
	{:else if ui.screen === 'workout' || ui.screen === 'rest'}
		<div id="screen-workout" class="screen active"><Workout /></div>
	{:else if ui.screen === 'finish'}
		<div id="screen-finish" class="screen active"><Finish /></div>
	{:else if ui.screen === 'history'}
		<div id="screen-history" class="screen active"><History /></div>
	{:else if ui.screen === 'session'}
		<div id="screen-session" class="screen active"><SessionDetail /></div>
	{:else if ui.screen === 'progress'}
		<div id="screen-progress" class="screen active"><Settings /></div>
	{/if}

	<nav class="bottom-nav">
		<button class="nav-btn {ui.screen === 'home' ? 'active' : ''}" id="nav-home" onclick={() => navTo('home')}>
			<svg width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M3 9.5L12 3l9 6.5V20a1 1 0 01-1 1H4a1 1 0 01-1-1V9.5z"/><path d="M9 21V12h6v9"/></svg>
			<span>Today</span>
		</button>
		<button class="nav-btn {ui.screen === 'history' || ui.screen === 'session' ? 'active' : ''}" id="nav-history" onclick={() => navTo('history')}>
			<svg width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><rect x="3" y="4" width="18" height="18" rx="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/></svg>
			<span>History</span>
		</button>
		<button class="nav-btn {ui.screen === 'progress' ? 'active' : ''}" id="nav-progress" onclick={() => navTo('progress')}>
			<svg width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 11-2.83 2.83l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 11-4 0v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 11-2.83-2.83l.06-.06a1.65 1.65 0 00.33-1.82 1.65 1.65 0 00-1.51-1H3a2 2 0 110-4h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 112.83-2.83l.06.06a1.65 1.65 0 001.82.33H9a1.65 1.65 0 001-1.51V3a2 2 0 114 0v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 112.83 2.83l-.06.06a1.65 1.65 0 00-.33 1.82V9a1.65 1.65 0 001.51 1H21a2 2 0 110 4h-.09a1.65 1.65 0 00-1.51 1z"/></svg>
			<span>Settings</span>
		</button>
	</nav>
</div>
