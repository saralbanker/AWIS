#!/usr/bin/env python3
"""
Staleness Audit

Scans .agent/skills/*/SKILL.md for the `last_verified` / `kill_condition`
frontmatter this kit relies on for governance, and flags anything missing
them or overdue for review.

This is a calendar-based check, distinct from skill-forge's usage-based TTL
watcher (ttl_watcher.py): staleness audit asks "has this been reviewed
recently enough to still trust it," TTL watcher asks "has this forged skill
been used recently enough to keep it around." Run both periodically.

Usage:
    python .agent/scripts/audit_staleness.py [--max-age-days 180]
"""

import argparse
import re
import sys
from datetime import date
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent.parent
SKILLS_DIR = REPO_ROOT / ".agent" / "skills"


def parse_frontmatter(text: str) -> dict:
    match = re.match(r"^---\n(.*?)\n---\n", text, re.S)
    if not match:
        return {}
    fm = {}
    for line in match.group(1).splitlines():
        m = re.match(r"^([A-Za-z_]+):\s*(.*)$", line)
        if m:
            fm[m.group(1)] = m.group(2).strip()
    return fm


def check_file(path: Path, max_age_days: int, today: date) -> list:
    issues = []
    text = path.read_text(encoding="utf-8")
    fm = parse_frontmatter(text)
    label = path.relative_to(REPO_ROOT)

    last_verified = fm.get("last_verified")
    if not last_verified:
        issues.append(f"{label}: missing `last_verified`")
    else:
        try:
            verified_date = date.fromisoformat(last_verified)
            age = (today - verified_date).days
            if age > max_age_days:
                issues.append(f"{label}: last_verified {last_verified} is {age} days old (>{max_age_days})")
        except ValueError:
            issues.append(f"{label}: `last_verified` is not a valid ISO date: {last_verified!r}")

    if "kill_condition" not in fm and "kill_condition:" not in text:
        issues.append(f"{label}: missing `kill_condition`")

    return issues


def main():
    parser = argparse.ArgumentParser(description="Audit skills for staleness")
    parser.add_argument("--max-age-days", type=int, default=180,
                         help="Flag anything not reviewed within this many days (default: 180)")
    args = parser.parse_args()

    today = date.today()
    targets = sorted(SKILLS_DIR.glob("*/SKILL.md"))

    if not targets:
        print("No SKILL.md files found.")
        sys.exit(0)

    all_issues = []
    for path in targets:
        all_issues.extend(check_file(path, args.max_age_days, today))

    if not all_issues:
        print(f"OK — {len(targets)} file(s) checked, all within {args.max_age_days} days and fully tagged.")
        sys.exit(0)

    print(f"Staleness audit — {len(all_issues)} issue(s) across {len(targets)} file(s):\n")
    for issue in all_issues:
        print(f"  - {issue}")
    sys.exit(1)


if __name__ == "__main__":
    main()
