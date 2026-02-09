# Magefiles (isolated)

This directory contains the project's Mage tasks and a dedicated `go.mod` so Mage's dependencies are isolated from the main module.

Usage:

- From the repo root:
  ```bash
  # Run mage tasks directly — mage auto-detects ./magefiles
  mage <target>
  # optional: mage -d ./magefiles <target>
  ```
