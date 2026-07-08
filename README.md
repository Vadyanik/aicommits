# aicommits

`aicommits` (`aic`) is a small Go CLI that generates concise commit messages from your **staged Git diff** using AI, then optionally runs `git commit` for you.

It supports:
- **Gemini** (default, via `GOOGLE_API_KEY`)
- **Ollama** (local models, with optional automatic fallback)

---

## Why this tool?

Writing clear commit messages repeatedly can be slow and inconsistent.  
`aic` helps by:
- reading your staged changes
- looking at your last 10 commit subjects for style
- generating a short Conventional Commit message
- letting you confirm before commit (or auto-commit with a flag)

---

## Features

- ✅ Generates commit messages from `git diff --cached`
- ✅ Uses **Conventional Commits** style prompt
- ✅ Tries to mimic recent repository commit style
- ✅ Interactive confirmation before committing
- ✅ `-y` mode for non-interactive workflows
- ✅ `-p` mode to print message only
- ✅ Local Ollama support with saved model selection
- ✅ Optional `.aicomignore` support to exclude files from diff context
- ✅ Gemini → Ollama fallback when Gemini is unavailable

---

## Requirements

- Git
- Go (to build from source)
- One of:
  - Gemini API key, or
  - Ollama running locally (or via `OLLAMA_HOST`)

---

## Installation

### Option 1: Install using script

From repository root:

```bash
./install.sh
```

This builds `aic` and places it in:

```text
~/.local/bin/aic
```

Make sure `~/.local/bin` is in your `PATH`.

---

### Option 2: Manual build

```bash
go build -o aic main.go
mkdir -p ~/.local/bin
mv aic ~/.local/bin/
```

---

### Option 3: Run directly without installing

```bash
go run main.go --help
```

---

## Configuration

### Gemini API key

Set and persist your key:

```bash
aic -api <your_google_api_key>
```

The key is stored at:

```text
~/.config/aicommits/apikey
```

If no saved key exists, `aic` will also read:

```text
GOOGLE_API_KEY
```

---

### Ollama setup

Use Ollama mode with:

```bash
aic -o
```

Behavior:
- On first use, `aic` fetches available models from Ollama (`/api/tags`)
- Prompts you to choose one
- Saves selection to:

```text
~/.config/aicommits/ollama-model
```

By default, Ollama endpoint is:

```text
http://localhost:11434
```

Override with:

```bash
export OLLAMA_HOST=http://your-host:11434
```

---

## Usage

### Standard workflow

1. Stage changes:

```bash
git add .
```

2. Generate and confirm commit:

```bash
aic
```

You’ll see the generated message and a prompt:

```text
Apply this commit? (Y/n):
```

---

### Print only (no commit)

```bash
aic -p
```

Useful for previewing or editing manually.

---

### Auto-commit without prompt

```bash
aic -y
```

Good for fast local workflows after reviewing staged changes yourself.

---

### Force Ollama

```bash
aic -o
```

---

## CLI flags

| Flag | Description |
|------|-------------|
| `-y` | Automatically apply commit without confirmation |
| `-p` | Print commit message only; do not commit |
| `-o` | Use local Ollama model instead of Gemini |
| `-api <key>` | Save Gemini API key to config and exit |

---

## `.aicomignore`

Create a `.aicomignore` file in repo root to exclude paths from the diff context used for message generation.

Example:

```gitignore
# Ignore generated files
dist/
coverage/
*.lock
```

Notes:
- Empty lines are ignored
- Lines starting with `#` are comments
- Patterns are applied as Git diff excludes for staged changes

---

## How message generation works

At a high level, `aic`:
1. Reads staged diff (`git diff --cached -- .`)
2. Applies `.aicomignore` exclusions (if present)
3. Reads last 10 commit subjects (`git log -n 10 --format=%s`)
4. Builds a concise prompt that asks for Conventional Commit style
5. Calls Gemini (default) or Ollama (`-o`)
6. If Gemini fails, automatically tries Ollama (unless `-p` suppresses extra noise)
7. Prints message and optionally commits with `git commit -m "<message>"`

---

## Exit/error behavior

- If no staged changes exist:
  - `aic` prints: `No changes to commit.`
  - with `-p`, this message goes to stderr
- If Gemini key is missing and `-o` is not used, command exits with error
- If Ollama is selected but unavailable, command exits with Ollama error details
- If `git commit` fails, command prints combined Git output and exits non-zero

---

## Troubleshooting

### “API key not found”

Run:

```bash
aic -api <your_google_api_key>
```

or use:

```bash
aic -o
```

### “No Ollama models found”

Pull a model first:

```bash
ollama pull llama3.1
```

Then run `aic -o` again.

### Ollama connection errors

- Ensure Ollama server is running
- Verify `OLLAMA_HOST`
- Test endpoint manually:

```bash
curl "$OLLAMA_HOST/api/tags"
```

### “No changes to commit”

`aic` only uses staged files. Run `git add` first.

---

## Development

### Build

```bash
go build -o aic main.go
```

### Format (recommended)

```bash
go fmt ./...
```

### Dependency management

```bash
go mod tidy
```

---

## Security and privacy notes

- Your staged diff content is sent to the selected AI backend (Gemini or Ollama).
- Gemini calls are remote; Ollama can run fully local.
- API key is stored in your user config directory with file mode `0600`.
- Avoid staging secrets before running the tool.

---

## License

No license file is currently present in this repository. Add a `LICENSE` file if you want explicit reuse terms.

