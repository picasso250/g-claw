---
name: mail-script-executor
description: Send a `.py` or `.ps1` script to a trusted remote machine by email, trigger execution via a subject keyword, then inspect the reply attachments and latest mailbox state. Use when Codex needs to drive a cautious remote workflow step by step through email instead of SSH or direct shell access, especially for probe scripts, environment checks, staged upgrade preparation, or final execution on the remote machine.
---

# Mail Script Executor

Use this skill when the remote machine already runs `glaw serve --exec-subject-keyword <keyword>` and executes one attached `.py` / `.ps1` by mail.

Current convention:

- Default execution keyword: `claw-life-saver`.
- Prefer subjects such as `claw-life-saver probe ...`, `claw-life-saver rerun ...`, or `claw-life-saver finalize ...` unless the user or a fresh runtime check says otherwise.

## Rules

1. One mail, one executable attachment.
2. Default to `.py` for execution-chain scripts. Use `.ps1` only when Python is genuinely unsuitable for the task.
3. Every mailed `.py` should include:

```python
if hasattr(sys.stdout, "reconfigure"):
    sys.stdout.reconfigure(encoding="utf-8")
```

4. Trigger execution by subject keyword and keep the body short, for example `请执行附件。`
5. After the remote upgrades to the zip-reply version, you may append absolute Windows paths in the body, one per line after the first short instruction line; existing files will be added to the reply zip.
6. On Windows, detached follow-up PowerShell launches must prefer `-ExecutionPolicy Bypass -File <script>` over inline `-Command`.
7. Keep high-risk steps narrow and reversible until the final cutover.
8. Send local helper scripts with `python`, not `uv`.

## Reply Inspection

1. Wait 30 seconds after sending, then inspect the latest reply from that sender.
2. Treat `stderr` as authoritative for failures.
3. Before upgrade, expect separate `stdout.txt` / `stderr.txt`; after upgrade, expect one zip attachment and inspect its contents first.
4. For detached start flows, verify the next state with a second probe that reads the target process plus startup and redirected log files.

## Script

Prefer one serial shell command for send + latest-reply polling instead of a wrapper helper:

```powershell
python C:\path\to\send_email.py --to trusted-recipient@example.com --subject "claw-life-saver probe git clean" --markdown-body-file C:\path\to\gateway\outbox\mail.md --attachments C:\path\to\script.py; go run .\cmd\glaw mail latest --sender trusted-recipient@example.com --max-sleep-seconds 60 --poll-interval-seconds 2
```
