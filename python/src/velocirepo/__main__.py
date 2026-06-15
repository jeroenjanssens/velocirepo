"""Entry point for `python -m velocirepo` and the console script."""

import os
import sys


def _find_binary():
    name = "velocirepo.exe" if sys.platform == "win32" else "velocirepo"

    # Look in the package's own bin/ directory (where build_wheels embeds it)
    pkg_dir = os.path.dirname(os.path.abspath(__file__))
    path = os.path.join(pkg_dir, "bin", name)
    if os.path.isfile(path):
        return path

    return None


def main():
    binary = _find_binary()
    if binary is None:
        print("error: velocirepo binary not found", file=sys.stderr)
        sys.exit(1)

    if sys.platform == "win32":
        import subprocess

        result = subprocess.run([binary] + sys.argv[1:])
        sys.exit(result.returncode)
    else:
        os.execvp(binary, [binary] + sys.argv[1:])


if __name__ == "__main__":
    main()
