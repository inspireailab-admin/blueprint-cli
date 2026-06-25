# Blueprint

[![Test](https://github.com/inspireailab-admin/blueprint-cli/actions/workflows/test.yml/badge.svg)](https://github.com/inspireailab-admin/blueprint-cli/actions/workflows/test.yml)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Reference](https://pkg.go.dev/badge/github.com/inspireailab-admin/blueprint-cli.svg)](https://pkg.go.dev/github.com/inspireailab-admin/blueprint-cli)

**Local install for open LLMs.** Pull a model, serve it, chat with it â€” all on your own hardware. Free, no telemetry, no account.

> Made by [Inspire AI Lab](https://inspireailab.com). Companion to the [Blueprint planner](https://blueprint.inspireailab.com), which sizes the hardware and cost before you commit to a model.

```sh
# macOS, Linux
curl -sSL https://blueprint.inspireailab.com/install.sh | sh

# Windows (PowerShell)
iwr -useb https://blueprint.inspireailab.com/install.ps1 | iex

# Then:
blueprint runtime install              # one-time, ~16 MB llama.cpp release
blueprint pull qwen-2.5-7b-instruct    # ~4.4 GB Q4 GGUF
blueprint serve qwen-2.5-7b-instruct   # http://127.0.0.1:8080/v1
```

That's it. The model is now running locally with an OpenAI-compatible API.

## What it does

Blueprint wraps [llama.cpp](https://github.com/ggml-org/llama.cpp) with three concerns the raw binary leaves to you:

- **Curated catalog** â€” a hand-picked set of open models we've verified work end-to-end: correct chat template, sane quant variants, working GGUF source.
- **Cross-platform runtime install** â€” fetches the right llama.cpp release for your OS and CPU architecture into `~/.blueprint/bin/`.
- **Resumable model downloads** â€” pulls GGUF weights from HuggingFace with a progress bar and Range-based resume.

What llama.cpp does, Blueprint does not re-do. We don't ship our own inference engine. We make the existing one easy.

## Commands

```
blueprint pull [model-id]              List the catalog, or pull a model's GGUF
  --quant q4|q8|q3|fp16                Which weight quantization (default q4)

blueprint serve [model-id]             Run llama-server against a pulled model
  --port 8080                          Port to listen on
  --host 127.0.0.1                     Host to bind (don't expose publicly without auth)
  --ctx-size 4096                      Context window in tokens
  --n-gpu-layers 999                   Layers to offload to GPU (0 = CPU only)
  --threads 0                          CPU threads (0 = let llama-server decide)

blueprint runtime install              Download + install llama.cpp into ~/.blueprint/bin/
blueprint runtime version              Show the installed llama.cpp tag
blueprint runtime path                 Print the resolved llama-server path

blueprint status                       List pulled models and their sizes
blueprint version                      Print the CLI version
```

Run `blueprint --help` or `blueprint <command> --help` for the full reference.

## Supported models

The catalog is curated by hand â€” see [`internal/catalog/models.json`](internal/catalog/models.json) for the full current list and [CONTRIBUTING.md](CONTRIBUTING.md) for how to add one.

| Family | Sizes | License notes |
|---|---|---|
| Llama 3.1 / 3.3 | 8B, 70B | Community licenses; HuggingFace gating |
| Qwen 2.5 | 7B, 14B, 32B, 72B | Apache 2.0 (â‰¤32B); 72B is non-commercial |
| Qwen 2.5 Coder | 32B | Apache 2.0 |
| Gemma 2 | 9B, 27B | Gemma Terms â€” review for commercial use |
| Phi-4 | 14B | MIT |
| Mixtral 8x7B | 47B MoE / 13B active | Apache 2.0 |

Run `blueprint pull` with no arguments for the live list including which quants are published.

## Where things live

```
~/.blueprint/
â”œâ”€â”€ bin/
â”‚   â”œâ”€â”€ llama-server[.exe]      llama.cpp runtime
â”‚   â”œâ”€â”€ ggml*.{dll,dylib,so}    runtime libraries
â”‚   â””â”€â”€ VERSION                 installed llama.cpp tag
â””â”€â”€ models/
    â””â”€â”€ <model-id>/
        â””â”€â”€ <file>.gguf         pulled weights
```

Override with `BLUEPRINT_HOME=/some/path`.

## Building from source

Requires Go 1.22+.

```sh
git clone https://github.com/inspireailab-admin/blueprint-cli.git
cd blueprint
go build -o blueprint .
./blueprint version
```

Cross-compile all six release platforms:

```sh
./scripts/build-release.sh v0.1.0
ls dist/
# blueprint-darwin-amd64, blueprint-darwin-arm64,
# blueprint-linux-amd64,  blueprint-linux-arm64,
# blueprint-windows-amd64.exe, blueprint-windows-arm64.exe
```

Tagged pushes (`git tag vX.Y.Z && git push --tags`) trigger the release workflow which cross-compiles and uploads to GitHub Releases automatically.

## License

[Apache 2.0](LICENSE). Same license as llama.cpp.

## Security

Found a vulnerability? Email **security@inspireailab.com** â€” please don't open a public issue. See [SECURITY.md](SECURITY.md).

## Why this exists

Inspire AI Lab is an AI engineering consultancy. Blueprint is one of two free tools we maintain to make it easy to evaluate private AI before talking to us:

- **Blueprint Planner** at [blueprint.inspireailab.com](https://blueprint.inspireailab.com) â€” pick a model, see the VRAM math, see what the hardware costs on-prem or cloud.
- **Blueprint CLI** (this repo) â€” actually run it locally, no infrastructure required.

If you'd like help getting any of this into production, [book a 30-min consultation](https://inspireailab.com/contact).
