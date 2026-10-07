.pragma library

// Pure helpers for the Study Pomodoro plugin. Kept free of QML types so they
// can be unit-tested with plain node (see tests/model.test.js).

var PHASES = ["focus", "short", "long"]

function clampInt(value, fallback, min, max) {
  var n = parseInt(value, 10)
  if (isNaN(n)) n = fallback
  return Math.max(min, Math.min(max, n))
}

// Normalized timer settings from the widget's shell.json entry.
function timerSettings(settings) {
  var s = settings || {}
  return {
    focusMins: clampInt(s.focusMins, 25, 1, 180),
    shortBreakMins: clampInt(s.shortBreakMins, 5, 1, 60),
    longBreakMins: clampInt(s.longBreakMins, 15, 1, 120),
    cycles: clampInt(s.cycles, 4, 1, 12)
  }
}

function phaseDurationMs(phase, cfg) {
  if (phase === "short") return cfg.shortBreakMins * 60000
  if (phase === "long") return cfg.longBreakMins * 60000
  return cfg.focusMins * 60000
}

function phaseLabel(phase) {
  if (phase === "short") return "Short Break"
  if (phase === "long") return "Long Break"
  return "Focus"
}

function initialState(cfg) {
  return {
    phase: "focus",
    running: false,
    endsAt: 0,
    remainingMs: phaseDurationMs("focus", cfg),
    cycle: 0
  }
}

// Validate persisted state; anything malformed falls back to a fresh timer.
function restoreState(raw, cfg) {
  var s = raw
  if (typeof raw === "string") {
    try { s = JSON.parse(raw) } catch (e) { s = null }
  }
  if (!s || PHASES.indexOf(s.phase) === -1) return initialState(cfg)
  var out = {
    phase: s.phase,
    running: s.running === true,
    endsAt: Number(s.endsAt) || 0,
    remainingMs: Number(s.remainingMs) || 0,
    cycle: clampInt(s.cycle, 0, 0, cfg.cycles - 1)
  }
  if (!out.running && out.remainingMs <= 0) out.remainingMs = phaseDurationMs(out.phase, cfg)
  return out
}

function remainingMs(state, now) {
  if (!state.running) return Math.max(0, state.remainingMs)
  return Math.max(0, state.endsAt - now)
}

function start(state, now) {
  if (state.running) return state
  return {
    phase: state.phase, cycle: state.cycle, running: true,
    endsAt: now + Math.max(0, state.remainingMs), remainingMs: 0
  }
}

function pause(state, now) {
  if (!state.running) return state
  return {
    phase: state.phase, cycle: state.cycle, running: false,
    endsAt: 0, remainingMs: remainingMs(state, now)
  }
}

function toggle(state, now) {
  return state.running ? pause(state, now) : start(state, now)
}

function reset(state, cfg) {
  return {
    phase: state.phase, cycle: state.cycle, running: false,
    endsAt: 0, remainingMs: phaseDurationMs(state.phase, cfg)
  }
}

// Same cycle as internal/pomodoro: focus -> short break, and after `cycles`
// focus rounds a long break; any break -> focus. The running flag carries
// over, so a running timer rolls straight into the next phase.
function advance(state, cfg, now) {
  var phase, cycle = state.cycle
  if (state.phase === "focus") {
    cycle += 1
    if (cycle >= cfg.cycles) {
      phase = "long"
      cycle = 0
    } else {
      phase = "short"
    }
  } else {
    phase = "focus"
  }
  var dur = phaseDurationMs(phase, cfg)
  return {
    phase: phase, cycle: cycle, running: state.running,
    endsAt: state.running ? now + dur : 0,
    remainingMs: state.running ? 0 : dur
  }
}

function formatClock(ms) {
  var total = Math.ceil(Math.max(0, ms) / 1000)
  var h = Math.floor(total / 3600)
  var m = Math.floor((total % 3600) / 60)
  var s = total % 60
  var mm = (m < 10 ? "0" : "") + m
  var ss = (s < 10 ? "0" : "") + s
  return h > 0 ? h + ":" + mm + ":" + ss : mm + ":" + ss
}

function progress(state, cfg, now) {
  var dur = phaseDurationMs(state.phase, cfg)
  if (dur <= 0) return 0
  return Math.max(0, Math.min(1, 1 - remainingMs(state, now) / dur))
}

// ---- Study sessions, read out of OmaCal's upcoming feed
// (~/.local/state/omacal/upcoming.json, see OmaCal's src-tauri/src/upcoming.rs).
// study_scheduler.py marks every session it creates with a title prefix.

var KINDS = [
  { prefix: "\uD83D\uDCDD", kind: "review", hint: "Rewrite notes, fill gaps, active recall" },          // 📝
  { prefix: "\uD83C\uDFAF", kind: "practice", hint: "Attempt problems, check solutions, fix errors" },  // 🎯
  { prefix: "\uD83D\uDD04", kind: "spaced", hint: "Flashcards and summary sheets" },                   // 🔄
  { prefix: "\uD83D\uDD00", kind: "integrated", hint: "Interleave subjects, one per pomodoro" }         // 🔀
]

var KIND_GLYPHS = {
  review: "󰏫",
  practice: "󰓾",
  spaced: "󰑓",
  integrated: "󰒝",
  other: "󰃭"
}

function kindGlyph(kind) {
  return KIND_GLYPHS[kind] || KIND_GLYPHS.other
}

function studyKind(title) {
  var t = String(title || "")
  for (var i = 0; i < KINDS.length; i++)
    if (t.indexOf(KINDS[i].prefix) === 0) return KINDS[i]
  return null
}

function isStudyEvent(ev) {
  return !!ev && !ev.all_day && studyKind(ev.title) !== null
}

// Every feed event the popup could know about: the forward-looking list,
// the complete day snapshot, and the agenda days (OmaCal fills whichever its
// version supports).
function feedEvents(feed) {
  var out = []
  if (!feed) return out
  function add(list) { if (Array.isArray(list)) for (var i = 0; i < list.length; i++) out.push(list[i]) }
  add(feed.events)
  if (feed.panel) {
    add(feed.panel.events)
    if (Array.isArray(feed.panel.agenda_days))
      for (var d = 0; d < feed.panel.agenda_days.length; d++) add(feed.panel.agenda_days[d].events)
  }
  return out
}

function dayBounds(now) {
  var d = new Date(now)
  var start = new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime()
  var end = new Date(d.getFullYear(), d.getMonth(), d.getDate() + 1).getTime()
  return { start: start, end: end }
}

// Today's study sessions (local day), deduplicated and in time order.
function todaySessions(feed, now) {
  var day = dayBounds(now)
  var seen = {}
  var out = []
  var events = feedEvents(feed)
  for (var i = 0; i < events.length; i++) {
    var ev = events[i]
    if (!isStudyEvent(ev)) continue
    if (!(ev.start_ms < day.end && ev.end_ms > day.start)) continue
    var key = ev.title + "|" + ev.start_ms
    if (seen[key]) continue
    seen[key] = true
    var k = studyKind(ev.title)
    out.push({
      title: String(ev.title),
      kind: k.kind,
      description: k.hint,
      start: ev.start_ms,
      end: ev.end_ms,
      color: ev.color || "",
      event: ev
    })
  }
  out.sort(function(a, b) { return a.start - b.start })
  return out
}

// The planner's index (~/.local/state/study-pomodoro/sessions.json) entry for
// a session: whether it takes a rating, and the rating it already has.
function indexEntry(index, session) {
  var rows = index && Array.isArray(index.sessions) ? index.sessions : []
  if (!session) return null
  for (var i = 0; i < rows.length; i++)
    if (rows[i].start_ms === session.start && rows[i].title === session.title) return rows[i]
  return null
}

var GRADES = ["again", "hard", "good", "easy"]

function withoutStudyList(list) {
  if (!Array.isArray(list)) return list
  return list.filter(function(ev) { return !isStudyEvent(ev) })
}

// A copy of the feed's panel with study sessions taken out, for OmaCal's own
// agenda sections, so a session is listed once (in STUDY SESSIONS).
function withoutStudy(panel) {
  if (!panel) return panel
  var copy = {}
  for (var k in panel) copy[k] = panel[k]
  copy.events = withoutStudyList(panel.events)
  if (Array.isArray(panel.agenda_days)) {
    copy.agenda_days = panel.agenda_days.map(function(day) {
      var d = {}
      for (var k2 in day) d[k2] = day[k2]
      d.events = withoutStudyList(day.events)
      return d
    })
  }
  return copy
}

// "past" | "current" | "next" | "later" for each session relative to now.
function sessionStates(sessions, now) {
  var out = []
  var nextAssigned = false
  for (var i = 0; i < sessions.length; i++) {
    var s = sessions[i]
    var state
    if (s.end <= now) state = "past"
    else if (s.start <= now) state = "current"
    else if (!nextAssigned) { state = "next"; nextAssigned = true }
    else state = "later"
    if (state === "current") nextAssigned = true
    out.push(state)
  }
  return out
}

function pad2(n) { return (n < 10 ? "0" : "") + n }

function formatTime(ms) {
  var d = new Date(ms)
  return pad2(d.getHours()) + ":" + pad2(d.getMinutes())
}

function formatRange(session) {
  return formatTime(session.start) + " – " + formatTime(session.end)
}

// Strip the planner's leading emoji ("📝 Review: X" -> "Review: X").
function cleanTitle(title) {
  return String(title || "").replace(/^[^A-Za-z0-9À-ɏ]+/, "").trim() || String(title || "")
}

if (typeof module !== "undefined") {
  module.exports = {
    timerSettings: timerSettings, phaseDurationMs: phaseDurationMs, phaseLabel: phaseLabel,
    initialState: initialState, restoreState: restoreState, remainingMs: remainingMs,
    start: start, pause: pause, toggle: toggle, reset: reset, advance: advance,
    formatClock: formatClock, progress: progress, kindGlyph: kindGlyph,
    studyKind: studyKind, isStudyEvent: isStudyEvent, todaySessions: todaySessions,
    withoutStudy: withoutStudy, sessionStates: sessionStates, indexEntry: indexEntry,
    formatTime: formatTime, formatRange: formatRange, cleanTitle: cleanTitle
  }
}
