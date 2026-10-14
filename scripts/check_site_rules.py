#!/usr/bin/env python3
"""House-rule checks for stale.elghaly.dev HTML (stdlib only)."""
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent

FOOTER_URLS = [
    "https://chatting.elghaly.dev/",
    "https://elghaly.dev/",
]

FORBIDDEN_WORDS = re.compile(r"\b(live|profit|guarantee|audit|launched)\b", re.I)
FORBIDDEN = [
    (re.compile(r"qq\.com", re.I), "qq.com"),
    (re.compile(r"telegram", re.I), "Telegram"),
    (re.compile(r"t\.me/", re.I), "t.me link"),
    (re.compile(r"facebook\.com", re.I), "Facebook"),
]


def visible_text(raw: str) -> str:
    raw = re.sub(r"<!--.*?-->", "", raw, flags=re.S)
    raw = re.sub(r"<script[\s\S]*?</script>", "", raw, flags=re.I)
    raw = re.sub(r"<style[\s\S]*?</style>", "", raw, flags=re.I)
    raw = re.sub(r"<[^>]+>", " ", raw)
    return re.sub(r"\s+", " ", raw)


def main():
    errors = []
    index = ROOT / "index.html"
    if index.exists():
        text = index.read_text(encoding="utf-8")
        for url in FOOTER_URLS:
            if url.rstrip("/") not in text and url not in text:
                errors.append("index.html missing footer URL %s" % url)
        if "elghaly, LLC" not in text:
            errors.append("index.html missing elghaly, LLC")
        vis = visible_text(text)
        for pat, label in FORBIDDEN:
            if pat.search(vis):
                errors.append("index.html: forbidden %s" % label)
        for match in FORBIDDEN_WORDS.finditer(vis):
            errors.append("index.html: forbidden word %r" % match.group(0))

    for msg in errors:
        print("FAIL", msg)
    if errors:
        sys.exit(1)
    print("ok: stale index footer + house rules")


if __name__ == "__main__":
    main()
