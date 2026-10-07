#!/usr/bin/env python3
"""Study planner: per-lecture spaced repetition, placed around your calendar.

Every lecture in the classes calendar becomes an item with its own review
ladder (encode the same day, then expanding intervals). Each run (daily)
turns what falls due in the next days into study sessions and places them
in free time, then creates them through OmaCal.

    study_planner.py plan [--dry-run] [--now ISO]   plan the next days
    study_planner.py rate START_MS TITLE GRADE      again|hard|good|easy
    study_planner.py setup                          interactive config
    study_planner.py status                         items and what's due

Config: ~/.config/study-pomodoro/planner.json (setup writes it).
State:  ~/.local/state/study-pomodoro/planner-state.json (the planner's memory),
        sessions.json (index the bar popup reads), ratings.json (from `rate`).

Only standard library; calendar access is the `omacal` CLI.
"""

import argparse
import datetime as dt
import json
import math
import os
import re
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

HOME = Path.home()
CONFIG_PATH = Path(os.environ.get("STUDY_PLANNER_CONFIG", HOME / ".config/study-pomodoro/planner.json"))
STATE_DIR = Path(os.environ.get("STUDY_STATE_DIR", HOME / ".local/state/study-pomodoro"))
STATE_PATH = STATE_DIR / "planner-state.json"
INDEX_PATH = STATE_DIR / "sessions.json"
RATINGS_PATH = STATE_DIR / "ratings.json"

POMODORO = 25          # minutes of work
POMODORO_BREAK = 5
GRADES = ("again", "hard", "good", "easy")
# Interval growth per grade after the second review (SM-2-like, simplified).
GRADE_MULT = {"again": 0.0, "hard": 1.5, "good": 3.0, "easy": 4.0}
# First interval after the same-day encoding review (days).
FIRST_INTERVAL = {"again": 1, "hard": 1, "good": 2, "easy": 3}
# Difficulty 1..5 → interval factor (harder = reviewed sooner) and practice/week.
DIFFICULTY_FACTOR = {1: 1.3, 2: 1.15, 3: 1.0, 4: 0.85, 5: 0.7}
PRACTICE_PER_WEEK = {1: 0, 2: 1, 3: 1, 4: 2, 5: 2}
RETIRE_AFTER_DAYS = 90       # stop reviewing once the interval passes this (no exam ahead)
AUTO_GRADE_AFTER_H = 20      # an unrated finished review counts as "good" after this
FREEZE_HOURS = 24            # sessions starting sooner than this are never moved
EXAM_WINDOW_DAYS = 21        # exam mode starts this many days before an exam
REVIEW_RESERVE_MINUTES = 45  # of each future day's cap, kept free of practice for reviews

# Title prefixes; the bar widget recognises sessions by these.
P_REVIEW, P_PRACTICE, P_SPACED, P_MIXED = "📝", "🎯", "🔄", "🔀"

DEFAULT_CONFIG = {
    "classes_calendar": "Universidad",
    "study_calendar": "Practicas",
    "exam_calendar": "Examenes",
    "day_start": "08:30",
    "day_end": "20:30",
    "morning_end": "13:00",
    "daily_cap_minutes": 180,
    "horizon_days": 7,
    # Same-day review length as a fraction of the lecture (¼: 2 h lecture → 1 pomodoro).
    "same_day_review_ratio": 0.25,
    "start_date": None,
    "courses": {},
}


# ---------------------------------------------------------------- utilities

def now_local() -> dt.datetime:
    return dt.datetime.now().replace(second=0, microsecond=0)


def parse_local(value: str) -> dt.datetime:
    """ISO string (with or without offset) → naive local datetime."""
    t = dt.datetime.fromisoformat(value)
    if t.tzinfo is not None:
        t = t.astimezone().replace(tzinfo=None)
    return t


def to_ms(t: dt.datetime) -> int:
    return int(t.astimezone().timestamp() * 1000)


def hhmm(value: str) -> dt.time:
    h, m = value.split(":")
    return dt.time(int(h), int(m))


def minutes_for(pomodoros: int) -> int:
    return pomodoros * POMODORO + (pomodoros - 1) * POMODORO_BREAK


def pomodoros_for(minutes: float, lo: int = 1, hi: int = 2) -> int:
    return max(lo, min(hi, int(minutes / POMODORO + 0.5)))


def monday_of(day: dt.date) -> dt.date:
    return day - dt.timedelta(days=day.weekday())


def load_json(path: Path, default):
    try:
        return json.loads(path.read_text())
    except (OSError, ValueError):
        return default


def save_json(path: Path, data) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    fd, tmp = tempfile.mkstemp(dir=path.parent, prefix="." + path.name, suffix=".tmp")
    with os.fdopen(fd, "w") as f:
        json.dump(data, f, indent=2, ensure_ascii=False, default=str)
    os.replace(tmp, path)


def overlaps(a_start, a_end, b_start, b_end) -> bool:
    return a_start < b_end and a_end > b_start


# ---------------------------------------------------------------- omacal

class OmacalError(RuntimeError):
    pass


class Omacal:
    def __init__(self):
        self.binary = shutil.which("omacal") or str(HOME / ".local/bin/omacal")
        if not Path(self.binary).exists():
            raise OmacalError("omacal is not installed — see https://omacal.app")
        self._calendars = None

    def run(self, *args):
        proc = subprocess.run([self.binary, *args, "--json"], capture_output=True, text=True, timeout=90)
        try:
            env = json.loads(proc.stdout or "{}")
        except ValueError:
            env = {}
        if proc.returncode == 0 and env.get("ok"):
            return env.get("data")
        msg = (env.get("error") or {}).get("message") or proc.stderr.strip() or f"exit {proc.returncode}"
        if proc.returncode == 3:
            msg = "OmaCal has no calendar data yet — open OmaCal and sign in"
        elif proc.returncode == 5:
            msg = "OmaCal is not running — start it so events can be created"
        raise OmacalError(f"omacal {' '.join(args[:2])}: {msg}")

    def calendars(self):
        if self._calendars is None:
            self._calendars = self.run("calendars") or []
        return self._calendars

    def resolve(self, ref):
        if ref in (None, ""):
            return None
        cals = self.calendars()
        text = str(ref).strip()
        if text.lower() == "primary":
            return next((c["id"] for c in cals if c.get("is_primary")), cals[0]["id"] if cals else None)
        for c in cals:
            if str(c["id"]) == text or str(c.get("summary", "")).lower() == text.lower():
                return c["id"]
        names = ", ".join(f"{c['id']}={c.get('summary')}" for c in cals)
        raise OmacalError(f"No OmaCal calendar {text!r}. Calendars: {names}")

    def events(self, start: dt.date, end: dt.date):
        return self.run("events", "list", "--from", start.isoformat(), "--to", end.isoformat()) or []

    def create(self, calendar_id, title, start: dt.datetime, end: dt.datetime, description=""):
        args = ["events", "create", "--title", title, "--date", start.date().isoformat(),
                "--start", start.strftime("%H:%M"), "--end", end.strftime("%H:%M"),
                "--calendar", str(calendar_id)]
        if description:
            args += ["--description", description]
        return self.run(*args)

    def delete(self, event_id, start_ms):
        return self.run("events", "delete", str(event_id), "--occurrence", str(start_ms))


# ---------------------------------------------------------------- model

def difficulty(config, course) -> int:
    c = config["courses"].get(course) or {}
    try:
        return max(1, min(5, int(c.get("difficulty", 3))))
    except (TypeError, ValueError):
        return 3


def exam_codes(config, course):
    c = config["courses"].get(course) or {}
    return [str(x) for x in c.get("exam_codes", []) if str(x).strip()]


def next_exam(config, course, exam_rows, today: dt.date):
    """Nearest exam date on/after today whose title carries one of the course's codes."""
    codes = exam_codes(config, course)
    if not codes:
        return None
    best = None
    for row in exam_rows:
        title = row.get("title") or ""
        if not any(re.search(r"(?<![\w])" + re.escape(code) + r"(?![\w])", title, re.I) for code in codes):
            continue
        day = parse_local(row["start"]).date()
        if day >= today and (best is None or day < best):
            best = day
    return best


def review_window(item, today: dt.date):
    """(earliest, ideal, latest) datetimes for an item's next review."""
    if item["rep"] == 0:
        end = parse_local(item["end"])
        latest = dt.datetime.combine(end.date() + dt.timedelta(days=1), dt.time(12, 0))
        return end + dt.timedelta(minutes=30), end + dt.timedelta(minutes=30), latest
    due = dt.date.fromisoformat(item["due"])
    interval = max(1.0, float(item["interval"]))
    early = math.floor(interval * 0.15)
    late = max(1, round(interval * 0.25))
    earliest = dt.datetime.combine(due - dt.timedelta(days=early), dt.time(0, 0))
    ideal = dt.datetime.combine(due, dt.time(12, 0))
    latest = dt.datetime.combine(due + dt.timedelta(days=late), dt.time(23, 59))
    return earliest, ideal, latest


def apply_review(item, grade: str, reviewed_at: dt.datetime, config, exam_day):
    """Advance an item's ladder after a review with `grade`."""
    dfac = DIFFICULTY_FACTOR[difficulty(config, item["course"])]
    if item["rep"] == 0:
        interval = FIRST_INTERVAL[grade] * (dfac if grade in ("good", "easy") else 1.0)
    elif grade == "again":
        interval = 1.0
    else:
        interval = max(1.0, float(item["interval"]) * GRADE_MULT[grade] * dfac)
    item["rep"] += 1 if grade != "again" or item["rep"] == 0 else 0
    item["interval"] = round(interval, 2)
    due = reviewed_at.date() + dt.timedelta(days=max(1, round(interval)))
    if exam_day and reviewed_at.date() < exam_day:
        # Keep a review close to the exam: never plan past two days before it.
        due = min(due, max(reviewed_at.date() + dt.timedelta(days=1), exam_day - dt.timedelta(days=2)))
    item["due"] = due.isoformat()
    item["history"].append({"at": reviewed_at.isoformat(), "grade": grade, "interval": item["interval"]})
    item["retired"] = interval > RETIRE_AFTER_DAYS and not exam_day


# ---------------------------------------------------------------- planner

class Planner:
    def __init__(self, config, state, omacal, now: dt.datetime, dry_run=False, out=print):
        self.config, self.state, self.omacal, self.now = config, state, omacal, now
        self.today = now.date()
        self.dry_run = dry_run
        self.out = out
        self.horizon_end = dt.datetime.combine(self.today + dt.timedelta(days=int(config["horizon_days"])), dt.time(23, 59))
        self.created, self.deleted, self.unplaced = [], [], []

    # -- calendar snapshot
    def load_calendar(self):
        c = self.config
        self.cal_classes = self.omacal.resolve(c["classes_calendar"])
        self.cal_study = self.omacal.resolve(c["study_calendar"])
        self.cal_exams = self.omacal.resolve(c.get("exam_calendar")) if c.get("exam_calendar") else None
        start = dt.date.fromisoformat(c["start_date"]) if c.get("start_date") else self.today
        frm = min(start, self.today - dt.timedelta(days=2))
        rows = self.omacal.events(frm, self.horizon_end.date() + dt.timedelta(days=1))
        exam_rows = self.omacal.events(self.today, self.today + dt.timedelta(days=200)) if self.cal_exams else []
        self.exam_rows = [r for r in exam_rows if r.get("calendarId") == self.cal_exams]
        self.lectures, self.study_rows, self.other_rows = [], [], []
        for r in rows:
            if r.get("allDay"):
                continue
            r = dict(r, _start=parse_local(r["start"]), _end=parse_local(r["end"]))
            cal = r.get("calendarId")
            if cal == self.cal_classes:
                self.lectures.append(r)
            elif cal == self.cal_study:
                self.study_rows.append(r)
            elif cal != self.cal_exams:
                self.other_rows.append(r)
        self.plan_start = start

    # -- items
    def sync_lectures(self):
        items = self.state["lectures"]
        seen = set()
        for r in self.lectures:
            if r["_start"].date() < self.plan_start:
                continue
            course = r.get("title") or "Class"
            key = f"{course}|{r['_start'].isoformat()}"
            seen.add(key)
            if key not in items:
                items[key] = {"course": course, "start": r["_start"].isoformat(), "end": r["_end"].isoformat(),
                              "rep": 0, "interval": 0, "due": r["_end"].date().isoformat(),
                              "history": [], "pending": None, "retired": False}
            if course not in self.config["courses"]:
                self.config["courses"][course] = {"difficulty": 3, "exam_codes": []}
                self.out(f"  new course {course!r}: difficulty 3, no exam code (edit with: study-pomodoro setup)")
        # A future lecture that vanished from the calendar was cancelled.
        for key in list(items):
            it = items[key]
            if it["rep"] == 0 and key not in seen and parse_local(it["start"]) > self.now \
                    and parse_local(it["start"]) <= self.horizon_end:
                del items[key]

    def exam_for(self, course):
        return next_exam(self.config, course, self.exam_rows, self.today)

    # -- sessions already planned
    def find_row(self, sess):
        start = parse_local(sess["start"])
        for r in self.study_rows:
            if r.get("title") == sess["title"] and abs((r["_start"] - start).total_seconds()) < 90:
                return r
        return None

    def reconcile(self):
        ratings = load_json(RATINGS_PATH, {})
        for sid, sess in self.state["sessions"].items():
            if sess["status"] not in ("planned", "done"):
                continue
            start, end = parse_local(sess["start"]), parse_local(sess["end"])
            row = self.find_row(sess)
            if row is None and sess["status"] == "planned":
                row = self.adopt_moved(sess, start)
                if row is not None:
                    start, end = row["_start"], row["_end"]
            if row is None:
                if sess["status"] == "planned":
                    # Deleted or moved by you: its material goes back in the queue.
                    sess["status"] = "missing"
                    self.release(sess)
                continue
            if sess["status"] == "planned" and end <= self.now:
                sess["status"] = "done"
            if sess["status"] == "planned" and start > self.now + dt.timedelta(hours=FREEZE_HOURS):
                clash = next((o for o in self.lectures + self.other_rows
                              if overlaps(start, end, o["_start"], o["_end"])), None)
                if clash:
                    self.remove_session(sid, sess, row, f"clashes with {clash.get('title')!r}")
                    continue
            if sess["status"] == "done":
                key = f"{to_ms(start)}|{sess['title']}"
                grade = (ratings.get(key) or {}).get("grade")
                if grade not in GRADES and self.now >= end + dt.timedelta(hours=AUTO_GRADE_AFTER_H):
                    grade = "good"
                if grade in GRADES:
                    self.finish(sess, grade, end)

    def adopt_moved(self, sess, start):
        """You dragged a session to another time: follow it instead of re-planning."""
        moved = [r for r in self.study_rows
                 if r.get("title") == sess["title"] and not self.known(r)
                 and abs((r["_start"] - start).total_seconds()) <= 7 * 86400]
        if not moved:
            return None
        row = min(moved, key=lambda r: abs((r["_start"] - start).total_seconds()))
        sess["start"], sess["end"] = row["_start"].isoformat(), row["_end"].isoformat()
        return row

    def release(self, sess):
        for key in sess.get("items", []):
            it = self.state["lectures"].get(key)
            if it and it.get("pending") == sess["id"]:
                it["pending"] = None

    def finish(self, sess, grade, when):
        sess["status"], sess["grade"] = "rated", grade
        for key in sess.get("items", []):
            it = self.state["lectures"].get(key)
            if it and it.get("pending") == sess["id"]:
                it["pending"] = None
                apply_review(it, grade, when, self.config, self.exam_for(it["course"]))

    def remove_session(self, sid, sess, row, why):
        self.out(f"  moving {sess['title']} ({sess['start'][:16]}): {why}")
        if not self.dry_run:
            self.omacal.delete(row["eventId"], row["startMs"])
        sess["status"] = "moved"
        self.release(sess)
        self.study_rows = [r for r in self.study_rows if r is not row]
        self.deleted.append(sess)

    # -- tasks
    def build_tasks(self):
        """Group due material into session-sized tasks."""
        tasks = []
        by_course_r2, spaced = {}, []
        for key, it in self.state["lectures"].items():
            if it.get("pending") or it.get("retired"):
                continue
            earliest, ideal, latest = review_window(it, self.today)
            if earliest > self.horizon_end:
                continue
            if latest < self.now:
                # Overdue: review as soon as possible.
                earliest, ideal, latest = self.now, self.now, self.now + dt.timedelta(days=2)
            if it["rep"] == 0:
                lecture_min = (parse_local(it["end"]) - parse_local(it["start"])).total_seconds() / 60
                n = pomodoros_for(lecture_min * float(self.config["same_day_review_ratio"]), 1, 3)
                tasks.append(self.task("review", it["course"], [key], earliest, ideal, latest, n,
                                       f"{P_REVIEW} Review: {it['course']}",
                                       "Same-day review of today's lecture: rewrite notes, fill gaps, "
                                       "recall 3-5 takeaways without looking."))
            elif it["rep"] == 1:
                by_course_r2.setdefault(it["course"], []).append((key, earliest, ideal, latest))
            else:
                spaced.append((key, it["course"], earliest, ideal, latest))

        for course, rows in by_course_r2.items():
            for group in cluster(rows):
                keys = [g[0] for g in group]
                e, i, l = window_of(group)
                n = pomodoros_for(15 * len(keys))
                tasks.append(self.task("review", course, keys, e, i, l, n,
                                       f"{P_REVIEW} Review: {course} ({len(keys)} lecture{'s' if len(keys) > 1 else ''})",
                                       "Second pass: close the notes, write down what you remember, then check. "
                                       + self.lecture_list(keys)))

        # Later reviews from different courses share a session (interleaving).
        spaced.sort(key=lambda s: s[3])
        while spaced:
            head = spaced.pop(0)
            group = [head]
            for s in list(spaced):
                if len(group) >= 6 or 10 * (len(group) + 1) > minutes_for(2):
                    break
                if s[2] <= head[4] and head[2] <= s[4]:
                    group.append(s)
                    spaced.remove(s)
            keys = [g[0] for g in group]
            courses = sorted({g[1] for g in group})
            e = max(g[2] for g in group)
            l = min(g[4] for g in group)
            if e > l:
                e, l = head[2], head[4]
            i = min(max(head[3], e), l)
            n = pomodoros_for(10 * len(keys))
            if len(courses) > 1:
                title = f"{P_MIXED} Interleaved review: " + " · ".join(courses)
                desc = "Switch subject every pomodoro; recall first, then check. "
            else:
                title = f"{P_SPACED} Spaced review: {courses[0]}"
                desc = "Recall from memory first (blank page / flashcards), then check. "
            tasks.append(self.task("spaced", courses[0] if len(courses) == 1 else None, keys, e, i, l, n,
                                   title, desc + self.lecture_list(keys)))

        tasks += self.practice_tasks()
        return tasks

    def practice_tasks(self):
        tasks = []
        weeks = {monday_of(self.today), monday_of(self.today) + dt.timedelta(days=7)}
        for week in sorted(weeks):
            week_end = dt.datetime.combine(week + dt.timedelta(days=6), dt.time(23, 59))
            for course in sorted(self.config["courses"]):
                lectures = sorted(parse_local(it["end"]) for it in self.state["lectures"].values()
                                  if it["course"] == course and monday_of(parse_local(it["start"]).date()) == week)
                exam = self.exam_for(course)
                days_to_exam = (exam - self.today).days if exam else None
                want = PRACTICE_PER_WEEK[difficulty(self.config, course)] if lectures else 0
                prep = exam is not None and 0 <= days_to_exam <= EXAM_WINDOW_DAYS \
                    and monday_of(exam) >= week >= monday_of(self.today)
                if prep:
                    want += 3 if days_to_exam <= 7 else 1
                have = self.count_practice(course, week)
                earliest = max(self.now, dt.datetime.combine(week, dt.time(0, 0)))
                if lectures:
                    # Practise after the week's first lecture has been reviewed.
                    earliest = max(earliest, lectures[0] + dt.timedelta(hours=2))
                latest = min(week_end, dt.datetime.combine(exam - dt.timedelta(days=1), dt.time(23, 59))) if prep else week_end
                if earliest > self.horizon_end or earliest > latest:
                    continue
                for k in range(max(0, want - have)):
                    title = f"{P_PRACTICE} Exam prep: {course}" if prep and k == 0 else f"{P_PRACTICE} Practice: {course}"
                    desc = ("Past exam questions under time pressure, then correct them." if "Exam prep" in title
                            else "Problems without looking at solutions first; log mistakes.")
                    ideal = earliest + (latest - earliest) * ((k + 1) / (max(1, want - have) + 1))
                    tasks.append(self.task("practice", course, [], earliest, ideal, latest, 2, title, desc,
                                           week=week.isoformat()))
        return tasks

    def count_practice(self, course, week):
        n = 0
        for sess in self.state["sessions"].values():
            if sess.get("kind") == "practice" and sess.get("course") == course \
                    and sess.get("week") == week.isoformat() and sess["status"] in ("planned", "done", "rated"):
                n += 1
        for r in self.study_rows:  # sessions from the old weekly planner
            if (r.get("title") or "") == f"{P_PRACTICE} Practice Problems: {course}" \
                    and monday_of(r["_start"].date()) == week and not self.known(r):
                n += 1
        return n

    def known(self, row):
        return any(s["title"] == row.get("title") and s["start"] == row["_start"].isoformat()
                   for s in self.state["sessions"].values())

    def lecture_list(self, keys):
        days = sorted({(it["course"], parse_local(it["start"]))
                       for it in (self.state["lectures"][k] for k in keys)}, key=lambda x: x[1])
        return "Lectures: " + ", ".join(f"{c} {t.strftime('%a %d %b')}" for c, t in days)

    def task(self, kind, course, items, earliest, ideal, latest, pomodoros, title, desc, week=None):
        return {"kind": kind, "course": course, "items": items, "earliest": earliest, "ideal": ideal,
                "latest": latest, "minutes": minutes_for(pomodoros), "title": title, "desc": desc,
                "week": week}

    # -- placement
    def busy_blocks(self, day):
        c = []
        for r in self.lectures:
            if r["_start"].date() == day:
                c.append((r["_start"] - dt.timedelta(minutes=30), r["_end"] + dt.timedelta(minutes=15)))
        for r in self.other_rows:
            if r["_start"].date() == day:
                c.append((r["_start"] - dt.timedelta(minutes=30), r["_end"] + dt.timedelta(minutes=30)))
        for r in self.study_rows:
            if r["_start"].date() == day:
                c.append((r["_start"] - dt.timedelta(minutes=15), r["_end"] + dt.timedelta(minutes=15)))
        return sorted(c)

    def study_minutes(self, day):
        return sum((r["_end"] - r["_start"]).total_seconds() / 60 for r in self.study_rows if r["_start"].date() == day)

    def courses_on(self, day):
        out = set()
        for r in self.study_rows:
            if r["_start"].date() == day:
                t = r.get("title") or ""
                for course in self.config["courses"]:
                    if course in t:
                        out.add(course)
        return out

    def candidates(self, task, slack_days=0):
        day_start, day_end = hhmm(self.config["day_start"]), hhmm(self.config["day_end"])
        not_before = self.now + dt.timedelta(minutes=30)
        latest = task["latest"] + dt.timedelta(days=slack_days)
        length = dt.timedelta(minutes=task["minutes"])
        day = max(task["earliest"].date(), self.today)
        while day <= min(latest.date(), self.horizon_end.date()):
            lo = max(dt.datetime.combine(day, day_start), task["earliest"], not_before)
            hi = min(dt.datetime.combine(day, day_end), max(latest, lo))
            blocks = self.busy_blocks(day)
            t = lo.replace(minute=(lo.minute // 15) * 15)
            if t < lo:
                t += dt.timedelta(minutes=15)
            while t + length <= dt.datetime.combine(day, day_end) and t <= hi:
                if not any(overlaps(t, t + length, b0, b1) for b0, b1 in blocks):
                    yield t
                t += dt.timedelta(minutes=15)
            day += dt.timedelta(days=1)

    def score(self, task, t):
        cap = float(self.config["daily_cap_minutes"])
        used = self.study_minutes(t.date())
        limit = cap
        if task["kind"] == "practice" and t.date() > self.today + dt.timedelta(days=1):
            # Reviews that fall due later (after you rate today's) need room
            # on those days too: practice planned ahead leaves a reserve.
            limit = cap - REVIEW_RESERVE_MINUTES
        if used + task["minutes"] > limit:
            return None
        s = abs((t - task["ideal"]).total_seconds()) / 3600.0
        if t > task["latest"]:
            s += 24 + (t - task["latest"]).total_seconds() / 3600.0 * 2
        morning = t.time() < hhmm(self.config["morning_end"])
        if task["kind"] == "practice":
            s += 0 if morning else 8
        elif morning:
            s += 1  # keep mornings for practice when possible
        s += 6 * used / cap                               # spread the load over the week
        if task["course"] and task["course"] in self.courses_on(t.date()):
            s += 10 if task["kind"] == "practice" else 4  # one session per course per day
        return s

    def place(self, tasks):
        order = {"review": 0, "practice": 1, "spaced": 2}
        tasks.sort(key=lambda k: (order[k["kind"]], k["latest"]))
        for task in tasks:
            best = None
            for slack in (0, 3):
                for t in self.candidates(task, slack):
                    s = self.score(task, t)
                    if s is not None and (best is None or s < best[0]):
                        best = (s, t)
                if best:
                    break
            if not best:
                self.unplaced.append(task)
                continue
            start = best[1]
            end = start + dt.timedelta(minutes=task["minutes"])
            self.commit(task, start, end)

    def commit(self, task, start, end):
        sid = f"{start.isoformat()}|{task['title']}"
        if not self.dry_run:
            self.omacal.create(self.cal_study, task["title"], start, end,
                               task["desc"] + "\n[study-planner]")
        sess = {"id": sid, "title": task["title"], "kind": task["kind"], "course": task["course"],
                "start": start.isoformat(), "end": end.isoformat(), "items": task["items"],
                "status": "planned", "grade": None, "week": task.get("week")}
        self.state["sessions"][sid] = sess
        for key in task["items"]:
            self.state["lectures"][key]["pending"] = sid
        # Later placements see it as busy.
        self.study_rows.append({"title": task["title"], "_start": start, "_end": end})
        self.created.append(sess)

    # -- run
    def run(self):
        self.load_calendar()
        self.sync_lectures()
        self.reconcile()
        tasks = self.build_tasks()
        self.place(tasks)
        return self


def cluster(rows):
    """Group (key, earliest, ideal, latest) rows whose windows overlap."""
    rows = sorted(rows, key=lambda r: r[3])
    groups, current = [], []
    for r in rows:
        if current and (r[1] > min(c[3] for c in current) or len(current) >= 4):
            groups.append(current)
            current = []
        current.append(r)
    if current:
        groups.append(current)
    return groups


def window_of(group):
    e = max(g[1] for g in group)
    l = min(g[3] for g in group)
    if e > l:
        e, l = group[0][1], group[0][3]
    i = min(max(min(g[2] for g in group), e), l)
    return e, i, l


def write_index(state):
    """What the bar popup needs: planner sessions and whether they take a rating."""
    rows = []
    for s in state["sessions"].values():
        if s["status"] in ("moved", "missing"):
            continue
        start, end = parse_local(s["start"]), parse_local(s["end"])
        rows.append({"start_ms": to_ms(start), "end_ms": to_ms(end), "title": s["title"],
                     "kind": s["kind"], "ratable": bool(s.get("items")), "grade": s.get("grade")})
    save_json(INDEX_PATH, {"version": 1, "sessions": rows})


# ---------------------------------------------------------------- commands

def load_config():
    if not CONFIG_PATH.exists():
        raise SystemExit(f"No planner config at {CONFIG_PATH}. Run: study-pomodoro setup")
    config = dict(DEFAULT_CONFIG)
    config.update(load_json(CONFIG_PATH, {}))
    config.setdefault("courses", {})
    return config


def load_state():
    state = load_json(STATE_PATH, {})
    state.setdefault("version", 1)
    state.setdefault("lectures", {})
    state.setdefault("sessions", {})
    return state


def cmd_plan(args):
    config = load_config()
    state = load_state()
    now = parse_local(args.now) if args.now else now_local()
    if not args.dry_run and subprocess.run(["pgrep", "-x", "omacal"], capture_output=True).returncode != 0:
        raise OmacalError("OmaCal is not running — start it, then plan again")
    known_courses = set(config["courses"])
    p = Planner(config, state, Omacal(), now, dry_run=args.dry_run).run()
    print(f"Planned {len(p.created)} session(s) through {p.horizon_end.date()}"
          + (" (dry run, nothing created)" if args.dry_run else ""))
    for s in sorted(p.created, key=lambda s: s["start"]):
        start = parse_local(s["start"])
        print(f"  + {start.strftime('%a %d %b %H:%M')}–{parse_local(s['end']).strftime('%H:%M')}  {s['title']}")
    for t in p.unplaced:
        print(f"  ! no room for {t['title']} (due by {t['latest'].strftime('%a %d %b')}) — "
              "raise daily_cap_minutes or free some time")
    if not args.dry_run:
        save_json(STATE_PATH, state)
        write_index(state)
        if set(config["courses"]) != known_courses:
            save_json(CONFIG_PATH, config)
    return 0


def cmd_rate(args):
    if args.grade not in GRADES:
        raise SystemExit(f"grade must be one of {', '.join(GRADES)}")
    ratings = load_json(RATINGS_PATH, {})
    ratings[f"{int(args.start_ms)}|{args.title}"] = {"grade": args.grade, "at": now_local().isoformat()}
    save_json(RATINGS_PATH, ratings)
    # Show it in the popup right away; the ladder moves on the next plan run.
    index = load_json(INDEX_PATH, {"version": 1, "sessions": []})
    for row in index.get("sessions", []):
        if row.get("start_ms") == int(args.start_ms) and row.get("title") == args.title:
            row["grade"] = args.grade
    save_json(INDEX_PATH, index)
    print(f"Rated {args.title}: {args.grade}")
    return 0


def cmd_status(args):
    config = load_config()
    state = load_state()
    today = now_local().date()
    items = state["lectures"].values()
    print(f"{len(items)} lectures tracked, {sum(1 for i in items if i.get('retired'))} retired")
    for course in sorted(config["courses"]):
        mine = [i for i in items if i["course"] == course and not i.get("retired")]
        due = sorted(i["due"] for i in mine if not i.get("pending"))
        d = difficulty(config, course)
        codes = ", ".join(exam_codes(config, course)) or "—"
        print(f"  {course}: difficulty {d}, exam code {codes}, {len(mine)} active, "
              f"next due {due[0] if due else '—'}")
    return 0


def ask(prompt, default=""):
    reply = input(f"  {prompt}{f' [{default}]' if default != '' else ''}: ").strip()
    return reply or str(default)


def cmd_setup(args):
    om = Omacal()
    config = dict(DEFAULT_CONFIG)
    config.update(load_json(CONFIG_PATH, {}))
    old = load_json(HOME / ".config/study_scheduler_config.json", {}).get("global_settings", {})
    print("OmaCal calendars:")
    for c in om.calendars():
        print(f"  {c['id']:>4}  {c.get('summary')}")
    config["classes_calendar"] = ask("Calendar with your lectures", config.get("classes_calendar") or old.get("calendar_id") or "")
    config["study_calendar"] = ask("Calendar to put study sessions in", config.get("study_calendar") or "")
    config["exam_calendar"] = ask("Calendar with exams (Enter to skip)", config.get("exam_calendar") or "") or None
    for key in ("classes_calendar", "study_calendar", "exam_calendar"):
        if config[key]:
            om.resolve(config[key])  # fail early on a typo
    config["daily_cap_minutes"] = int(ask("Max self-study minutes per day", config["daily_cap_minutes"]))
    config["day_start"] = ask("Earliest study time", config["day_start"])
    config["day_end"] = ask("Latest study time", config["day_end"])
    config["morning_end"] = ask("Mornings (practice time) end at", config["morning_end"])
    today = dt.date.today()
    default_start = config.get("start_date") or monday_of(today + dt.timedelta(days=7)).isoformat()
    config["start_date"] = ask("Track lectures from (YYYY-MM-DD)", default_start)

    cal_classes = om.resolve(config["classes_calendar"])
    rows = om.events(today - dt.timedelta(days=30), today + dt.timedelta(days=60))
    courses = sorted({r["title"] for r in rows if r.get("calendarId") == cal_classes and r.get("title")})
    if config["exam_calendar"]:
        cal_exams = om.resolve(config["exam_calendar"])
        past = om.events(today - dt.timedelta(days=400), today + dt.timedelta(days=200))
        titles = sorted({r["title"] for r in past if r.get("calendarId") == cal_exams and r.get("title")})
        if titles:
            print("\n  Exam titles seen so far (use the short code a course's exams carry):")
            print("    " + ", ".join(titles[:30]))
    print("\n  Difficulty 1 (easy) – 5 (hard): harder courses get more practice and shorter review intervals.")
    for course in courses:
        cur = config["courses"].get(course, {})
        print(f"\n  {course}")
        d = int(ask("difficulty 1-5", cur.get("difficulty", 3)))
        codes = ask("exam code(s), comma-separated (Enter if unknown)", ",".join(cur.get("exam_codes", [])))
        config["courses"][course] = {"difficulty": max(1, min(5, d)),
                                     "exam_codes": [c.strip() for c in codes.split(",") if c.strip()]}
    save_json(CONFIG_PATH, config)
    print(f"\n✓ Saved {CONFIG_PATH}. Preview with: study-pomodoro plan --dry-run")
    return 0


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = parser.add_subparsers(dest="cmd", required=True)
    p = sub.add_parser("plan")
    p.add_argument("--dry-run", action="store_true")
    p.add_argument("--now", help="pretend it is this local time (testing)")
    p.set_defaults(func=cmd_plan)
    r = sub.add_parser("rate")
    r.add_argument("start_ms")
    r.add_argument("title")
    r.add_argument("grade")
    r.set_defaults(func=cmd_rate)
    sub.add_parser("setup").set_defaults(func=cmd_setup)
    sub.add_parser("status").set_defaults(func=cmd_status)
    args = parser.parse_args(argv)
    try:
        return args.func(args)
    except OmacalError as e:
        print(f"✗ {e}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
