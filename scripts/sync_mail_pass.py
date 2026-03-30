from __future__ import annotations

import pathlib
import sys


def load_env_map(path: pathlib.Path) -> dict[str, str]:
    values: dict[str, str] = {}
    for raw_line in path.read_text(encoding="utf-8-sig").splitlines():
        line = raw_line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, value = line.split("=", 1)
        values[key.strip()] = value.strip()
    return values


def replace_env_key(path: pathlib.Path, key: str, value: str) -> None:
    lines = path.read_text(encoding="utf-8").splitlines()
    replaced = False
    new_lines: list[str] = []
    for line in lines:
        stripped = line.strip()
        if stripped.startswith(f"{key}="):
            new_lines.append(f"{key}={value}")
            replaced = True
        else:
            new_lines.append(line)
    if not replaced:
        new_lines.append(f"{key}={value}")
    path.write_text("\n".join(new_lines) + "\n", encoding="utf-8")


def main() -> int:
    if len(sys.argv) != 3:
        print("usage: python scripts/sync_mail_pass.py <source-env> <target-env>", file=sys.stderr)
        return 2

    source = pathlib.Path(sys.argv[1]).expanduser()
    target = pathlib.Path(sys.argv[2]).expanduser()

    source_values = load_env_map(source)
    if "MAIL_PASS" not in source_values or not source_values["MAIL_PASS"]:
        print("MAIL_PASS missing in source env", file=sys.stderr)
        return 1

    replace_env_key(target, "MAIL_PASS", source_values["MAIL_PASS"])
    print(f"MAIL_PASS synced to {target}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
