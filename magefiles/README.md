# Magefiles (isolated)

This directory contains the project's Mage tasks and a dedicated `go.mod` so Mage's dependencies are isolated from the main module.

Usage:

- From the repo root:
  ```bash
  # Run mage tasks directly — mage auto-detects ./magefiles
  mage <target>
  # optional: mage -d ./magefiles <target>
  ```

Notes:
- Paths used in the tasks are relative to the repository root (the mage tasks use `..` where appropriate).
- The original `magefile.go` at the repo root is disabled and kept for history; use the files in this directory instead.
