# g-claw

Minimal open-source starting point for a mail-driven assistant gateway.

## What it does

`g-claw` polls an IMAP inbox or accepts Feishu bot messages over a long connection, archives matched content into `gateway/pending/`, and dispatches those files to an external PowerShell wrapper that runs your assistant.

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
- `FEISHU_ENABLE`: set to `true` to enable the Feishu long-connection bot client
- `FEISHU_APP_ID`: Feishu app ID for the bot-enabled custom app
- `FEISHU_APP_SECRET`: Feishu app secret for the bot-enabled custom app
- `FEISHU_ALLOWED_OPEN_IDS`: comma-separated trusted Feishu sender `open_id` values
- `FEISHU_ALLOWED_CHAT_IDS`: comma-separated trusted Feishu chat IDs

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

## Feishu Bot

When `FEISHU_ENABLE=true`, the gateway starts a Feishu long-connection bot client using the official Go SDK. The current implementation handles inbound `im.message.receive_v1` text events. Trusted messages are persisted into `gateway/pending/` and dispatched through the same processing pipeline as mail.

## Notes Before Open Source

- Replace the module path in `go.mod` with the final repository path.
- Review the prompt text in `cmd/gateway/main.go` for product-specific policy.
- The wrapper contract is still local and opinionated by design; if you want broader reuse, the next step is to abstract the assistant runner interface.
