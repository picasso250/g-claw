export default {
  async fetch(request, env) {
    try {
      if (!isAuthorized(request, env)) {
        return json({ ok: false, error: "unauthorized" }, 401);
      }

      const url = new URL(request.url);
      const path = url.pathname.replace(/\/+$/, "") || "/";

      if (request.method === "POST" && path === "/tasks") {
        return await createTask(request, env);
      }
      if (request.method === "POST" && path === "/tasks/claim") {
        return await claimTask(request, env);
      }
      if (request.method === "GET" && path.startsWith("/tasks/")) {
        const taskId = path.slice("/tasks/".length);
        return await getTask(taskId, env);
      }
      if (request.method === "POST" && path.startsWith("/tasks/") && path.endsWith("/result")) {
        const taskId = path.slice("/tasks/".length, -"/result".length);
        return await saveResult(taskId, request, env);
      }

      return json({ ok: false, error: "not_found" }, 404);
    } catch (error) {
      return json(
        {
          ok: false,
          error: "internal_error",
          detail: error instanceof Error ? error.message : String(error),
        },
        500,
      );
    }
  },
};

const QUEUE_KEY = "queue:pending";

function isAuthorized(request, env) {
  const expected = (env.EXECUTOR_TOKEN || "").trim();
  if (!expected) {
    return true;
  }
  const actual = request.headers.get("authorization") || "";
  return actual === `Bearer ${expected}`;
}

function taskSpecKey(taskId) {
  return `task:${taskId}:spec`;
}

function taskStateKey(taskId) {
  return `task:${taskId}:state`;
}

function stdoutKey(taskId) {
  return `results/${taskId}/stdout.txt`;
}

function stderrKey(taskId) {
  return `results/${taskId}/stderr.txt`;
}

function artifactKey(taskId, fileName) {
  return `results/${taskId}/${fileName}`;
}

async function createTask(request, env) {
  const payload = await request.json();
  const task = normalizeTaskSpec(payload);
  const now = new Date().toISOString();
  task.created_at = task.created_at || now;

  const state = {
    id: task.id,
    status: "pending",
    claimed_by: "",
    claimed_at: "",
    started_at: "",
    finished_at: "",
    exit_code: null,
    result_stdout_key: "",
    result_stderr_key: "",
    artifact_key: "",
    error: "",
  };

  await env.EXECUTOR_KV.put(taskSpecKey(task.id), JSON.stringify(task));
  await env.EXECUTOR_KV.put(taskStateKey(task.id), JSON.stringify(state));

  const queue = await readQueue(env);
  if (!queue.includes(task.id)) {
    queue.push(task.id);
    await writeQueue(env, queue);
  }

  return json({ ok: true, task, state });
}

async function claimTask(request, env) {
  const payload = await request.json().catch(() => ({}));
  const agentId = String(payload.agent_id || "").trim();
  if (!agentId) {
    return json({ ok: false, error: "agent_id_required" }, 400);
  }

  const queue = await readQueue(env);
  const nextQueue = [...queue];

  while (nextQueue.length > 0) {
    const taskId = nextQueue.shift();
    const rawSpec = await env.EXECUTOR_KV.get(taskSpecKey(taskId));
    const rawState = await env.EXECUTOR_KV.get(taskStateKey(taskId));
    if (!rawSpec || !rawState) {
      continue;
    }

    const task = JSON.parse(rawSpec);
    const state = JSON.parse(rawState);
    if (state.status !== "pending") {
      continue;
    }

    const now = new Date().toISOString();
    state.status = "claimed";
    state.claimed_by = agentId;
    state.claimed_at = now;
    state.started_at = now;

    await env.EXECUTOR_KV.put(taskStateKey(taskId), JSON.stringify(state));
    await writeQueue(env, nextQueue);
    return json({ ok: true, task, state });
  }

  await writeQueue(env, nextQueue);
  return json({ ok: true, task: null });
}

async function getTask(taskId, env) {
  if (!taskId) {
    return json({ ok: false, error: "task_id_required" }, 400);
  }

  const [rawSpec, rawState] = await Promise.all([
    env.EXECUTOR_KV.get(taskSpecKey(taskId)),
    env.EXECUTOR_KV.get(taskStateKey(taskId)),
  ]);
  if (!rawSpec || !rawState) {
    return json({ ok: false, error: "task_not_found" }, 404);
  }

  return json({
    ok: true,
    task: JSON.parse(rawSpec),
    state: JSON.parse(rawState),
  });
}

async function saveResult(taskId, request, env) {
  if (!taskId) {
    return json({ ok: false, error: "task_id_required" }, 400);
  }

  const rawState = await env.EXECUTOR_KV.get(taskStateKey(taskId));
  if (!rawState) {
    return json({ ok: false, error: "task_not_found" }, 404);
  }

  const payload = await request.json();
  const state = JSON.parse(rawState);
  const now = new Date().toISOString();

  const stdout = String(payload.stdout || "");
  const stderr = String(payload.stderr || "");
  await env.EXECUTOR_RESULTS.put(stdoutKey(taskId), stdout, {
    httpMetadata: { contentType: "text/plain; charset=utf-8" },
  });
  await env.EXECUTOR_RESULTS.put(stderrKey(taskId), stderr, {
    httpMetadata: { contentType: "text/plain; charset=utf-8" },
  });

  state.result_stdout_key = stdoutKey(taskId);
  state.result_stderr_key = stderrKey(taskId);
  state.exit_code = toNullableNumber(payload.exit_code);
  state.error = String(payload.error || "");
  state.finished_at = now;
  state.status = payload.status === "succeeded" ? "succeeded" : "failed";

  if (payload.artifact_name && payload.artifact_base64) {
    const bytes = Uint8Array.from(atob(String(payload.artifact_base64)), (c) => c.charCodeAt(0));
    const key = artifactKey(taskId, sanitizeFileName(String(payload.artifact_name)));
    await env.EXECUTOR_RESULTS.put(key, bytes, {
      httpMetadata: { contentType: "application/octet-stream" },
    });
    state.artifact_key = key;
  }

  await env.EXECUTOR_KV.put(taskStateKey(taskId), JSON.stringify(state));
  return json({ ok: true, state });
}

function normalizeTaskSpec(payload) {
  const task = {
    id: String(payload.id || "").trim(),
    type: String(payload.type || "script").trim(),
    lang: String(payload.lang || "").trim(),
    filename: String(payload.filename || "").trim(),
    content: String(payload.content || ""),
    cwd: String(payload.cwd || "").trim(),
    timeout_sec: Number(payload.timeout_sec || 300),
    created_at: String(payload.created_at || "").trim(),
  };

  if (!task.id) {
    throw new Error("task id is required");
  }
  if (!task.lang) {
    throw new Error("task lang is required");
  }
  if (!task.filename) {
    throw new Error("task filename is required");
  }
  if (!task.content) {
    throw new Error("task content is required");
  }
  return task;
}

async function readQueue(env) {
  const raw = await env.EXECUTOR_KV.get(QUEUE_KEY);
  if (!raw) {
    return [];
  }
  try {
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? parsed.map((x) => String(x)) : [];
  } catch {
    return [];
  }
}

async function writeQueue(env, queue) {
  await env.EXECUTOR_KV.put(QUEUE_KEY, JSON.stringify(queue));
}

function json(body, status = 200) {
  return new Response(JSON.stringify(body, null, 2), {
    status,
    headers: {
      "content-type": "application/json; charset=utf-8",
    },
  });
}

function sanitizeFileName(value) {
  return value.replace(/[^a-zA-Z0-9._-]/g, "_");
}

function toNullableNumber(value) {
  if (value === null || value === undefined || value === "") {
    return null;
  }
  const n = Number(value);
  return Number.isFinite(n) ? n : null;
}
