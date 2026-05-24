# CRONTAB.md — Scheduled tasks
# Each entry: YAML frontmatter between --- delimiters, followed by task instructions.
# Fields: name, schedule (5-field cron), description, enabled (true/false).

---
description: 每10分钟自动报时
enabled: true
name: every-10min-timer
schedule: '*/10 * * * *'
---
现在时间到了！请向用户报告当前的准确时间。

