from __future__ import annotations

import os
import pathlib
import subprocess
import sys
from datetime import datetime


def run_command(args: list[str], cwd: pathlib.Path | None = None) -> str:
    try:
        result = subprocess.run(
            args,
            cwd=str(cwd) if cwd else None,
            capture_output=True,
            text=True,
            encoding="utf-8",
            errors="replace",
            check=False,
        )
    except FileNotFoundError:
        return "missing\n"

    output = result.stdout
    if result.stderr:
        output += result.stderr
    if not output.endswith("\n"):
        output += "\n"
    return output


def command_path(name: str) -> str:
    import shutil

    path = shutil.which(name)
    return path if path else "missing"


def main() -> int:
    cwd = pathlib.Path.cwd()
    home = pathlib.Path.home()
    repo_dir = home / "glaw"
    output_path = cwd / "probe-shuyao-before-upgrade.txt"

    lines: list[str] = [
        f"GeneratedAt: {datetime.now().astimezone().isoformat()}",
        f"UserProfile: {home}",
        f"PWD: {cwd}",
        "",
        "===== PWD =====",
        str(cwd),
        "",
        "===== Repo Exists =====",
        f"RepoDir: {repo_dir}",
        f"Exists: {repo_dir.exists()}",
    ]

    if repo_dir.exists():
        lines.extend(
            [
                "",
                "===== Git Status =====",
                run_command(["git", "-C", str(repo_dir), "status", "--short"]).rstrip("\n"),
                "",
                "===== Git HEAD =====",
                run_command(["git", "-C", str(repo_dir), "rev-parse", "HEAD"]).rstrip("\n"),
            ]
        )

    lines.extend(["", "===== Commands ====="])
    for name in ["python", "git", "go"]:
        lines.append(f"[{name}]")
        lines.append(command_path(name))
        lines.append("")

    lines.extend(
        [
            "===== Relevant Processes =====",
            run_command(
                [
                    "powershell",
                    "-NoProfile",
                    "-Command",
                    "Get-CimInstance Win32_Process | Where-Object { $_.Name -match 'glaw|kilocode|gemini|node' } | Select-Object ProcessId, Name, CommandLine | Format-List | Out-String",
                ]
            ).rstrip("\n"),
            "",
            "===== Rescue Files =====",
        ]
    )
    for path in [
        home / "claw-life-saver" / "INIT.md",
        home / "claw-life-saver" / "SOUL.md",
        home / "claw-life-saver" / "USER.md",
        home / "claw-life-saver" / "MEMORY.txt",
        home / "claw-life-saver" / ".env",
    ]:
        lines.append(f"{path} :: {path.exists()}")

    lines.extend(["OK: probe finished", ""])
    output_path.write_text("\n".join(lines), encoding="utf-8", newline="\n")
    print(f"Probe finished: {output_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
