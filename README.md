# aicommits

`aicommits` (binary: `aic`) is a small Go CLI that generates concise Conventional Commit messages from your staged git diff using Google Gemini, then optionally runs `git commit` for you.

## Requirements

- Go (for building from source)
- Git
- A Google Gemini API key

## Installation

### Option 1: Install script

```bash
./install.sh
```

This builds `aic` and installs it to `~/.local/bin`.

### Option 2: Build manually

```bash
go build -o aic main.go
```

Then move `aic` somewhere in your `PATH`.

## Configure API key

Save your API key once:

```bash
aic -api <your_key>
```

This stores the key at `~/.config/aicommits/apikey` with user-only permissions.  
Alternatively, you can provide `GOOGLE_API_KEY` as an environment variable.

## Usage

1. Stage your changes:

   ```bash
   git add .
   ```

2. Run:

   ```bash
   aic
   ```

The tool:
- reads `git diff --cached`
- asks Gemini for a concise Conventional Commit message
- shows the message and prompts for confirmation before committing

### Flags

- `-api <key>`: save API key and exit
- `-p`: print generated commit message only (does not commit)
- `-y`: auto-apply commit without prompt

## Ignore file

You can add a `.aicomignore` file in your repository root to exclude files/paths from staged diff processing.

Example:

```txt
# comments are allowed
docs/
*.md
```

## Notes

- If no staged changes are found, the tool exits with: `No changes to commit.`
- The generated message style is influenced by your last 10 commit subjects.
