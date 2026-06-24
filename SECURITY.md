# Security Policy

## Reporting a vulnerability

If you've found a security issue in Blueprint, please email
**security@inspireailab.com** instead of opening a public issue.

We'll respond within two business days to acknowledge receipt, and we aim to
have a fix or a coordinated disclosure plan within 14 days for confirmed
issues.

## What's in scope

- The Blueprint CLI binary and its dependencies
- The runtime installer (asset selection, archive extraction, file placement)
- The HTTP downloader (TLS handling, integrity, path traversal)
- The install scripts (`install.sh`, `install.ps1`) hosted at
  blueprint.inspireailab.com

## What's out of scope

- llama.cpp itself — report at [github.com/ggml-org/llama.cpp/issues](https://github.com/ggml-org/llama.cpp/issues)
- Model weights / GGUF files — these come from HuggingFace
- The Inspire AI Lab marketing site at inspireailab.com — separate codebase

## Disclosure

We follow coordinated disclosure. Once a fix is released we'll credit the
reporter (unless asked otherwise) in the release notes.
