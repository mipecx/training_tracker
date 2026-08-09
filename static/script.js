const CIRC = 2 * Math.PI * 78;
const DAYS  = ['Sun','Mon','Tue','Wed','Thu','Fri','Sat'];
const MONTHS = ['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'];
const ALL_DAYS = [
  { day: 'monday',    label: 'Mon', name: 'PUSH' },
  { day: 'wednesday', label: 'Wed', name: 'PULL' },
  { day: 'friday',    label: 'Fri', name: 'FULL' },
  { day: 'sunday',    label: 'Sun', name: 'LEGS+CORE' },
];

let workout = null;
let exIdx = 0, setIdx = 0, curSide = 'left';
let log = [];
let startTs = null;
let currentSessionId = null;
let prevSessionSets = {};

let restIv = null, restRem = 0, restTotal = 0, restCb = null;

function init() {
  const now = new Date();
  const hdr = document.getElementById('hdr-date');
  if (hdr) hdr.textContent = `${DAYS[now.getDay()]}, ${now.getDate()} ${MONTHS[now.getMonth()]}`;
  fetchToday();
}

async function fetchToday() {
  try {
    const d = await fetch('/api/today').then(r => r.json());
    renderHome(d);
  } catch { renderHome(null); }
}

function navTo(section) {
  document.querySelectorAll('.nav-btn').forEach(b => b.classList.remove('active'));
  document.getElementById('nav-' + section)?.classList.add('active');
  if (section === 'home') {
    showScreen('home');
    fetchToday();
  } else if (section === 'history') {
    showScreen('history');
    loadHistory();
  } else if (section === 'progress') {
    showScreen('progress');
    loadMetrics();
  }
}

async function renderHome(data) {
  const el = document.getElementById('home-body');
  if (!data || data.message) {
    el.innerHTML = `
      <div class="rest-day-card">
        <div class="workout-tag">Today</div>
        <div class="workout-name">Rest day</div>
        <p style="color:var(--text2);font-size:13px;margin-top:8px">Recovery is part of training.</p>
      </div>
      <div class="section-label">Other days</div>
      <div class="day-chips" id="chips"></div>`;
    renderChips('');
    return;
  }
  workout = data;

  const exercisesWithHistory = await Promise.all(data.exercises.map(async (e) => {
    const history = await fetch(`/api/history/${e.id}?limit=10`).then(r => r.json()).catch(() => []);
    const seen = new Set();
    const lastSets = history.filter(h => {
      const key = `${h.SessionID}_${h.SetNum}`;
      if (seen.has(key)) return false;
      seen.add(key);
      return true;
    }).slice(0, e.sets);
    return { ...e, lastSets };
  }));

  const listHTML = exercisesWithHistory.map(e => {
    const lvl = e.difficulty_level;
    const lvlMax = e.difficulty_max;
    const lvlBadge = lvl ? `<span class="lvl-badge">LVL ${lvl}/${lvlMax}</span>` : '';
    const prevStr = e.lastSets.length > 0
      ? e.lastSets.map(s => s.Reps ?? '—').join(' · ')
      : null;
    return `
      <div class="ex-preview-item">
        <div style="flex:1">
          <span class="ex-preview-name">${e.name}</span>
          ${lvlBadge}
          <span class="ex-preview-meta">${e.sets}×${e.reps_target}${prevStr ? ` <b style="color:var(--accent)">(${prevStr})</b>` : ''}</span>
        </div>
      </div>`;
  }).join('');

  el.innerHTML = `
    <div class="workout-card">
      <div class="workout-tag">Today</div>
      <div class="workout-name">${data.name}</div>
      <div class="ex-preview">${listHTML}</div>
      <button class="btn btn-accent" onclick="startWorkout()">Start workout</button>
    </div>
    <div class="section-label">Other days</div>
    <div class="day-chips" id="chips"></div>`;
  renderChips(data.day);
}

function renderChips(activeDay) {
  const el = document.getElementById('chips');
  if (!el) return;
  ALL_DAYS.filter(d => d.day !== activeDay).forEach(d => {
    const div = document.createElement('div');
    div.className = 'day-chip';
    div.innerHTML = `<div class="day-chip-name">${d.name}</div><div class="day-chip-day">${d.label}</div>`;
    div.onclick = async () => {
      const data = await fetch(`/api/workout/${d.day}`).then(r => r.json());
      workout = data;
      renderHome(data);
    };
    el.appendChild(div);
  });
}

async function startWorkout() {
  prevSessionSets = {};
  try {
    const sessions = await fetch('/api/sessions?limit=50').then(r => r.json());
    const prev = sessions.find(s => s.WorkoutID === workout.day && s.FinishedAt);
    if (prev) {
      const data = await fetch(`/api/sessions/${prev.ID}`).then(r => r.json());
      if (data.sets) {
        const seen = new Set();
        data.sets.forEach(s => {
          if (!prevSessionSets[s.ExerciseID]) prevSessionSets[s.ExerciseID] = [];
          const key = `${s.SetNum}_${s.Side || ''}`;
          if (!seen.has(key)) {
            seen.add(key);
            prevSessionSets[s.ExerciseID].push({ set_num: s.SetNum, reps: s.Reps, side: s.Side });
          }
        });
      }
    }
  } catch(e) {}

  try {
    const resp = await fetch('/api/sessions', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ workout_id: workout.day, workout_name: workout.name })
    }).then(r => r.json());
    currentSessionId = resp.session_id;
    exIdx = 0; setIdx = 0; curSide = 'left'; log = []; startTs = Date.now();
    showScreen('workout');
    renderEx();
  } catch(err) {
    alert('Failed to start workout');
  }
}

function renderEx() {
  const ex = workout.exercises[exIdx];
  const totalSets = workout.exercises.reduce((s, e) => s + e.sets, 0);
  const doneSets = workout.exercises.slice(0, exIdx).reduce((s, e) => s + e.sets, 0) + setIdx;
  const pct = totalSets ? doneSets / totalSets * 100 : 0;
  document.getElementById('prog-fill').style.width = pct + '%';
  document.getElementById('prog-txt').textContent = `${exIdx + 1}/${workout.exercises.length}`;

  const sideHTML = ex.per_side ? `
    <div class="side-toggle">
      <button class="side-btn ${curSide==='left'?'active':''}" onclick="setSide('left')">Left</button>
      <button class="side-btn ${curSide==='right'?'active':''}" onclick="setSide('right')">Right</button>
    </div>` : '';

  const setsHTML = Array.from({length: ex.sets}, (_, i) => {
    const done = i < setIdx;
    const current = i === setIdx;
    const entry = log.find(l => l.exercise_id === ex.id && l.set_num === i && (!ex.per_side || l.side === curSide));
    return `
      <div class="set-col">
        <div class="set-num">${i + 1}</div>
        <input class="set-inp ${done ? 'done' : ''}"
          id="si-${i}" type="number" inputmode="numeric"
          placeholder="${ex.reps_target}"
          value="${entry ? entry.reps : ''}"
          ${current ? '' : 'readonly'}
          onchange="saveReps(${i}, this.value)">
      </div>`;
  }).join('');

  const prevSets = prevSessionSets[ex.id] || [];
  const prevHTML = prevSets.length > 0 ? `
    <div class="prev-session-row">
      <span class="prev-session-lbl">Last time:</span>
      ${prevSets.map((s,i) => `<span class="prev-set-tag">${i+1}: <b>${s.reps ?? '—'}</b></span>`).join('')}
    </div>` : '';

  let diffHTML = '';
  if (ex.difficulty_level && ex.progression) {
    const lvl = ex.difficulty_level;
    const lvlMax = ex.difficulty_max || ex.progression.length;
    const threshold = ex.difficulty_threshold;
    const nextProg = ex.progression.find(p => p.level === lvl + 1);
    const pct = threshold ? Math.min(100, Math.round((parseInt(ex.reps_target) || threshold) / threshold * 100)) : 0;
    diffHTML = `
      <div class="diff-block">
        <div class="diff-header">
          <span class="diff-label">Level ${lvl}/${lvlMax}</span>
          <span class="diff-name">${ex.progression.find(p => p.level === lvl)?.name || ex.name}</span>
        </div>
        ${threshold ? `<div class="diff-progress">
          <div class="diff-prog-track"><div class="diff-prog-fill" style="width:${pct}%"></div></div>
          <span class="diff-prog-lbl">Next level at: ${threshold} reps</span>
        </div>` : ''}
        ${nextProg ? `<div class="next-lvl-hint">→ LVL ${lvl+1}: <b>${nextProg.name}</b> — ${nextProg.description}</div>` : '<div class="next-lvl-hint max-lvl">🏆 Max level</div>'}
      </div>`;
  }

  const techHTML = ex.technique && ex.technique.text ? `
    <div class="tech-toggle" onclick="toggleTech()">
      <div class="tech-icon">?</div>
      <span>How to perform</span>
    </div>
    <div class="tech-box" id="tech-box">${ex.technique.text}</div>` : '';

  document.getElementById('ex-wrap').innerHTML = `
    <div class="ex-card">
      <div class="ex-counter">Ex ${exIdx+1}/${workout.exercises.length} · Set ${setIdx+1}/${ex.sets}</div>
      <div class="ex-name">${ex.name}</div>
      <div class="meta-row">
        <div class="meta-item"><div class="meta-label">Reps</div><div class="meta-value">${ex.reps_target}</div></div>
        <div class="meta-item"><div class="meta-label">Rest</div><div class="meta-value">${fmtRest(ex.rest_sec)}</div></div>
        <div class="meta-item"><div class="meta-label">Sets</div><div class="meta-value">${ex.sets}</div></div>
      </div>
      ${ex.notes ? `<div class="notes-box">${ex.notes}</div>` : ''}
      ${prevHTML}
      ${sideHTML}
      <div class="sets-label">Reps</div>
      <div class="sets-row">${setsHTML}</div>
      ${diffHTML}
      ${techHTML}
    </div>`;

  const inp = document.getElementById(`si-${setIdx}`);
  if (inp) setTimeout(() => inp.focus(), 80);
}

async function saveReps(i, val) {
  if (!currentSessionId) return;
  const ex = workout.exercises[exIdx];
  const entry = {
    exercise_id: ex.id, exercise_name: ex.name,
    set_num: i, side: ex.per_side ? curSide : null,
    reps: parseInt(val) || 0, notes: ''
  };
  try {
    await fetch(`/api/sessions/${currentSessionId}/sets`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(entry)
    });
  } catch(e) {}

  const idx = log.findIndex(l =>
    l.exercise_id === entry.exercise_id && l.set_num === entry.set_num && l.side === entry.side);
  if (idx >= 0) log[idx] = entry; else log.push(entry);
}

function setSide(s) { curSide = s; renderEx(); }
function toggleTech() { document.getElementById('tech-box')?.classList.toggle('open'); }
function fmtRest(sec) {
  const m = Math.floor(sec / 60), s = sec % 60;
  return s ? `${m}m${s}s` : `${m} min`;
}

function doneSet() {
  const inp = document.getElementById(`si-${setIdx}`);
  if (inp?.value) saveReps(setIdx, inp.value);
  const ex = workout.exercises[exIdx];
  const lastSet = setIdx + 1 >= ex.sets;
  const lastEx  = exIdx + 1 >= workout.exercises.length;
  if (lastSet && lastEx) { showFinish(); return; }
  const nextLabel = lastSet
    ? `Next: <b>${workout.exercises[exIdx + 1].name}</b>`
    : `Set ${setIdx + 2}/${ex.sets} — <b>${ex.name}</b>`;
  document.getElementById('rest-next').innerHTML = nextLabel;
  startRest(ex.rest_sec, () => {
    if (lastSet) { exIdx++; setIdx = 0; curSide = 'left'; }
    else { setIdx++; }
    showScreen('workout');
    renderEx();
  });
}

function skipEx() {
  if (exIdx + 1 >= workout.exercises.length) { showFinish(); return; }
  exIdx++; setIdx = 0; curSide = 'left'; renderEx();
}

function startRest(sec, cb) {
  restCb = cb; restTotal = sec; restRem = sec;
  updateArc(); showScreen('rest');
  clearInterval(restIv);
  restIv = setInterval(() => {
    restRem--;
    updateArc();
    if (restRem <= 0) { clearInterval(restIv); setTimeout(() => restCb?.(), 300); }
  }, 1000);
}

function updateArc() {
  const m = Math.floor(restRem / 60), s = restRem % 60;
  const urgent = restRem <= 10 && restRem > 0;
  const el = document.getElementById('rest-time');
  el.textContent = `${m}:${s.toString().padStart(2,'0')}`;
  el.classList.toggle('urgent', urgent);
  const arc = document.getElementById('arc-fill');
  arc.style.strokeDasharray = CIRC;
  arc.style.strokeDashoffset = CIRC * (1 - (restTotal ? restRem / restTotal : 0));
  arc.classList.toggle('urgent', urgent);
}

function adjRest(d) { restRem = Math.max(0, restRem + d); restTotal = Math.max(restTotal, restRem); updateArc(); }
function skipRest() { clearInterval(restIv); restCb?.(); }

async function showFinish() {
  clearInterval(restIv);
  const durationSec = startTs ? Math.round((Date.now() - startTs) / 1000) : 0;
  try {
    await fetch(`/api/sessions/${currentSessionId}/finish`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ duration_sec: durationSec })
    });
  } catch(e) {}

  const mins = Math.round(durationSec / 60) || 0;
  const totalReps = log.reduce((s, l) => s + (parseInt(l.reps) || 0), 0);
  const byEx = {};
  log.forEach(l => { (byEx[l.exercise_name] = byEx[l.exercise_name] || []).push(l); });
  const summaryHTML = Object.entries(byEx).map(([name, entries]) => {
    const setsStr = entries.map(e => `${e.set_num+1}${e.side?` (${e.side==='left'?'L':'R'})`:''}: ${e.reps||'—'}`).join(' · ');
    return `<div class="summary-item"><span>${name}</span><span class="summary-sets">${setsStr}</span></div>`;
  }).join('');

  document.getElementById('finish-body').innerHTML = `
    <div class="finish-card">
      <div class="finish-title">Done</div>
      <div class="finish-sub">${workout.name}</div>
      <div class="finish-stats">
        <div class="finish-stat"><div class="finish-stat-val">${mins}</div><div class="finish-stat-lbl">min</div></div>
        <div class="finish-stat"><div class="finish-stat-val">${log.length}</div><div class="finish-stat-lbl">sets</div></div>
        <div class="finish-stat"><div class="finish-stat-val">${totalReps}</div><div class="finish-stat-lbl">reps</div></div>
      </div>
      ${summaryHTML}
    </div>`;
  showScreen('finish');
}

function goHome() {
  workout = null; exIdx = 0; setIdx = 0; log = []; startTs = null;
  document.querySelectorAll('.nav-btn').forEach(b => b.classList.remove('active'));
  document.getElementById('nav-home')?.classList.add('active');
  showScreen('home'); fetchToday();
}

async function loadHistory() {
  const el = document.getElementById('history-body');
  el.innerHTML = '<div class="loading-txt">Loading...</div>';
  try {
    const sessions = await fetch('/api/sessions?limit=100').then(r => r.json());
    const finished = sessions.filter(s => s.FinishedAt);
    if (!finished.length) { el.innerHTML = '<div class="empty-state">No completed workouts</div>'; return; }

    const byMonth = {};
    finished.forEach(s => {
      const d = new Date(s.Date);
      const key = `${d.getFullYear()}-${d.getMonth()}`;
      const label = `${['January','February','March','April','May','June','July','August','September','October','November','December'][d.getMonth()]} ${d.getFullYear()}`;
      if (!byMonth[key]) byMonth[key] = { label, sessions: [] };
      byMonth[key].sessions.push(s);
    });

    const typeColors = { monday:'#c8ff00', wednesday:'#4ade80', friday:'#60a5fa', sunday:'#f472b6' };
    el.innerHTML = Object.values(byMonth).map(({ label, sessions }) => `
      <div class="section-label">${label}</div>
      ${sessions.map(s => {
        const d = new Date(s.Date);
        const mins = Math.round(s.DurationSec / 60);
        const color = typeColors[s.WorkoutID] || '#888';
        return `<div class="history-item" onclick="openSession(${s.ID})">
          <div class="history-dot" style="background:${color}"></div>
          <div class="history-info">
            <div class="history-name">${s.WorkoutName}</div>
            <div class="history-date">${DAYS[d.getDay()]}, ${d.getDate()} ${MONTHS[d.getMonth()]}</div>
          </div>
          <div class="history-meta">
            <div class="history-dur">${mins} min</div>
            <svg width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><polyline points="9 18 15 12 9 6"/></svg>
          </div>
        </div>`;
      }).join('')}
    `).join('');
  } catch(e) {
    el.innerHTML = '<div class="empty-state">Failed to load</div>';
  }
}

let sessionChart = null;

async function openSession(id) {
  showScreen('session');
  const el = document.getElementById('session-body');
  el.innerHTML = '<div class="loading-txt">Loading...</div>';

  try {
    const [detail, compare] = await Promise.all([
      fetch(`/api/sessions/${id}`).then(r => r.json()),
      fetch(`/api/sessions/${id}/compare`).then(r => r.json())
    ]);

    const s = detail.session;
    const d = new Date(s.Date);
    const mins = Math.round(s.DurationSec / 60);

    const seen = new Set();
    const sets = (detail.sets || []).filter(s => {
      const key = `${s.ExerciseID}_${s.SetNum}_${s.Side||''}`;
      if (seen.has(key)) return false; seen.add(key); return true;
    });

    const byEx = {};
    sets.forEach(s => {
      if (!byEx[s.ExerciseID]) byEx[s.ExerciseID] = { name: s.ExerciseName, sets: [] };
      byEx[s.ExerciseID].sets.push(s);
    });

    const exListHTML = Object.entries(byEx).map(([exId, { name, sets }]) => {
      const totalReps = sets.reduce((a, s) => a + (s.Reps || 0), 0);
      const setsStr = sets.map((s, i) => `<span class="set-pill">${i+1}: ${s.Reps ?? '—'}</span>`).join('');
      return `<div class="session-ex-item">
        <div class="session-ex-header">
          <span class="session-ex-name">${name}</span>
          <span class="session-ex-vol">${totalReps} reps</span>
        </div>
        <div class="set-pills">${setsStr}</div>
      </div>`;
    }).join('');

    el.innerHTML = `
      <button class="btn btn-sm back-btn" onclick="navTo('history')">← Back</button>
      <div class="session-header-card">
        <div class="workout-tag">${DAYS[d.getDay()]}, ${d.getDate()} ${MONTHS[d.getMonth()]}</div>
        <div class="workout-name">${s.WorkoutName}</div>
        <div class="session-stat-row">
          <div class="session-stat"><div class="session-stat-val">${mins}</div><div class="session-stat-lbl">min</div></div>
          <div class="session-stat"><div class="session-stat-val">${sets.length}</div><div class="session-stat-lbl">sets</div></div>
          <div class="session-stat"><div class="session-stat-val">${sets.reduce((a,s)=>a+(s.Reps||0),0)}</div><div class="session-stat-lbl">reps</div></div>
        </div>
      </div>

      <div class="session-tabs">
        <button class="session-tab active" id="stab-ex" onclick="switchSessionTab('ex')">Exercises</button>
        <button class="session-tab" id="stab-chart" onclick="switchSessionTab('chart')">Chart</button>
      </div>

      <div id="stab-panel-ex">
        ${exListHTML}
      </div>

      <div id="stab-panel-chart" style="display:none">
        <div id="session-chart-wrap"></div>
      </div>
    `;

    window._compareData = compare;
    window._currentSessionSets = byEx;

  } catch(e) {
    el.innerHTML = '<div class="empty-state">Failed to load</div>';
  }
}

function switchSessionTab(tab) {
  document.querySelectorAll('.session-tab').forEach(t => t.classList.remove('active'));
  document.getElementById(`stab-${tab}`)?.classList.add('active');
  document.getElementById('stab-panel-ex').style.display = tab === 'ex' ? 'block' : 'none';
  document.getElementById('stab-panel-chart').style.display = tab === 'chart' ? 'block' : 'none';
  if (tab === 'chart') renderSessionChart();
}

function renderSessionChart() {
  const wrap = document.getElementById('session-chart-wrap');
  if (!window._compareData) return;

  const curr = window._compareData.current?.volume || [];
  const prev = window._compareData.previous?.volume || [];

  if (!curr.length) { wrap.innerHTML = '<div class="empty-state">No data</div>'; return; }

  const labels = curr.map(e => e.exercise_name.split(' ').slice(0,2).join(' '));
  const currVol = curr.map(e => (e.sets || 0) * Math.round(e.avg_reps || 0));
  const prevMap = {};
  prev.forEach(e => { prevMap[e.exercise_id] = (e.sets || 0) * Math.round(e.avg_reps || 0); });
  const prevVol = curr.map(e => prevMap[e.exercise_id] || 0);

  const hasPrev = window._compareData.previous?.session;

  const canvasHeight = Math.max(200, curr.length * 44 + 60);
  wrap.innerHTML = `
    <div class="chart-legend">
      <span class="leg-item"><span class="leg-dot" style="background:#333"></span>Previous</span>
      <span class="leg-item"><span class="leg-dot" style="background:#c8ff00"></span>Current</span>
    </div>
    <div style="position:relative;width:100%;height:${canvasHeight}px;margin-top:12px">
      <canvas id="session-chart" role="img" aria-label="Exercise volume comparison"></canvas>
    </div>
  `;

  if (sessionChart) { sessionChart.destroy(); sessionChart = null; }

  const ctx = document.getElementById('session-chart').getContext('2d');
  sessionChart = new Chart(ctx, {
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
          backgroundColor: currVol.map((v, i) => v >= prevVol[i] ? '#c8ff00' : '#ff4444'),
          borderColor: currVol.map((v, i) => v >= prevVol[i] ? '#a0cc00' : '#cc2222'),
          borderWidth: 1,
          borderRadius: 4,
        }
      ]
    },
    options: {
      indexAxis: 'y',
      responsive: true,
      maintainAspectRatio: false,
      plugins: {
        legend: { display: false },
        tooltip: {
          backgroundColor: '#1a1a1a', titleColor: '#f0f0f0', bodyColor: '#888',
          borderColor: '#2a2a2a', borderWidth: 1,
          callbacks: { label: ctx => `${ctx.dataset.label}: ${ctx.parsed.x} obj` }
        }
      },
      scales: {
        x: { ticks: { color: '#555', font: { family: 'JetBrains Mono', size: 10 } }, grid: { color: '#1a1a1a' }, beginAtZero: true },
        y: { ticks: { color: '#888', font: { family: 'JetBrains Mono', size: 10 }, autoSkip: false }, grid: { color: '#1a1a1a' } }
      }
    }
  });
}

function switchProgTab(tab) {
  document.querySelectorAll('.prog-tab').forEach(t => t.classList.remove('active'));
  document.getElementById(`tab-${tab === 'metrics' ? 'metrics' : 'history-p'}`)?.classList.add('active');
  document.getElementById('prog-metrics').style.display = tab === 'metrics' ? 'block' : 'none';
  document.getElementById('prog-history-p').style.display = tab === 'history' ? 'block' : 'none';
  if (tab === 'history') renderMetricsCharts();
}

function getMondayDate() {
  const dt = new Date();
  const day = dt.getDay();
  const diff = dt.getDate() - day + (day === 0 ? -6 : 1);
  dt.setDate(diff);
  return dt.toISOString().split('T')[0];
}

function fmtDateLabel(str) {
  const d = new Date(str);
  return `${d.getDate()} ${MONTHS[d.getMonth()]}`;
}

function calcBodyFat(weightKg, heightCm, waistCm, neckCm) {
  if (!heightCm || !waistCm || !neckCm) return null;
  const pct = 86.010 * Math.log10(waistCm - neckCm) - 70.041 * Math.log10(heightCm) + 36.76;
  return Math.max(0, Math.round(pct * 10) / 10);
}

function bfCategory(pct) {
  if (pct < 6)  return { label: 'Essential', color: '#60a5fa' };
  if (pct < 14) return { label: 'Athlete', color: '#4ade80' };
  if (pct < 18) return { label: 'Fitness', color: '#c8ff00' };
  if (pct < 25) return { label: 'Average', color: '#fb923c' };
  return { label: 'Obese', color: '#f87171' };
}

async function loadMetrics() {
  const formWrap = document.getElementById('metrics-form-wrap');
  const histWrap = document.getElementById('metrics-history-wrap');
  const weekDate = getMondayDate();

  const isSunday = new Date().getDay() === 0;
  const sundayNote = isSunday ? '<div class="sunday-badge">📋 Sunday — measurement day!</div>' : '';

  let existing = {};
  let allMetrics = [];
  try {
    allMetrics = await fetch('/api/metrics?limit=50').then(r => r.json());
    const thisWeek = allMetrics.find(m => m.week_date === weekDate);
    if (thisWeek) existing = thisWeek;
  } catch(e) {}

  const bf = calcBodyFat(existing.weight_kg, existing.height_cm, existing.waist_cm, existing.neck_cm);
  const bfCat = bf ? bfCategory(bf) : null;

  formWrap.innerHTML = `
    ${sundayNote}
    <div class="metrics-card">
      <div class="workout-tag">Week of ${fmtDateLabel(weekDate)}</div>
      <div class="m-grid">
        <div class="m-field"><label class="metric-label">Weight (kg)</label>
          <input class="metric-inp" id="m-weight" type="number" step="0.1" placeholder="78.0" value="${existing.weight_kg ?? ''}" oninput="updateBF()"></div>
        <div class="m-field"><label class="metric-label">Height (cm)</label>
          <input class="metric-inp" id="m-height" type="number" placeholder="181" value="${existing.height_cm ?? ''}" oninput="updateBF()"></div>
        <div class="m-field"><label class="metric-label">Waist (cm)</label>
          <input class="metric-inp" id="m-waist" type="number" step="0.5" placeholder="86" value="${existing.waist_cm ?? ''}" oninput="updateBF()"></div>
        <div class="m-field"><label class="metric-label">Neck (cm)</label>
          <input class="metric-inp" id="m-neck" type="number" step="0.5" placeholder="39" value="${existing.neck_cm ?? ''}" oninput="updateBF()"></div>
      </div>
      <div id="bf-result-wrap" style="margin-top:12px">
        ${bf ? renderBFResult(bf, existing.weight_kg) : '<div class="bf-placeholder">Enter measurements to calculate body fat %</div>'}
      </div>
      <div class="m-field" style="margin-top:10px">
        <label class="metric-label">Notes</label>
        <textarea class="metric-inp metric-notes" id="m-notes" placeholder="How was the week?">${existing.notes ?? ''}</textarea>
      </div>
      <button class="btn btn-accent" style="margin-top:14px" onclick="saveMetrics('${weekDate}')">Save</button>
    </div>
    <div id="save-status" style="text-align:center;min-height:20px;font-size:12px;color:var(--accent);margin-top:8px"></div>
  `;

  if (!allMetrics.length) { histWrap.innerHTML = ''; return; }
  const rows = [...allMetrics].reverse().map(m => {
    const bf2 = calcBodyFat(m.weight_kg, m.height_cm, m.waist_cm, m.neck_cm);
    const cat = bf2 ? bfCategory(bf2) : null;
    return `<div class="history-row">
      <div class="history-date">${fmtDateLabel(m.week_date)}</div>
      <div class="history-metrics">
        ${m.weight_kg != null ? `<span class="hm-tag">⚖️ ${m.weight_kg}kg</span>` : ''}
        ${m.waist_cm != null ? `<span class="hm-tag">📏 W:${m.waist_cm}cm</span>` : ''}
        ${m.neck_cm != null ? `<span class="hm-tag">📏 N:${m.neck_cm}cm</span>` : ''}
        ${bf2 != null ? `<span class="hm-tag" style="color:${cat.color}">${bf2}% ${cat.label}</span>` : ''}
        ${m.notes ? `<div class="hm-notes">${m.notes}</div>` : ''}
      </div>
    </div>`;
  }).join('');
  histWrap.innerHTML = `<div class="section-label" style="margin:16px 0 8px">History</div>${rows}`;
}

function renderBFResult(bf, weightKg) {
  const cat = bfCategory(bf);
  const fatKg = weightKg ? Math.round(weightKg * bf / 100 * 10) / 10 : null;
  return `<div class="bf-result" style="border-color:${cat.color}33">
    <div class="bf-num" style="color:${cat.color}">${bf}%</div>
    <div class="bf-meta">
      <b style="color:${cat.color}">${cat.label}</b>
      ${fatKg ? `Fat: ~${fatKg} kg` : ''}
    </div>
  </div>`;
}

function updateBF() {
  const w = parseFloat(document.getElementById('m-weight')?.value);
  const h = parseFloat(document.getElementById('m-height')?.value);
  const waist = parseFloat(document.getElementById('m-waist')?.value);
  const neck = parseFloat(document.getElementById('m-neck')?.value);
  const bf = calcBodyFat(w, h, waist, neck);
  const wrap = document.getElementById('bf-result-wrap');
  if (wrap) wrap.innerHTML = bf ? renderBFResult(bf, w) : '<div class="bf-placeholder">Enter measurements to calculate body fat %</div>';
}

async function saveMetrics(weekDate) {
  const weight = parseFloat(document.getElementById('m-weight').value) || null;
  const height = parseFloat(document.getElementById('m-height').value) || null;
  const waist = parseFloat(document.getElementById('m-waist').value) || null;
  const neck = parseFloat(document.getElementById('m-neck').value) || null;
  const notes = document.getElementById('m-notes').value.trim();
  const status = document.getElementById('save-status');
  try {
    await fetch('/api/metrics', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ week_date: weekDate, weight_kg: weight, height_cm: height, waist_cm: waist, neck_cm: neck, notes })
    });
    status.textContent = '✓ Saved';
    setTimeout(() => { status.textContent = ''; loadMetrics(); }, 1500);
  } catch(e) {
    status.style.color = 'var(--red)'; status.textContent = 'Error';
  }
}

let bfChart = null, weightChart = null;
async function renderMetricsCharts() {
  const wrap = document.getElementById('metrics-chart-wrap');
  try {
    const metrics = await fetch('/api/metrics?limit=20').then(r => r.json());
    const valid = metrics.filter(m => m.weight_kg && m.waist_cm && m.neck_cm && m.height_cm);
    if (valid.length < 2) { wrap.innerHTML = '<div class="empty-state">Need at least 2 weeks of data for the chart</div>'; return; }
    const labels = valid.map(m => fmtDateLabel(m.week_date));
    const weights = valid.map(m => m.weight_kg);
    const bfPcts = valid.map(m => calcBodyFat(m.weight_kg, m.height_cm, m.waist_cm, m.neck_cm));
    wrap.innerHTML = `
      <div class="section-label" style="margin-bottom:10px">Body weight</div>
      <div style="position:relative;width:100%;height:160px"><canvas id="weight-chart" role="img" aria-label="Body weight chart"></canvas></div>
      <div class="section-label" style="margin:16px 0 10px">Body fat % (US Navy)</div>
      <div style="position:relative;width:100%;height:160px"><canvas id="bf-chart" role="img" aria-label="Body fat percentage chart"></canvas></div>
    `;
    const opts = (color) => ({
      responsive: true, maintainAspectRatio: false,
      plugins: { legend: { display: false }, tooltip: { backgroundColor: '#1a1a1a', titleColor: '#f0f0f0', bodyColor: '#888', borderColor: '#2a2a2a', borderWidth: 1 } },
      scales: {
        x: { ticks: { color: '#555', font: { family: 'JetBrains Mono', size: 10 } }, grid: { color: '#1a1a1a' } },
        y: { ticks: { color: '#888', font: { family: 'JetBrains Mono', size: 10 } }, grid: { color: '#1a1a1a' } }
      }
    });
    if (weightChart) weightChart.destroy();
    weightChart = new Chart(document.getElementById('weight-chart').getContext('2d'), {
      type: 'line',
      data: { labels, datasets: [{ data: weights, borderColor: '#c8ff00', backgroundColor: 'rgba(200,255,0,0.06)', borderWidth: 2, pointBackgroundColor: '#c8ff00', pointRadius: 4, tension: 0.3, fill: true }] },
      options: opts('#c8ff00')
    });
    if (bfChart) bfChart.destroy();
    bfChart = new Chart(document.getElementById('bf-chart').getContext('2d'), {
      type: 'line',
      data: { labels, datasets: [{ data: bfPcts, borderColor: '#60a5fa', backgroundColor: 'rgba(96,165,250,0.06)', borderWidth: 2, pointBackgroundColor: '#60a5fa', pointRadius: 4, tension: 0.3, fill: true }] },
      options: opts('#60a5fa')
    });
  } catch(e) { wrap.innerHTML = '<div class="empty-state">Failed to load</div>'; }
}

function showScreen(name) {
  document.querySelectorAll('.screen').forEach(s => s.classList.remove('active'));
  document.getElementById(`screen-${name}`).classList.add('active');
}

async function getExHistory(exId) {
  try {
    const h = await fetch(`/api/history/${exId}?limit=1`).then(r => r.json());
    return h && h.length > 0 ? h[0].reps : null;
  } catch { return null; }
}

init();
