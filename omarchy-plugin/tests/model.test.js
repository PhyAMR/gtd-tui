// Run with: node omarchy-plugin/tests/model.test.js
// StudyModel.js starts with a QML-only `.pragma library` line, so load it as text.
const fs = require("fs")
const path = require("path")
const assert = require("assert")

const src = fs.readFileSync(path.join(__dirname, "..", "StudyModel.js"), "utf8").replace(/^\.pragma library\s*/, "")
const mod = { exports: {} }
new Function("module", src)(mod)
const M = mod.exports

const cfg = M.timerSettings({ focusMins: 25, shortBreakMins: 5, longBreakMins: 15, cycles: 4 })

// Settings clamp and default.
assert.deepStrictEqual(M.timerSettings({}), cfg)
assert.strictEqual(M.timerSettings({ focusMins: "0" }).focusMins, 1)

// start/pause keeps remaining time.
let s = M.initialState(cfg)
assert.strictEqual(M.remainingMs(s, 0), 25 * 60000)
s = M.start(s, 1000)
assert.strictEqual(M.remainingMs(s, 1000 + 60000), 24 * 60000)
s = M.pause(s, 1000 + 60000)
assert.strictEqual(s.running, false)
assert.strictEqual(M.remainingMs(s, 999999999), 24 * 60000)
assert.strictEqual(M.reset(s, cfg).remainingMs, 25 * 60000)

// Cycle: focus -> short x3, 4th focus -> long, long -> focus with cycle 0.
s = M.initialState(cfg)
const seen = []
for (let i = 0; i < 8; i++) { s = M.advance(s, cfg, 0); seen.push(s.phase) }
assert.deepStrictEqual(seen, ["short", "focus", "short", "focus", "short", "focus", "long", "focus"])
assert.strictEqual(s.cycle, 0)

// Running carries into the next phase.
s = M.advance(M.start(M.initialState(cfg), 0), cfg, 5000)
assert.strictEqual(s.running, true)
assert.strictEqual(s.endsAt, 5000 + 5 * 60000)

// Restore rejects junk, keeps valid state.
assert.deepStrictEqual(M.restoreState("nope", cfg), M.initialState(cfg))
assert.strictEqual(M.restoreState({ phase: "long", running: false, remainingMs: 1000, cycle: 9 }, cfg).cycle, 3)

// Formatting.
assert.strictEqual(M.formatClock(25 * 60000), "25:00")
assert.strictEqual(M.formatClock(59001), "01:00")
assert.strictEqual(M.formatClock(3600000 + 5000), "1:00:05")
assert.strictEqual(M.cleanTitle("📝 Review: Calculus"), "Review: Calculus")
assert.strictEqual(M.cleanTitle("🔀 Integrated Review"), "Integrated Review")

// Study sessions out of an OmaCal feed: prefix filter, today only, dedupe, sort.
const t = (hhmm) => Date.parse("2026-09-29T" + hhmm + ":00")
const ev = (title, a, b, extra) => Object.assign({ title, start_ms: t(a), end_ms: t(b), all_day: false, color: "#abc" }, extra || {})
const feed = {
  events: [ev("🎯 Practice Problems: Physics II", "12:00", "13:00"), ev("Dentist", "15:00", "16:00")],
  panel: {
    events: [ev("📝 Review: Calculus I", "09:00", "10:00"), ev("🎯 Practice Problems: Physics II", "12:00", "13:00")],
    agenda_days: [{ date_label: "Today", events: [
      ev("🔄 Spaced Review: Last Week", "14:00", "15:00"),
      ev("🔀 Integrated Review", "09:00", "10:00", { start_ms: t("09:00") + 86400000, end_ms: t("10:00") + 86400000 }),
      ev("📝 Holiday", "00:00", "23:59", { all_day: true }),
      ev("Dentist", "15:00", "16:00")
    ] }]
  }
}
const now = t("12:30")
const today = M.todaySessions(feed, now)
assert.deepStrictEqual(today.map(x => x.kind), ["review", "practice", "spaced"])
assert.strictEqual(today[0].color, "#abc")
assert.deepStrictEqual(M.sessionStates(today, now), ["past", "current", "later"])
assert.deepStrictEqual(M.sessionStates(today, t("11:00")), ["past", "next", "later"])
assert.deepStrictEqual(M.todaySessions(null, now), [])

// OmaCal's own agenda keeps everything except the study sessions.
const stripped = M.withoutStudy(feed.panel)
assert.deepStrictEqual(stripped.events, [])
assert.deepStrictEqual(stripped.agenda_days[0].events.map(e => e.title), ["📝 Holiday", "Dentist"])
assert.strictEqual(feed.panel.events.length, 2, "input panel is not mutated")

console.log("model tests: ok")
