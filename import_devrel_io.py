"""Import devrel-io data into velocirepo/data/.

Copies JSONL files from devrel-io/data/output/ into velocirepo's data directory,
skipping archive subdirectories and unsupported sources (dwh, google_analytics).

For pypi/cran/openvsx/plausible: adds the "source" key to each record.
For github (-> github-events): preserves raw event records as-is.
"""

import json
from pathlib import Path

SRC = Path("/Users/jeroen/repos/mine/devrel-io/data/output")
DST = Path("/Users/jeroen/repos/mine/velocirepo/data")

# devrel-io source name -> velocirepo source name
SOURCE_MAP = {
    "github": "github-events",
    "pypi": "pypi",
    "cran": "cran",
    "openvsx": "openvsx",
    "plausible": "plausible",
}

SKIP_SOURCES = {"dwh", "google_analytics"}


def process_standard_file(src_path: Path, dst_path: Path, source_name: str):
    """Add 'source' field to each record and write to destination."""
    dst_path.parent.mkdir(parents=True, exist_ok=True)
    records = []
    with open(src_path) as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            record = json.loads(line)
            record["source"] = source_name
            records.append(record)

    with open(dst_path, "w") as f:
        for record in records:
            f.write(json.dumps(record) + "\n")


def process_github_events_file(src_path: Path, dst_path: Path):
    """Add 'source' field to github event records and write to destination."""
    dst_path.parent.mkdir(parents=True, exist_ok=True)
    records = []
    with open(src_path) as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            record = json.loads(line)
            record["source"] = "github-events"
            records.append(record)

    with open(dst_path, "w") as f:
        for record in records:
            f.write(json.dumps(record) + "\n")


def main():
    stats = {"files": 0, "skipped_archive": 0, "skipped_source": 0}

    for source_dir in sorted(SRC.iterdir()):
        if not source_dir.is_dir():
            continue

        source_name = source_dir.name
        if source_name in SKIP_SOURCES:
            stats["skipped_source"] += 1
            print(f"Skipping unsupported source: {source_name}")
            continue

        if source_name not in SOURCE_MAP:
            print(f"Skipping unknown source: {source_name}")
            stats["skipped_source"] += 1
            continue

        velocirepo_source = SOURCE_MAP[source_name]

        for project_dir in sorted(source_dir.iterdir()):
            if not project_dir.is_dir():
                continue

            project_id = project_dir.name

            for jsonl_file in sorted(project_dir.glob("*.jsonl")):
                if "archive" in jsonl_file.parts:
                    stats["skipped_archive"] += 1
                    continue

                dst_path = DST / velocirepo_source / project_id / jsonl_file.name

                if source_name == "github":
                    process_github_events_file(jsonl_file, dst_path)
                else:
                    process_standard_file(jsonl_file, dst_path, velocirepo_source)

                stats["files"] += 1

            # Skip archive subdirectory entirely
            archive_dir = project_dir / "archive"
            if archive_dir.exists():
                n_archive = len(list(archive_dir.glob("*.jsonl")))
                stats["skipped_archive"] += n_archive

    print(f"\nDone!")
    print(f"  Files imported: {stats['files']}")
    print(f"  Archive files skipped: {stats['skipped_archive']}")
    print(f"  Sources skipped: {stats['skipped_source']}")


if __name__ == "__main__":
    main()
