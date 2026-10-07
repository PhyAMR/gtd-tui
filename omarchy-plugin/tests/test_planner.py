"""Planner tests against an in-memory calendar.  Run: python3 -m unittest discover tests"""
import datetime as dt
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "planner"))
import study_planner as sp  # noqa: E402

CLASSES, STUDY, EXAMS, OTHER = 9, 10, 5, 4


class FakeOmacal:
    """Enough of the omacal CLI for the planner: list, create, delete."""

    def __init__(self):
        self.rows = []
        self.next_id = 1

    def add(self, cal, title, start, end, all_day=False):
        self.rows.append({"eventId": self.next_id, "calendarId": cal, "title": title,
                          "start": start.isoformat(), "end": end.isoformat(),
                          "startMs": sp.to_ms(start), "allDay": all_day})
        self.next_id += 1

    def resolve(self, ref):
        return {"Universidad": CLASSES, "Practicas": STUDY, "Examenes": EXAMS}.get(ref)

    def events(self, start, end):
        lo = dt.datetime.combine(start, dt.time(0))
        hi = dt.datetime.combine(end + dt.timedelta(days=1), dt.time(0))
        return [dict(r) for r in self.rows if lo <= sp.parse_local(r["start"]) < hi]

    def create(self, cal, title, start, end, description=""):
        self.add(cal, title, start, end)

    def delete(self, event_id, start_ms):
        self.rows = [r for r in self.rows if r["eventId"] != event_id]

    def study(self):
        return sorted((sp.parse_local(r["start"]), sp.parse_local(r["end"]), r["title"])
                      for r in self.rows if r["calendarId"] == STUDY)


def at(day, hm):
    h, m = map(int, hm.split(":"))
    return dt.datetime.combine(day, dt.time(h, m))


MON = dt.date(2026, 10, 5)


def config(**over):
    c = dict(sp.DEFAULT_CONFIG)
    c.update({"start_date": MON.isoformat(),
              "courses": {"Game theory": {"difficulty": 3, "exam_codes": ["GT"]},
                          "Quantum computing": {"difficulty": 5, "exam_codes": []}}})
    c.update(over)
    return c


def week_of_lectures(om, weeks=1):
    for w in range(weeks):
        d = MON + dt.timedelta(days=7 * w)
        om.add(CLASSES, "Game theory", at(d, "16:30"), at(d, "18:30"))
        om.add(CLASSES, "Game theory", at(d + dt.timedelta(days=2), "10:30"), at(d + dt.timedelta(days=2), "12:30"))
        om.add(CLASSES, "Quantum computing", at(d + dt.timedelta(days=1), "12:30"), at(d + dt.timedelta(days=1), "14:30"))


class PlannerTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        sp.RATINGS_PATH = Path(self.tmp.name) / "ratings.json"
        self.om = FakeOmacal()
        self.state = {"version": 1, "lectures": {}, "sessions": {}}

    def tearDown(self):
        self.tmp.cleanup()

    def plan(self, now, cfg=None):
        return sp.Planner(cfg or config(), self.state, self.om, now, out=lambda *_: None).run()

    def test_same_day_review_after_each_lecture(self):
        week_of_lectures(self.om)
        self.plan(at(MON, "07:00"))
        reviews = [s for s in self.om.study() if s[2].startswith("📝 Review")]
        # Monday 16:30-18:30 lecture → reviewed that evening, after a 30 min gap.
        mon = [s for s in reviews if s[0].date() == MON]
        self.assertEqual(len(mon), 1)
        self.assertGreaterEqual(mon[0][0], at(MON, "19:00"))
        self.assertLessEqual(mon[0][1], at(MON, "20:30"))
        # 2 h lecture → a quarter of it, rounded to pomodoros: 1 pomodoro.
        self.assertEqual(mon[0][1] - mon[0][0], dt.timedelta(minutes=25))

    def test_everything_within_study_hours_and_cap(self):
        week_of_lectures(self.om)
        self.plan(at(MON, "07:00"), config(daily_cap_minutes=90))
        per_day = {}
        for s, e, _ in self.om.study():
            self.assertGreaterEqual(s.time(), dt.time(8, 30))
            self.assertLessEqual(e.time(), dt.time(20, 30))
            per_day[s.date()] = per_day.get(s.date(), 0) + (e - s).total_seconds() / 60
        self.assertTrue(all(v <= 90 for v in per_day.values()), per_day)

    def test_no_overlap_with_lectures_or_events(self):
        week_of_lectures(self.om)
        self.om.add(OTHER, "Gym", at(MON + dt.timedelta(days=1), "09:00"), at(MON + dt.timedelta(days=1), "11:00"))
        self.plan(at(MON, "07:00"))
        busy = [(sp.parse_local(r["start"]), sp.parse_local(r["end"])) for r in self.om.rows if r["calendarId"] != STUDY]
        for s, e, title in self.om.study():
            for b0, b1 in busy:
                self.assertFalse(sp.overlaps(s, e, b0, b1), f"{title} overlaps {b0}")

    def test_practice_in_the_morning_after_first_lecture(self):
        week_of_lectures(self.om)
        self.plan(at(MON, "07:00"))
        practice = [s for s in self.om.study() if s[2].startswith("🎯")]
        qc = [s for s in practice if "Quantum" in s[2]]
        self.assertEqual(len(qc), 2)  # difficulty 5 → 2 practice sessions a week
        for s, e, title in practice:
            self.assertLess(s.time(), dt.time(13, 0), title)
        first_gt_lecture_end = at(MON, "18:30")
        self.assertTrue(all(s > first_gt_lecture_end for s, _, t in practice if "Game" in t))

    def test_second_run_is_idempotent(self):
        week_of_lectures(self.om)
        self.plan(at(MON, "07:00"))
        before = self.om.study()
        p = self.plan(at(MON, "07:30"))
        self.assertEqual(p.created, [])
        self.assertEqual(self.om.study(), before)

    def test_ladder_advances_with_ratings(self):
        week_of_lectures(self.om)
        self.plan(at(MON, "07:00"))
        r1 = next(s for s in self.state["sessions"].values() if s["title"] == "📝 Review: Game theory"
                  and s["start"].startswith(MON.isoformat()))
        key = f"{sp.to_ms(sp.parse_local(r1['start']))}|{r1['title']}"
        sp.save_json(sp.RATINGS_PATH, {key: {"grade": "good"}})
        self.plan(at(MON + dt.timedelta(days=1), "07:00"))
        item = self.state["lectures"][r1["items"][0]]
        self.assertEqual(item["rep"], 1)
        self.assertEqual(item["due"], (MON + dt.timedelta(days=2)).isoformat())  # good → +2 days
        # The R2 for that lecture is now planned around Wednesday.
        pending = self.state["sessions"][item["pending"]]
        self.assertIn(sp.parse_local(pending["start"]).date(),
                      {MON + dt.timedelta(days=d) for d in (2, 3)})

    def test_again_brings_it_back_tomorrow_and_unrated_counts_as_good(self):
        item = {"course": "Game theory", "start": at(MON, "16:30").isoformat(), "end": at(MON, "18:30").isoformat(),
                "rep": 2, "interval": 6, "due": MON.isoformat(), "history": [], "pending": None, "retired": False}
        sp.apply_review(item, "again", at(MON, "19:00"), config(), None)
        self.assertEqual((item["rep"], item["interval"], item["due"]), (2, 1.0, (MON + dt.timedelta(days=1)).isoformat()))
        sp.apply_review(item, "good", at(MON + dt.timedelta(days=1), "19:00"), config(), None)
        self.assertEqual(item["interval"], 3.0)
        # Unrated finished session: auto "good" after 20 h.
        week_of_lectures(self.om)
        self.plan(at(MON, "07:00"))
        self.plan(at(MON + dt.timedelta(days=1), "18:00"))
        rated = [s for s in self.state["sessions"].values() if s["status"] == "rated"]
        self.assertTrue(rated and all(s["grade"] == "good" for s in rated))

    def test_harder_courses_come_back_sooner(self):
        easy = {"course": "Game theory", "rep": 3, "interval": 7, "history": [], "retired": False}
        hard = dict(easy, course="Quantum computing", history=[])
        cfg = config()
        sp.apply_review(easy, "good", at(MON, "10:00"), cfg, None)
        sp.apply_review(hard, "good", at(MON, "10:00"), cfg, None)
        self.assertEqual(easy["interval"], 21.0)
        self.assertLess(hard["interval"], easy["interval"])

    def test_exam_pulls_review_before_it_and_adds_prep(self):
        exam = MON + dt.timedelta(days=10)
        self.om.add(EXAMS, "GT written", at(exam, "00:00"), at(exam + dt.timedelta(days=1), "00:00"), all_day=True)
        item = {"course": "Game theory", "rep": 3, "interval": 7, "history": [], "retired": False}
        sp.apply_review(item, "good", at(MON, "10:00"), config(), exam)
        self.assertEqual(item["due"], (exam - dt.timedelta(days=2)).isoformat())
        week_of_lectures(self.om, weeks=2)
        self.plan(at(MON, "07:00"))
        self.assertTrue(any("Exam prep: Game theory" in t for _, _, t in self.om.study()))

    def test_exam_date_that_moves_is_followed(self):
        cfg = config()
        self.om.add(EXAMS, "GT oral", at(MON + dt.timedelta(days=30), "00:00"),
                    at(MON + dt.timedelta(days=31), "00:00"), all_day=True)
        self.assertEqual(sp.next_exam(cfg, "Game theory", self.om.events(MON, MON + dt.timedelta(days=60)), MON),
                         MON + dt.timedelta(days=30))
        self.om.rows[-1]["start"] = at(MON + dt.timedelta(days=12), "00:00").isoformat()
        self.assertEqual(sp.next_exam(cfg, "Game theory", self.om.events(MON, MON + dt.timedelta(days=60)), MON),
                         MON + dt.timedelta(days=12))
        # A code must match as a word: "GT" is not in "LGTB".
        self.assertIsNone(sp.next_exam(cfg, "Game theory", [{"title": "LGTB exam", "start": MON.isoformat()}], MON))

    def test_clash_moves_future_session_but_frozen_ones_stay(self):
        week_of_lectures(self.om)
        self.plan(at(MON, "07:00"))
        future = next(s for s in self.state["sessions"].values()
                      if sp.parse_local(s["start"]) > at(MON, "07:00") + dt.timedelta(days=2))
        fs, fe = sp.parse_local(future["start"]), sp.parse_local(future["end"])
        self.om.add(OTHER, "Dentist", fs, fe)
        soon = next(s for s in self.state["sessions"].values()
                    if sp.parse_local(s["start"]) < at(MON, "07:00") + dt.timedelta(hours=20))
        ss, se = sp.parse_local(soon["start"]), sp.parse_local(soon["end"])
        self.om.add(OTHER, "Call", ss, se)
        self.plan(at(MON, "07:10"))
        self.assertEqual(future["status"], "moved")
        self.assertEqual(soon["status"], "planned")  # inside 24 h: left alone
        replanned = [s for s in self.state["sessions"].values()
                     if s["title"] == future["title"] and s["status"] == "planned"]
        self.assertTrue(replanned)

    def test_user_moved_session_is_followed_not_duplicated(self):
        week_of_lectures(self.om)
        self.plan(at(MON, "07:00"))
        sess = next(s for s in self.state["sessions"].values() if s["kind"] == "practice")
        row = next(r for r in self.om.rows if r["title"] == sess["title"] and r["start"] == sess["start"])
        new_start = sp.parse_local(sess["start"]) + dt.timedelta(days=1)
        row["start"] = new_start.isoformat()
        row["end"] = (new_start + dt.timedelta(minutes=55)).isoformat()
        count = len(self.om.study())
        self.plan(at(MON, "07:10"))
        self.assertEqual(sess["start"], new_start.isoformat())
        self.assertEqual(len(self.om.study()), count)

    def test_interleaves_later_reviews_across_courses(self):
        for course, h in (("Game theory", "09:00"), ("Quantum computing", "11:00")):
            key = f"{course}|x"
            self.state["lectures"][key] = {"course": course, "start": at(MON - dt.timedelta(days=20), h).isoformat(),
                                           "end": at(MON - dt.timedelta(days=20), h).isoformat(), "rep": 3,
                                           "interval": 7, "due": (MON + dt.timedelta(days=1)).isoformat(),
                                           "history": [], "pending": None, "retired": False}
        self.plan(at(MON, "07:00"))
        mixed = [t for _, _, t in self.om.study() if t.startswith("🔀")]
        self.assertEqual(mixed, ["🔀 Interleaved review: Game theory · Quantum computing"])

    def test_old_planner_practice_counts_toward_the_week(self):
        week_of_lectures(self.om)
        self.om.add(STUDY, "🎯 Practice Problems: Quantum computing", at(MON, "09:00"), at(MON, "10:00"))
        self.plan(at(MON, "07:00"))
        new = [t for _, _, t in self.om.study() if t == "🎯 Practice: Quantum computing"]
        self.assertEqual(len(new), 1)


if __name__ == "__main__":
    unittest.main()
