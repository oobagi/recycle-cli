# recycle

A safe `rm` replacement for macOS. Moves files to the native Trash instead of permanently deleting them. Files appear in Finder's Trash with full "Put Back" support.

## Install

```bash
go install github.com/jaden/recycle-cli/cmd/recycle@latest
```

Or build from source:

```bash
git clone https://github.com/jaden/recycle-cli.git
cd recycle-cli
go build -o recycle ./cmd/recycle/
sudo mv recycle /usr/local/bin/
```

## Usage

```bash
# Trash files
recycle file.txt
recycle -r some-directory/
recycle -rf node_modules/

# List trashed files (with preview)
recycle --list

# Restore a file to its original location
recycle --restore file.txt

# Permanently empty the trash
recycle --empty
```

## Alias as rm

Add to your `~/.zshrc` or `~/.bashrc`:

```bash
alias rm='recycle'
```

Then `rm file.txt` moves to Trash instead of deleting. Use `/bin/rm` when you need real deletion.

## Agentic safety net

AI coding agents (Claude Code, Cursor, Copilot, etc.) frequently run `rm` and `rm -rf` to clean up files, reset state, or retry builds. With the alias in place, every destructive `rm` an agent executes becomes a recoverable Trash operation. If an agent accidentally deletes your source code, config, or anything else — it's sitting in the Trash waiting to be restored. No custom tool integration required; the alias works transparently.

## How it works

- **Trash** — uses Finder's AppleScript interface (`tell application "Finder" to delete`), so files get native "Put Back" support
- **Restore** — stores the original path as an extended attribute (`com.recycle.original-path`) on the trashed file
- **List** — asks Finder for trash contents, shows inline file previews
- **Empty** — tells Finder to empty the trash

No config files, no metadata directories. Just a Go binary wrapping macOS native APIs.

## Flags

| Flag | Description |
|------|-------------|
| `-r` | Recursive — required for directories |
| `-f` | Force — silently skip files that don't exist |
| `-rf` / `-fr` | Both combined |

## License

MIT
