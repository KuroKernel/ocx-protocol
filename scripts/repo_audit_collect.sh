#!/usr/bin/env bash
set -euo pipefail
mkdir -p docs/audit/raw

# Repo tree (ignore heavy dirs)
tree -a -I '.git|node_modules|vendor|target|dist|build|out|coverage|bin|tmp|.cache|site|personal|ocx.world-site' > docs/audit/raw/repo_tree.txt || true

# Go
if command -v go >/dev/null; then
  go version > docs/audit/raw/go.txt
  go list ./... > docs/audit/raw/go_packages.txt || true
  go mod graph > docs/audit/raw/go_mod_graph.txt || true
fi

# Rust
if command -v cargo >/dev/null; then
  cargo metadata --format-version=1 > docs/audit/raw/cargo_metadata.json || true
  cargo tree -e normal > docs/audit/raw/cargo_tree.txt || true
fi

# Node
if command -v npm >/dev/null; then
  node -v > docs/audit/raw/node.txt
  npm ls --all --json > docs/audit/raw/npm_ls.json || true
fi

# Java/Maven
if command -v mvn >/dev/null; then
  mvn -q -DskipTests dependency:tree > docs/audit/raw/mvn_dep_tree.txt || true
fi

# C++ (CMake)
if [ -f "CMakeLists.txt" ]; then
  echo "CMake project present" > docs/audit/raw/cpp.txt
fi

# CI
cp -r .github/workflows docs/audit/raw/ci 2>/dev/null || true

echo "OK"
