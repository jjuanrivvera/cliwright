#!/usr/bin/env python3
"""Validate cliwright's Claude Code plugin manifests.

Reused verbatim by the pre-commit hook (.githooks/pre-commit) and CI
(.github/workflows/validate.yml) so local and remote checks never drift.

Two layers run against .claude-plugin/{plugin,marketplace}.json:

  1. JSON Schema — validates each manifest against the schemastore schema
     vendored under .claude-plugin/schemas/. This is the layer that catches a
     bad `agents` field (must be a .md file path, not a directory). Requires
     the `jsonschema` package; skipped with a notice if it is not importable.

  2. Structural — pure stdlib, always runs: every component path referenced or
     auto-discovered must resolve, and each agent/command/skill markdown must
     carry the frontmatter Claude Code needs to load it.

Exit status: 0 when clean, 1 on any error. Warnings never fail the build.
"""

from __future__ import annotations

import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PLUGIN_DIR = ROOT / ".claude-plugin"
SCHEMA_DIR = PLUGIN_DIR / "schemas"

GREEN, RED, YELLOW, DIM, RESET = "\033[32m", "\033[31m", "\033[33m", "\033[2m", "\033[0m"
if not sys.stdout.isatty():
    GREEN = RED = YELLOW = DIM = RESET = ""

errors: list[str] = []
warnings: list[str] = []


def err(msg: str) -> None:
    errors.append(msg)


def warn(msg: str) -> None:
    warnings.append(msg)


def load_json(path: Path) -> dict | None:
    """Parse JSON, recording a precise error instead of raising."""
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except FileNotFoundError:
        err(f"{rel(path)}: file not found")
    except json.JSONDecodeError as e:
        err(f"{rel(path)}: invalid JSON — {e.msg} (line {e.lineno}, col {e.colno})")
    return None


def rel(path: Path) -> str:
    try:
        return str(path.relative_to(ROOT))
    except ValueError:
        return str(path)


def parse_frontmatter(path: Path) -> dict[str, str] | None:
    """Minimal top-level YAML frontmatter reader (no PyYAML dependency).

    Returns the set of top-level keys (values flattened to strings) found in the
    leading `---` fenced block, or None when no frontmatter block is present.
    Good enough to assert required keys exist; not a full YAML parser.
    """
    text = path.read_text(encoding="utf-8")
    if not text.startswith("---"):
        return None
    lines = text.splitlines()
    end = next((i for i in range(1, len(lines)) if lines[i].strip() == "---"), None)
    if end is None:
        return None
    keys: dict[str, str] = {}
    for line in lines[1:end]:
        # only top-level keys (no indentation), of the form `key: ...`
        if line and not line[0].isspace() and ":" in line:
            k, _, v = line.partition(":")
            keys[k.strip()] = v.strip()
    return keys


def require_frontmatter(md: Path, required: list[str], kind: str) -> None:
    fm = parse_frontmatter(md)
    if fm is None:
        err(f"{rel(md)}: {kind} is missing YAML frontmatter (`---` block)")
        return
    for key in required:
        if not fm.get(key):
            err(f"{rel(md)}: {kind} frontmatter missing required `{key}`")


# --------------------------------------------------------------------------- #
# Layer 1: JSON Schema validation
# --------------------------------------------------------------------------- #
def schema_validate(manifest: dict, manifest_path: Path, schema_path: Path) -> None:
    try:
        from jsonschema import Draft7Validator
    except ImportError:
        warn(
            "jsonschema not installed — skipped full schema validation "
            f"({rel(manifest_path)}). Install with `pip install jsonschema` "
            "for complete coverage (CI runs it)."
        )
        return
    schema = load_json(schema_path)
    if schema is None:
        return
    validator = Draft7Validator(schema)
    found = False
    for e in sorted(validator.iter_errors(manifest), key=lambda e: e.path):
        found = True
        loc = getattr(e, "json_path", None) or "$." + ".".join(map(str, e.absolute_path))
        err(f"{rel(manifest_path)}: schema violation at {loc} — {e.message}")
    if not found:
        print(f"  {GREEN}✓{RESET} {rel(manifest_path)} valid against {schema_path.name}")


# --------------------------------------------------------------------------- #
# Layer 2: structural / self-containment checks
# --------------------------------------------------------------------------- #
def check_component_pointer(field: str, value, md_only: bool) -> None:
    """A manifest component pointer must resolve on disk.

    `md_only` mirrors the schema: the `agents` field only accepts .md file
    paths, while `commands`/`skills` also accept a directory.
    """
    items = value if isinstance(value, list) else [value]
    for item in items:
        if not isinstance(item, str):
            err(f"plugin.json: `{field}` entries must be strings, got {type(item).__name__}")
            continue
        target = (ROOT / item).resolve()
        if md_only and not item.endswith(".md"):
            err(f"plugin.json: `{field}` must point to a .md file, got '{item}'")
        if not target.exists():
            err(f"plugin.json: `{field}` path '{item}' does not exist")


def check_plugin(manifest: dict) -> None:
    for field, md_only in (("agents", True), ("commands", False), ("skills", False)):
        if field in manifest:
            check_component_pointer(field, manifest[field], md_only)

    # Auto-discovered component directories: every markdown must be loadable.
    for md in sorted((ROOT / "agents").glob("*.md")):
        require_frontmatter(md, ["name", "description"], "agent")
    for md in sorted((ROOT / "commands").glob("*.md")):
        require_frontmatter(md, ["description"], "command")
    skills_dir = ROOT / "skills"
    if skills_dir.is_dir():
        for sub in sorted(p for p in skills_dir.iterdir() if p.is_dir()):
            skill_md = sub / "SKILL.md"
            if not skill_md.exists():
                err(f"skills/{sub.name}: missing SKILL.md")
            else:
                require_frontmatter(skill_md, ["name", "description"], "skill")


def check_marketplace(manifest: dict, plugin_manifest: dict | None) -> None:
    for i, entry in enumerate(manifest.get("plugins", [])):
        name = entry.get("name", f"#{i}")
        source = entry.get("source")
        if not isinstance(source, str):
            continue  # object sources (git/github) aren't local; schema covers shape
        src_root = (ROOT / source).resolve()
        src_manifest = src_root / ".claude-plugin" / "plugin.json"
        if not src_manifest.exists():
            err(
                f"marketplace.json: plugin '{name}' source '{source}' has no "
                ".claude-plugin/plugin.json"
            )
            continue
        if plugin_manifest and src_root == ROOT:
            if entry.get("name") != plugin_manifest.get("name"):
                err(
                    f"marketplace.json: plugin name '{entry.get('name')}' != "
                    f"plugin.json name '{plugin_manifest.get('name')}'"
                )
            if entry.get("version") and entry["version"] != plugin_manifest.get("version"):
                warn(
                    f"marketplace.json: plugin '{name}' version "
                    f"'{entry['version']}' != plugin.json '{plugin_manifest.get('version')}'"
                )


def main() -> int:
    print(f"{DIM}Validating Claude Code plugin manifests in {rel(PLUGIN_DIR)}…{RESET}")

    plugin_path = PLUGIN_DIR / "plugin.json"
    market_path = PLUGIN_DIR / "marketplace.json"
    plugin = load_json(plugin_path)
    market = load_json(market_path)

    if plugin is not None:
        schema_validate(plugin, plugin_path, SCHEMA_DIR / "plugin-manifest.schema.json")
        check_plugin(plugin)
    if market is not None:
        schema_validate(market, market_path, SCHEMA_DIR / "marketplace.schema.json")
        check_marketplace(market, plugin)

    print()
    for w in warnings:
        print(f"  {YELLOW}⚠{RESET} {w}")
    if errors:
        for e in errors:
            print(f"  {RED}✗{RESET} {e}")
        print(f"\n{RED}✗ {len(errors)} error(s){RESET}"
              f"{f', {len(warnings)} warning(s)' if warnings else ''}.")
        return 1
    print(f"{GREEN}✓ Manifests valid"
          f"{f' ({len(warnings)} warning(s))' if warnings else ''}.{RESET}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
