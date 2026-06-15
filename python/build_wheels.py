"""Build platform-specific wheels embedding the velocirepo binary.

Usage:
    python build_wheels.py --binary ../dist/velocirepo --version 0.1.0 --platform manylinux_2_17_x86_64
    python build_wheels.py --binary ../dist/velocirepo --version 0.1.0 --platform macosx_11_0_arm64
    python build_wheels.py --binary ../dist/velocirepo.exe --version 0.1.0 --platform win_amd64
"""

import argparse
import os
import shutil
import stat
import subprocess
import sys
import tempfile
from pathlib import Path


PLATFORM_TAGS = {
    "linux_amd64": "manylinux_2_17_x86_64.manylinux2014_x86_64",
    "linux_arm64": "manylinux_2_17_aarch64.manylinux2014_aarch64",
    "darwin_amd64": "macosx_11_0_x86_64",
    "darwin_arm64": "macosx_11_0_arm64",
    "windows_amd64": "win_amd64",
}


def build_wheel(binary_path: str, version: str, platform: str, output_dir: str):
    binary_path = Path(binary_path).resolve()
    output_dir = Path(output_dir).resolve()
    output_dir.mkdir(parents=True, exist_ok=True)

    with tempfile.TemporaryDirectory() as tmpdir:
        tmpdir = Path(tmpdir)

        pkg_dir = tmpdir / "velocirepo"
        pkg_dir.mkdir()

        # __init__.py
        (pkg_dir / "__init__.py").write_text(
            'from importlib.metadata import version\n__version__ = version("velocirepo")\n'
        )

        # __main__.py
        main_src = Path(__file__).parent / "src" / "velocirepo" / "__main__.py"
        shutil.copy(main_src, pkg_dir / "__main__.py")

        # Embed binary
        bin_dir = pkg_dir / "bin"
        bin_dir.mkdir()
        bin_name = "velocirepo.exe" if binary_path.suffix == ".exe" else "velocirepo"
        dest = bin_dir / bin_name
        shutil.copy2(binary_path, dest)
        dest.chmod(dest.stat().st_mode | stat.S_IEXEC)

        # _version.py
        (pkg_dir / "_version.py").write_text(f'__version__ = "{version}"\n')

        # Write pyproject.toml for hatchling
        pyproject = tmpdir / "pyproject.toml"
        pyproject.write_text(f"""\
[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"

[project]
name = "velocirepo"
version = "{version}"
description = "Fetch and aggregate open-source project metrics"
license = "MIT"
requires-python = ">=3.9"
authors = [{{ name = "Jeroen Janssens", email = "jeroen@jeroenjanssens.com" }}]

[project.scripts]
velocirepo = "velocirepo.__main__:main"

[tool.hatch.build.targets.wheel]
packages = ["velocirepo"]
include = ["velocirepo/**"]
""")

        # Build using hatchling
        subprocess.run(
            [sys.executable, "-m", "hatchling", "build", "-t", "wheel", "-d", str(output_dir)],
            cwd=tmpdir,
            check=True,
        )

        # Rename the wheel to include the correct platform tag
        for whl in output_dir.glob("velocirepo-*.whl"):
            if "none-any" in whl.name:
                new_name = whl.name.replace("py3-none-any", f"py3-none-{platform}")
                new_path = whl.parent / new_name
                whl.rename(new_path)
                print(f"Built: {new_path.name}")
                return str(new_path)

        # Already platform-tagged
        for whl in output_dir.glob("velocirepo-*.whl"):
            print(f"Built: {whl.name}")
            return str(whl)


def main():
    parser = argparse.ArgumentParser(description="Build platform-specific wheel")
    parser.add_argument("--binary", required=True, help="Path to velocirepo binary")
    parser.add_argument("--version", required=True, help="Package version")
    parser.add_argument(
        "--platform",
        required=True,
        choices=list(PLATFORM_TAGS.values()) + list(PLATFORM_TAGS.keys()),
        help="Platform tag or shorthand (e.g., linux_amd64)",
    )
    parser.add_argument("--output", default="dist", help="Output directory")
    args = parser.parse_args()

    platform = PLATFORM_TAGS.get(args.platform, args.platform)
    build_wheel(args.binary, args.version, platform, args.output)


if __name__ == "__main__":
    main()
