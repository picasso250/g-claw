from __future__ import annotations

import shutil
import subprocess
import sys
import textwrap
import time
from pathlib import Path

if hasattr(sys.stdout, "reconfigure"):
    sys.stdout.reconfigure(encoding="utf-8")


RUN_DIR = Path.home() / "claw-life-saver"
DEST_EXE_PATH = Path.home() / "bin" / "claw-life-saver.exe"
ENV_PATH = RUN_DIR / ".env"
MAIL_FILTER_PATH = RUN_DIR / "mail_filter_senders.txt"
CRON_CONFIG_PATH = RUN_DIR / "cron.json"
LOG_DIR = RUN_DIR / "logs"
STDOUT_LOG_PATH = LOG_DIR / "claw-life-saver-stdout.log"
STDERR_LOG_PATH = LOG_DIR / "claw-life-saver-stderr.log"
UPGRADE_LOG_PATH = LOG_DIR / "upgrade-claw-life-saver.log"
UPGRADE_DIR = RUN_DIR / "upgrade"
BUILT_EXE_PATH = UPGRADE_DIR / "claw-life-saver.next.exe"
START_SCRIPT_PATH = RUN_DIR / "start-claw-life-saver.py"
EXEC_SCRIPT_PATH = UPGRADE_DIR / "finalize-detached.py"
EXEC_STDOUT_PATH = UPGRADE_DIR / "finalize-detached.stdout.txt"
EXEC_STDERR_PATH = UPGRADE_DIR / "finalize-detached.stderr.txt"
TARGET_PROCESS_NAME = "claw-life-saver.exe"


def write_log(message: str) -> None:
    line = f"[{time.strftime('%Y-%m-%dT%H:%M:%S%z')}] {message}"
    print(line)
    LOG_DIR.mkdir(parents=True, exist_ok=True)
    with UPGRADE_LOG_PATH.open("a", encoding="utf-8") as f:
        f.write(line + "\n")


def require_path(path: Path, label: str) -> None:
    if not path.exists():
        raise SystemExit(f"Missing {label}: {path}")


def build_start_script() -> str:
    return textwrap.dedent(
        f"""\
        from __future__ import annotations

        import subprocess
        from pathlib import Path

        RUN_DIR = Path(r"{RUN_DIR}")
        EXE_PATH = Path(r"{DEST_EXE_PATH}")
        ENV_PATH = Path(r"{ENV_PATH}")
        MAIL_FILTER_PATH = Path(r"{MAIL_FILTER_PATH}")
        CRON_CONFIG_PATH = Path(r"{CRON_CONFIG_PATH}")
        LOG_DIR = Path(r"{LOG_DIR}")
        STDOUT_LOG_PATH = Path(r"{STDOUT_LOG_PATH}")
        STDERR_LOG_PATH = Path(r"{STDERR_LOG_PATH}")

        LOG_DIR.mkdir(parents=True, exist_ok=True)
        with STDOUT_LOG_PATH.open("ab") as stdout_file, STDERR_LOG_PATH.open("ab") as stderr_file:
            subprocess.Popen(
                [
                    str(EXE_PATH),
                    "serve",
                    "--env",
                    str(ENV_PATH),
                    "--mail-filter",
                    str(MAIL_FILTER_PATH),
                    "--cron-config",
                    str(CRON_CONFIG_PATH),
                    "--exec-subject-keyword",
                    "claw-life-saver",
                ],
                cwd=str(RUN_DIR),
                stdout=stdout_file,
                stderr=stderr_file,
            )
        """
    )


def build_exec_script() -> str:
    return textwrap.dedent(
        f"""\
        from __future__ import annotations

        import shutil
        import subprocess
        import sys
        import time
        from pathlib import Path

        if hasattr(sys.stdout, "reconfigure"):
            sys.stdout.reconfigure(encoding="utf-8")

        DEST_EXE_PATH = Path(r"{DEST_EXE_PATH}")
        BUILT_EXE_PATH = Path(r"{BUILT_EXE_PATH}")
        RUN_DIR = Path(r"{RUN_DIR}")
        START_SCRIPT_PATH = Path(r"{START_SCRIPT_PATH}")
        UPGRADE_LOG_PATH = Path(r"{UPGRADE_LOG_PATH}")
        TARGET_PROCESS_NAME = "{TARGET_PROCESS_NAME}"

        def write_log(message: str) -> None:
            line = f"[{{time.strftime('%Y-%m-%dT%H:%M:%S%z')}}] {{message}}"
            print(line)
            UPGRADE_LOG_PATH.parent.mkdir(parents=True, exist_ok=True)
            with UPGRADE_LOG_PATH.open("a", encoding="utf-8") as f:
                f.write(line + "\\n")

        write_log("Detached finalize stage starting after grace sleep")
        time.sleep(5)

        result = subprocess.run(
            [
                "powershell",
                "-NoProfile",
                "-Command",
                "(Get-CimInstance Win32_Process | Where-Object {{ $_.Name -eq '" + TARGET_PROCESS_NAME + "' }} | Select-Object -ExpandProperty ProcessId) -join '\\n'",
                ],
                capture_output=True,
                text=True,
                encoding="utf-8",
                errors="replace",
            check=False,
        )
        pids = [line.strip() for line in result.stdout.splitlines() if line.strip()]
        write_log(f"Found {{len(pids)}} matching claw-life-saver.exe process(es)")
        for pid in pids:
            write_log(f"Stopping PID={{pid}}")
            subprocess.run(["taskkill", "/PID", pid, "/F"], check=False, capture_output=True)

        time.sleep(5)

        result = subprocess.run(
            [
                "powershell",
                "-NoProfile",
                "-Command",
                "(Get-CimInstance Win32_Process | Where-Object {{ $_.Name -eq '" + TARGET_PROCESS_NAME + "' }} | Select-Object ProcessId, CommandLine | Format-List | Out-String)",
                ],
                capture_output=True,
                text=True,
                encoding="utf-8",
            errors="replace",
            check=False,
        )
        if result.stdout.strip():
            raise SystemExit("claw-life-saver.exe was observed again before replace:\\n" + result.stdout)
        write_log("Confirmed no claw-life-saver.exe process is running")

        DEST_EXE_PATH.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(BUILT_EXE_PATH, DEST_EXE_PATH)
        write_log("Replaced executable from prepared build")

        creationflags = 0
        if sys.platform == "win32":
            creationflags = subprocess.CREATE_NO_WINDOW | subprocess.DETACHED_PROCESS
        subprocess.Popen(
            ["python", str(START_SCRIPT_PATH)],
            cwd=str(RUN_DIR),
            creationflags=creationflags,
        )
        write_log("Detached finalize stage completed")
        """
    )


def main() -> int:
    LOG_DIR.mkdir(parents=True, exist_ok=True)
    UPGRADE_DIR.mkdir(parents=True, exist_ok=True)
    write_log("Starting claw-life-saver finalize prepare stage")

    require_path(RUN_DIR, "run dir")
    require_path(ENV_PATH, ".env")
    require_path(MAIL_FILTER_PATH, "mail filter")
    require_path(CRON_CONFIG_PATH, "cron config")
    require_path(BUILT_EXE_PATH, "prepared built exe")

    START_SCRIPT_PATH.write_text(build_start_script(), encoding="utf-8")
    write_log(f"Wrote start script to {START_SCRIPT_PATH}")

    EXEC_SCRIPT_PATH.write_text(build_exec_script(), encoding="utf-8")
    write_log(f"Wrote detached exec script to {EXEC_SCRIPT_PATH}")

    creationflags = 0
    if sys.platform == "win32":
        creationflags = subprocess.CREATE_NO_WINDOW | subprocess.DETACHED_PROCESS
    with EXEC_STDOUT_PATH.open("wb") as exec_stdout, EXEC_STDERR_PATH.open("wb") as exec_stderr:
        subprocess.Popen(
            ["python", str(EXEC_SCRIPT_PATH)],
            cwd=str(RUN_DIR),
            creationflags=creationflags,
            stdout=exec_stdout,
            stderr=exec_stderr,
        )
    write_log("Launched detached finalize stage")
    print(str(BUILT_EXE_PATH))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
