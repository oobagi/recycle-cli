# recycle

A safe `rm` replacement for macOS. Moves files to the native Trash instead of permanently deleting them — with full Finder "Put Back" support.

Great as a safety net for AI coding agents (Claude Code, Cursor, Copilot) that run `rm -rf` — everything stays recoverable.

## Install

```bash
brew install oobagi/tap/recycle
```

Then alias it in `~/.zshrc`:

```bash
alias rm='recycle'
```

## Usage

```bash
rm file.txt                # trash a file
rm -rf node_modules/       # trash a directory
rm --list                  # see what's in trash (with previews)
rm --restore file.txt      # put it back
rm --empty                 # permanently empty trash
```

Use `/bin/rm` when you actually need permanent deletion.

## How it works

All operations go through Finder's native APIs — no config files, no metadata directories.

- **Trash** → `tell application "Finder" to delete` (native Put Back)
- **Restore** → xattr stores original path on the trashed file
- **List/Empty** → Finder AppleScript

## License

MIT
