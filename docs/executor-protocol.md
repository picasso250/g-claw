# Executor Protocol

This is the minimal single-executor protocol for `claw-executor`.

## Fixed API

- `POST /tasks`
- `POST /tasks/claim`
- `POST /tasks/:id/result`
- `GET /tasks/:id`

## Auth

- Use `Authorization: Bearer <EXECUTOR_TOKEN>`.
- Set the token as a Cloudflare Worker secret:

```bash
wrangler secret put EXECUTOR_TOKEN
```

- Do not store the token in `wrangler.toml`.
- On the client side, prefer setting `EXECUTOR_TOKEN` in the local environment instead of passing it in shell history.

## KV Keys

- `queue:pending`
- `task:<id>:spec`
- `task:<id>:state`

## R2 Keys

- `results/<id>/stdout.txt`
- `results/<id>/stderr.txt`

## Single-executor assumption

- This version assumes one active executor: `claw-executor`.
- Queue updates are intentionally simple and rely on that single-consumer model.
- Because this executor can run arbitrary scripts on the target machine, do not deploy it without a non-empty `EXECUTOR_TOKEN`.

## Deploy

1. Copy `cloudflare-executor/wrangler.toml.example` to `cloudflare-executor/wrangler.toml`.
2. Fill in the real KV namespace ID and R2 bucket name.
3. Keep the custom domain route as:

```toml
routes = [
  { pattern = "remote-executor.io99.xyz", custom_domain = true }
]
```

4. Set the Worker secret:

```bash
wrangler secret put EXECUTOR_TOKEN
```

5. Deploy from `cloudflare-executor/`:

```bash
wrangler deploy
```

## Task Spec

```json
{
  "id": "task_20260330T140000Z",
  "type": "script",
  "lang": "python",
  "filename": "probe.py",
  "content": "print('hello')",
  "timeout_sec": 300,
  "created_at": "2026-03-30T14:00:00Z"
}
```

- `cwd` is optional.
- If `cwd` is omitted, the executor runs the task in its own current working directory.

## Task State

```json
{
  "id": "task_20260330T140000Z",
  "status": "pending",
  "claimed_by": "",
  "claimed_at": "",
  "started_at": "",
  "finished_at": "",
  "exit_code": null,
  "result_stdout_key": "",
  "result_stderr_key": "",
  "artifact_key": "",
  "error": ""
}
```

## Result Upload

Executor posts JSON to `POST /tasks/:id/result`:

```json
{
  "status": "succeeded",
  "exit_code": 0,
  "stdout": "...\n",
  "stderr": "",
  "error": ""
}
```
