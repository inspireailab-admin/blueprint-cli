# Contributing to Blueprint

Thanks for considering a contribution. The two most common contributions are
**adding a model** to the catalog and **fixing a bug or rough edge in the CLI**.
Both are welcome.

## Adding a model

The catalog at [`internal/catalog/models.json`](internal/catalog/models.json) is
hand-curated. The bar is "we've verified this model actually works end-to-end" —
correct chat template, sane quant variants from a trustworthy GGUF source.

To add a model, open a PR that adds an entry like:

```json
{
  "id": "qwen-2.5-7b-instruct",
  "displayName": "Qwen 2.5 7B Instruct",
  "family": "Qwen 2.5",
  "params": 7,
  "totalParams": 7,
  "license": "apache-2.0",
  "ggufRepo": "bartowski/Qwen2.5-7B-Instruct-GGUF",
  "ggufFiles": {
    "q3": "Qwen2.5-7B-Instruct-Q3_K_L.gguf",
    "q4": "Qwen2.5-7B-Instruct-Q4_K_M.gguf",
    "q8": "Qwen2.5-7B-Instruct-Q8_0.gguf",
    "fp16": "Qwen2.5-7B-Instruct-f16.gguf"
  }
}
```

Field reference:

| Field | What it is |
|---|---|
| `id` | Kebab-case stable id you'd pass to `blueprint pull <id>` |
| `displayName` | User-facing name shown in `blueprint pull` output |
| `family` | Used for grouping in the planner UI |
| `params` | Active parameter count in billions (for MoE, the active count) |
| `totalParams` | Total parameter count in billions (for MoE, the full count) |
| `license` | Short license id — see `internal/catalog/catalog.go` for known ids |
| `ggufRepo` | HuggingFace repo containing the GGUF quants. Stick to trustworthy sources (bartowski, unsloth, official model orgs). |
| `ggufFiles` | Map of quant key → filename in that repo. Only list quants you've verified |

### Before opening the PR — verification checklist

We can only support models that actually work. Please confirm:

- [ ] **Pulled it.** `blueprint pull <id> --quant q4` succeeds and the file is the right size.
- [ ] **Served it.** `blueprint serve <id>` boots without errors, `/health` returns 200, and a simple chat completion request through `/v1/chat/completions` returns coherent output.
- [ ] **Chat template is right.** The output isn't garbled with special tokens (`<|im_start|>`, `<|start_header_id|>`, etc.). If garbled, the embedded chat template in the GGUF is wrong and the model isn't ready to add.
- [ ] **License is sane.** If the license restricts commercial use or distribution, note it in the PR description.

In the PR description, paste the first prompt + response you got. Sanity check, that's all.

### Don't add

- **Experimental quants** (IQ3_XXS, IQ2, etc.) — they have wide quality variance. Stick to standard K-quants and FP16.
- **Random users' GGUF re-quantizations** — quality is unpredictable. Use original-publisher quants or trusted re-quantizers only.
- **Untested chat models** — if the response is gibberish, the model isn't ready.

## Fixing the CLI

```sh
git clone https://github.com/inspireailab-admin/blueprint.git
cd blueprint
go test ./...
go build -o blueprint .
./blueprint --help
```

We use Go 1.22+. Standard `go fmt` + `go vet` apply. The CI workflow runs the
test suite on macOS, Linux, and Windows on every push and PR.

For substantial changes, open an issue first so we can talk about the approach.
For small fixes, just open a PR — we'll review.

## Project layout

```
main.go                  Entrypoint
internal/
  cmd/                   Cobra subcommands (one file each)
  catalog/               Embedded model catalog + lookup
  download/              Resumable HTTP downloader (no external deps)
  paths/                 ~/.blueprint cross-platform paths
  runtime/               llama.cpp runtime install + lookup
scripts/
  build-release.sh       Cross-compile for all six release platforms
```

## Code of conduct

This project follows the [Contributor Covenant](CODE_OF_CONDUCT.md).
Be kind, be patient, assume good faith.
