#!/bin/sh
# Point this clone's git hooks at the committed .githooks/ directory.
set -e
root=$(git rev-parse --show-toplevel)
git -C "$root" config core.hooksPath .githooks
echo "✓ core.hooksPath set to .githooks — manifest validation will run on commit."
