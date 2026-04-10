from __future__ import annotations

import argparse
import subprocess
import zipfile
from pathlib import Path


def tracked_files(repo_root: Path) -> list[Path]:
    result = subprocess.run(
        ["git", "ls-files", "*.go", "go.mod", "go.sum"],
        cwd=repo_root,
        capture_output=True,
        text=True,
        encoding="utf-8",
        errors="replace",
        check=True,
    )
    files: list[Path] = []
    for line in result.stdout.splitlines():
        rel = line.strip()
        if not rel:
            continue
        path = repo_root / rel
        if path.is_file():
            files.append(path)
    return files


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo", default=".", help="repo root")
    parser.add_argument("--output", required=True, help="output zip path")
    args = parser.parse_args()

    repo_root = Path(args.repo).resolve()
    output_path = Path(args.output).resolve()
    output_path.parent.mkdir(parents=True, exist_ok=True)

    files = tracked_files(repo_root)
    if not files:
        raise SystemExit("no tracked Go source files found")

    with zipfile.ZipFile(output_path, "w", compression=zipfile.ZIP_DEFLATED) as zf:
        for path in files:
            zf.write(path, arcname=path.relative_to(repo_root).as_posix())

    print(f"repo={repo_root}")
    print(f"output={output_path}")
    print(f"files={len(files)}")
    for path in files:
        print(path.relative_to(repo_root).as_posix())
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
