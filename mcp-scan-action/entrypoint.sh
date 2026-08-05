#!/bin/sh
set -eu
scan_path="${INPUT_PATH:-.}"
allowed_tools="${INPUT_ALLOWED_TOOLS:-}"
if [ ! -d "$scan_path" ]; then
  echo "scan path does not exist: $scan_path" >&2
  exit 2
fi
matches="$(grep -RInE '(sk-[A-Za-z0-9_-]{20,}|AKIA[0-9A-Z]{16}|password[[:space:]]*[:=])' "$scan_path" --exclude-dir=.git || true)"
if [ -n "$matches" ]; then
  echo "Potential secret detected:"
  echo "$matches"
  exit 1
fi
if [ -n "$allowed_tools" ]; then
  tools="$(find "$scan_path" -type f -name '*.json' -exec grep -hoE '"name"[[:space:]]*:[[:space:]]*"[^"]+"' {} + 2>/dev/null || true)"
  for item in $tools; do
    tool="$(echo "$item" | sed -E 's/.*"([^"]+)"$/\1/')"
    case ",$allowed_tools," in *",$tool,"*) ;; *) echo "Unauthorized MCP tool: $tool" >&2; exit 1;; esac
  done
fi
echo "MCP scan passed"
