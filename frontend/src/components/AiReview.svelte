<script lang="ts">
	import { api } from '../lib/api';

	const DEFAULT_WORKER = 'https://plan-llm-proxy.mipec404.workers.dev';

	const SYSTEM = [
		'Ты тренер по калистенике и аналитик тренировок.',
		'Отвечай ТОЛЬКО валидным JSON: {"summary":"2–3 предложения","score":0-100,"progress":["string"],"risks":["string"],"next":["string"]}',
		'Опирайся только на цифры: даты, регулярность, длительность, подходы, повторения и прогресс упражнений.',
		'Если был большой перерыв — скажи прямо и предложи реалистичный возврат без перегрузки.',
		'next — 2–4 конкретных действия на ближайшую неделю. Без мотивационного шума.'
	].join('\n');

	type AiResult = {
		summary?: string;
		score?: number;
		progress?: string[];
		risks?: string[];
		next?: string[];
	};

	let loading = $state(false);
	let error = $state('');
	let result = $state<AiResult | null>(null);

	let workerUrl = $state(localStorage.getItem('gym_ai_worker') || DEFAULT_WORKER);
	let model = $state(localStorage.getItem('gym_ai_model') || 'deepseek-chat');
	let token = $state(localStorage.getItem('gym_ai_token') || '');

	function saveConfig() {
		localStorage.setItem('gym_ai_worker', workerUrl.trim());
		localStorage.setItem('gym_ai_model', model.trim());
		localStorage.setItem('gym_ai_token', token.trim());
	}

	async function analyze() {
		loading = true;
		error = '';
		result = null;

		try {
			const sessions = (await api.listSessions(100)).filter((s) => s.FinishedAt).slice(0, 8);
			if (!sessions.length) throw new Error('Пока нет завершённых тренировок');

			const [summaries, metrics, details] = await Promise.all([
				api.allExerciseSummary(),
				api.listMetrics(50),
				Promise.all(sessions.map((s) => api.getSession(s.ID))),
			]);

			const sessionRows = sessions.map((s, i) => {
				const sets = details[i].sets || [];
				const reps = sets.reduce((sum, set) => sum + (set.Reps || 0), 0);
				const exercises = Array.from(new Set(sets.map((set) => set.ExerciseName))).slice(0, 8);
				return {
					date: s.Date,
					name: s.WorkoutName,
					durationMin: Math.round(s.DurationSec / 60),
					sets: sets.length,
					reps,
					exercises,
				};
			});

			const payload = {
				today: new Date().toISOString().slice(0, 10),
				targetPerWeek: 4,
				sessions: sessionRows,
				exercises: summaries.slice(0, 15).map((x) => ({
					name: x.exercise_name,
					max: x.all_time_max,
					maxSec: x.all_time_max_sec,
					sessions: x.sessions_count,
					lastDate: x.last_date,
					unit: x.unit,
				})),
				metrics: metrics.slice(-8),
			};

			const headers: Record<string, string> = { 'content-type': 'application/json' };
			if (token.trim()) headers['x-app-token'] = token.trim();

			const res = await fetch(workerUrl.trim(), {
				method: 'POST',
				headers,
				body: JSON.stringify({
					model: model.trim() || undefined,
					messages: [
						{ role: 'system', content: SYSTEM },
						{ role: 'user', content: JSON.stringify(payload) },
					],
					response_format: { type: 'json_object' },
					temperature: 0.2,
				}),
			});

			if (!res.ok) throw new Error(`Worker ответил ${res.status}`);
			const data = await res.json();
			const text = data.choices?.[0]?.message?.content || '';
			if (!text) throw new Error('Пустой ответ модели');

			const clean = String(text).trim().replace(/^```(?:json)?/i, '').replace(/```$/, '').trim();
			result = JSON.parse(clean) as AiResult;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Не удалось получить разбор';
		} finally {
			loading = false;
		}
	}
</script>

<div class="ai-card">
	<div class="ai-head">
		<div>
			<div class="workout-tag">AI</div>
			<div class="ai-title">Training review</div>
		</div>
		<button class="btn btn-accent btn-sm" onclick={analyze} disabled={loading}>
			{loading ? '…' : '✨ Analyze'}
		</button>
	</div>

	{#if error}
		<div class="ai-error">{error}</div>
	{:else if result}
		{#if typeof result.score === 'number'}
			<div class="ai-score">Оценка {result.score}/100</div>
		{/if}
		{#if result.summary}
			<p class="ai-summary">{result.summary}</p>
		{/if}
		{#if result.progress?.length}
			<div class="ai-label">Прогресс</div>
			<ul>{#each result.progress as item}<li>{item}</li>{/each}</ul>
		{/if}
		{#if result.risks?.length}
			<div class="ai-label">Риски</div>
			<ul>{#each result.risks as item}<li>{item}</li>{/each}</ul>
		{/if}
		{#if result.next?.length}
			<div class="ai-label">План</div>
			<ul>{#each result.next as item}<li>{item}</li>{/each}</ul>
		{/if}
	{:else if !loading}
		<p class="ai-hint">Разбор последних тренировок: регулярность, объём, прогресс и что делать дальше.</p>
	{/if}

	<details class="ai-settings">
		<summary>AI settings</summary>
		<label>
			<span>Worker URL</span>
			<input bind:value={workerUrl} onchange={saveConfig} placeholder="https://plan-llm-proxy.USER.workers.dev" />
		</label>
		<label>
			<span>Model</span>
			<input bind:value={model} onchange={saveConfig} placeholder="deepseek-chat" />
		</label>
		<label>
			<span>App token (optional)</span>
			<input type="password" bind:value={token} onchange={saveConfig} placeholder="x-app-token" />
		</label>
	</details>
</div>

<style>
	.ai-card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: 14px;
		padding: 16px;
		margin-bottom: 12px;
	}
	.ai-head {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 12px;
		margin-bottom: 10px;
	}
	.ai-title {
		font-size: 18px;
		font-weight: 600;
		letter-spacing: -0.01em;
	}
	.ai-hint {
		color: var(--text2);
		font-size: 12.5px;
		line-height: 1.5;
	}
	.ai-error {
		color: var(--red);
		font-size: 12.5px;
		padding: 6px 0;
	}
	.ai-score {
		display: inline-block;
		border: 1px solid var(--accent-line);
		background: var(--accent-dim);
		color: var(--accent);
		border-radius: 99px;
		padding: 2px 9px;
		font-size: 12px;
		font-weight: 600;
		font-variant-numeric: tabular-nums;
		margin-bottom: 8px;
	}
	.ai-summary {
		font-size: 13.5px;
		line-height: 1.5;
		margin-bottom: 8px;
	}
	.ai-label {
		font-size: 11px;
		letter-spacing: 0.06em;
		text-transform: uppercase;
		font-weight: 600;
		color: var(--text2);
		margin: 10px 0 4px;
	}
	.ai-card ul {
		margin: 0;
		padding-left: 18px;
		color: var(--text2);
		font-size: 13px;
		line-height: 1.5;
	}
	.ai-settings {
		margin-top: 14px;
		border-top: 1px solid var(--border);
		padding-top: 10px;
		font-size: 12px;
		color: var(--text2);
	}
	.ai-settings summary {
		cursor: pointer;
		font-size: 12px;
		letter-spacing: 0.02em;
	}
	.ai-settings label {
		display: block;
		margin-top: 8px;
	}
	.ai-settings span {
		display: block;
		margin-bottom: 3px;
	}
	.ai-settings input {
		width: 100%;
		background: var(--surface2);
		border: 1px solid var(--border2);
		border-radius: 10px;
		color: var(--text);
		padding: 9px 10px;
		font-size: 14px;
	}
</style>
