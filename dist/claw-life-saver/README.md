# claw-life-saver

This bundle installs and starts a minimal rescue `glaw` instance under `$HOME\claw-life-saver`.

What it does:

- runs `git pull` in `$HOME\glaw`
- builds `glaw` locally into `$HOME\bin\glaw-life-saver.exe`
- creates `$HOME\claw-life-saver`
- syncs `MAIL_USER`, `MAIL_PASS`, and `MAIL_IMAP_SERVER` from `$HOME\.env`
- writes rescue-specific `.env`, `mail_filter_senders.txt`, `cron.json`, and `INIT.md`
- writes a reusable start script
- starts the rescue instance in a new PowerShell window

What it does not do:

- does not enable Feishu
- does not copy any scheduler tasks
- does not ship any compiled executable inside the zip

Run:

```powershell
pwsh -File .\install-claw-life-saver.ps1
```
