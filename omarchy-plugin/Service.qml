import QtQuick
import Quickshell
import Quickshell.Io
import "StudyModel.js" as Model

// The pomodoro timer, shared by every bar instance (one per monitor) so all
// screens count down together. Owns the `study-pomodoro` IPC target used by
// the keybindings and CLI. Calendar data is not here: the widget reads
// OmaCal's feed itself, the same way omacal.upcoming does.
Item {
  id: root

  // Host injection.
  property var shell: null
  property var manifest: null

  readonly property string home: Quickshell.env("HOME")
  readonly property string stateDir: home + "/.local/state/study-pomodoro"

  // ---- Settings, pushed in by the bar widget from its shell.json entry.
  property var cfg: Model.timerSettings({})
  property bool notify: true

  function configure(settings) {
    var next = Model.timerSettings(settings)
    var prev = cfg
    cfg = next
    notify = !(settings && settings.notify === false)
    // An untouched idle timer follows new durations; a started one keeps its time.
    if (!timer.running && timer.remainingMs === Model.phaseDurationMs(timer.phase, prev))
      setTimer(Model.reset(timer, cfg))
  }

  // ---- Pomodoro state
  property var timer: Model.initialState(cfg)
  property real now: Date.now()
  property bool stateLoaded: false

  readonly property bool running: timer.running
  readonly property string phase: timer.phase
  readonly property string phaseLabel: Model.phaseLabel(timer.phase)
  readonly property real remainingMs: Model.remainingMs(timer, now)
  readonly property string clockText: Model.formatClock(remainingMs)
  readonly property real progress: Model.progress(timer, cfg, now)
  readonly property int completedCycles: timer.cycle
  readonly property int totalCycles: cfg.cycles
  readonly property string nextPhaseLabel: Model.phaseLabel(Model.advance(timer, cfg, now).phase)
  // Idle = never started in this phase; the pill shows only its icon then.
  readonly property bool idle: !timer.running && timer.remainingMs === Model.phaseDurationMs(timer.phase, cfg)

  function setTimer(next) {
    timer = next
    now = Date.now()
    if (stateLoaded) stateFile.setText(JSON.stringify(timer) + "\n")
  }

  function toggle() { setTimer(Model.toggle(timer, Date.now())) }
  function reset() { setTimer(Model.reset(timer, cfg)) }
  function skip() { setTimer(Model.advance(timer, cfg, Date.now())) }

  function completePhase() {
    var finished = timer.phase
    setTimer(Model.advance(timer, cfg, Date.now()))
    if (!notify) return
    var headline = finished === "focus" ? "Focus session done" : "Break over"
    var body = finished === "focus"
      ? phaseLabel + " — " + Math.round(Model.phaseDurationMs(timer.phase, cfg) / 60000) + " min"
      : "Back to focus"
    Quickshell.execDetached(["omarchy-notification-send", "-g", "󱎫", "-u", "normal", headline, body])
  }

  function statusText() {
    return phaseLabel + " " + clockText + (running ? " running" : " paused")
      + " (" + completedCycles + "/" + totalCycles + ")"
  }

  Timer {
    interval: 1000
    repeat: true
    running: root.timer.running
    onTriggered: {
      root.now = Date.now()
      if (root.remainingMs <= 0) root.completePhase()
    }
  }

  // Timer state survives shell restarts and plugin hot-reloads.
  FileView {
    id: stateFile
    path: root.stateDir + "/timer.json"
    atomicWrites: true
    printErrors: false
    onLoaded: root.restore(text())
    onLoadFailed: root.restore("")
  }

  function restore(raw) {
    var restored = Model.restoreState(raw, cfg)
    // A phase that ended while the shell was down: move on, but don't auto-run.
    if (restored.running && restored.endsAt <= Date.now()) {
      restored = Model.advance(restored, cfg, Date.now())
      restored = Model.pause(restored, Date.now())
    }
    timer = restored
    now = Date.now()
    stateLoaded = true
  }

  Process {
    id: mkdirProc
    command: ["mkdir", "-p", root.stateDir]
    onExited: stateFile.reload()
  }

  Component.onCompleted: mkdirProc.running = true

  IpcHandler {
    target: "study-pomodoro"
    function toggle(): string { root.toggle(); return root.statusText() }
    function reset(): string { root.reset(); return root.statusText() }
    function skip(): string { root.skip(); return root.statusText() }
    function status(): string { return root.statusText() }
  }
}
