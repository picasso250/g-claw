from __future__ import annotations

import sys
import time


def main() -> int:
    print("blocking test: start", flush=True)
    for i in range(1, 4):
        print(f"blocking test: tick {i}", flush=True)
        time.sleep(2)
    print("blocking test: done", flush=True)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
