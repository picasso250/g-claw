from __future__ import annotations

import sys
from pathlib import Path

if hasattr(sys.stdout, "reconfigure"):
    sys.stdout.reconfigure(encoding="utf-8")


def main() -> int:
    print(f"argc={len(sys.argv)}")
    for index, arg in enumerate(sys.argv):
        print(f"argv[{index}]={arg}")

    if len(sys.argv) < 2:
        print("multi_attachment_supported=false")
        return 0

    payload_path = Path(sys.argv[1])
    print(f"payload_exists={payload_path.exists()}")
    if payload_path.exists():
        print(f"payload_name={payload_path.name}")
        print(f"payload_size={payload_path.stat().st_size}")
        print(f"payload_text={payload_path.read_text(encoding='utf-8', errors='replace').strip()}")
    print("multi_attachment_supported=true")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
