# g-claw

Minimal open-source starting point for a mail-driven assistant gateway.

## What it does

`g-claw` polls an IMAP inbox, filters messages from configured senders, archives matched content into `gateway/pending/`, and dispatches those files to an external PowerShell wrapper that runs your assistant.

The current implementation is intentionally small:

- `cmd/gateway/main.go`: gateway entrypoint
- `INIT.md`: initialization prompt file consumed by the wrapper
- `.env.example`: local configuration template

## Configuration

Copy `.env.example` to `.env` and fill in:

- `MAIL_USER`: IMAP login address
- `MAIL_PASS`: IMAP password or app password
- `MAIL_IMAP_SERVER`: IMAP host, for example `imap.example.com`
- `MAIL_FILTER_SENDER`: comma-separated trusted senders to process
- `AGENT_WRAP_PATH`: absolute path to the PowerShell wrapper that accepts `-p <prompt>`

## Run

Build:

```powershell
go build ./...
```

Start:

```powershell
go run ./cmd/gateway
```

The process expects to be started from the repository root so it can access `gateway/` and `INIT.md`.

## Notes Before Open Source

- Replace the module path in `go.mod` with the final repository path.
- Review the prompt text in `cmd/gateway/main.go` for product-specific policy.
- The wrapper contract is still local and opinionated by design; if you want broader reuse, the next step is to abstract the assistant runner interface.
