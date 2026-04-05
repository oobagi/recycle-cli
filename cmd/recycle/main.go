package main

/*
#include <stdlib.h>
#include <string.h>
#include <sys/xattr.h>

int setOrigPath(const char* trashPath, const char* origPath) {
	return setxattr(trashPath, "com.recycle.original-path", origPath, strlen(origPath), 0, 0);
}

char* getOrigPath(const char* trashPath) {
	char buf[4096];
	ssize_t len = getxattr(trashPath, "com.recycle.original-path", buf, sizeof(buf), 0, 0);
	if (len < 0) return NULL;
	buf[len] = '\0';
	return strdup(buf);
}
*/
import "C"

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unsafe"
)

func trashRoot() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "recycle: cannot determine home directory: %v\n", err)
		os.Exit(1)
	}
	return filepath.Join(home, ".Trash")
}

// finderTrash tells Finder to move a file to Trash, returning the resulting path in ~/.Trash/.
// Finder handles Put Back metadata natively.
func finderTrash(absPath string) (string, error) {
	script := fmt.Sprintf(
		`tell application "Finder" to return POSIX path of ((delete POSIX file %q) as alias)`,
		absPath)
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg != "" {
			return "", fmt.Errorf("%s", msg)
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func setOriginalPath(trashItemPath, originalPath string) {
	cTrash := C.CString(trashItemPath)
	defer C.free(unsafe.Pointer(cTrash))
	cOrig := C.CString(originalPath)
	defer C.free(unsafe.Pointer(cOrig))
	C.setOrigPath(cTrash, cOrig)
}

func getOriginalPath(trashItemPath string) string {
	cPath := C.CString(trashItemPath)
	defer C.free(unsafe.Pointer(cPath))
	result := C.getOrigPath(cPath)
	if result == nil {
		return ""
	}
	defer C.free(unsafe.Pointer(result))
	return C.GoString(result)
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		printUsage()
		os.Exit(0)
	}

	switch args[0] {
	case "--help", "-h":
		printUsage()
		return
	case "--version":
		fmt.Println("recycle 2.0.0")
		return
	case "--list":
		cmdList()
		return
	case "--restore":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "recycle: --restore requires an argument")
			os.Exit(1)
		}
		cmdRestore(args[1])
		return
	case "--empty":
		cmdEmpty()
		return
	}

	force := false
	recursive := false
	var files []string

	for _, arg := range args {
		if strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") && len(arg) > 1 {
			for _, ch := range arg[1:] {
				switch ch {
				case 'f':
					force = true
				case 'r', 'R':
					recursive = true
				default:
					fmt.Fprintf(os.Stderr, "recycle: unknown flag -%c\n", ch)
					os.Exit(1)
				}
			}
		} else if arg == "--" {
			continue
		} else {
			files = append(files, arg)
		}
	}

	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "recycle: missing operand")
		os.Exit(1)
	}

	exitCode := 0
	trashed := 0
	for _, f := range files {
		if err := trashFile(f, force, recursive); err != nil {
			fmt.Fprintf(os.Stderr, "recycle: %v\n", err)
			exitCode = 1
		} else {
			trashed++
		}
	}

	if trashed > 0 {
		if trashed == 1 {
			fmt.Printf("Recycled %s\n", files[0])
		} else {
			fmt.Printf("Recycled %d items\n", trashed)
		}
	}

	os.Exit(exitCode)
}

func trashFile(path string, force, recursive bool) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) && force {
			return nil
		}
		return fmt.Errorf("cannot stat '%s': %v", path, err)
	}

	if info.IsDir() && !recursive {
		return fmt.Errorf("cannot remove '%s': is a directory (use -r)", path)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("cannot resolve '%s': %v", path, err)
	}

	trashPath, err := finderTrash(absPath)
	if err != nil {
		return fmt.Errorf("cannot move '%s' to trash: %v", path, err)
	}

	setOriginalPath(trashPath, absPath)
	return nil
}

func cmdList() {
	out, err := exec.Command("osascript", "-e",
		`tell application "Finder" to get name of every item of trash`).Output()
	if err != nil {
		fmt.Println("Trash is empty.")
		return
	}

	raw := strings.TrimSpace(string(out))
	if raw == "" {
		fmt.Println("Trash is empty.")
		return
	}

	names := strings.Split(raw, ", ")
	root := trashRoot()

	for _, name := range names {
		fullPath := filepath.Join(root, name)
		display := name
		isDir := false
		ago := ""
		if info, err := os.Lstat(fullPath); err == nil {
			isDir = info.IsDir()
			ago = timeAgo(info.ModTime())
		}
		if isDir {
			display += "/"
		}
		original := getOriginalPath(fullPath)
		if original == "" {
			original = "-"
		}
		fmt.Printf("%-30s  %-12s  %s\n", display, ago, original)
		if preview := filePreview(fullPath, isDir); preview != "" {
			fmt.Printf("  \033[2m%s\033[0m\n", preview)
		}
	}
}

func filePreview(path string, isDir bool) string {
	if isDir {
		entries, err := os.ReadDir(path)
		if err != nil {
			return ""
		}
		var names []string
		for i, e := range entries {
			if i >= 5 {
				names = append(names, fmt.Sprintf("… +%d more", len(entries)-5))
				break
			}
			n := e.Name()
			if e.IsDir() {
				n += "/"
			}
			names = append(names, n)
		}
		return strings.Join(names, "  ")
	}

	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	buf := make([]byte, 120)
	n, err := f.Read(buf)
	if n == 0 {
		return ""
	}

	// Check for binary content.
	for _, b := range buf[:n] {
		if b == 0 {
			return "(binary)"
		}
	}

	line := strings.SplitN(string(buf[:n]), "\n", 2)[0]
	if len(line) > 80 {
		line = line[:80] + "…"
	}
	return line
}

func timeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(math.Round(d.Minutes()))
		if m == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d min ago", m)
	case d < 24*time.Hour:
		h := int(math.Round(d.Hours()))
		if h == 1 {
			return "1 hr ago"
		}
		return fmt.Sprintf("%d hr ago", h)
	default:
		days := int(math.Round(d.Hours() / 24))
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}

func cmdRestore(name string) {
	trashPath := filepath.Join(trashRoot(), name)

	if _, err := os.Lstat(trashPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "recycle: cannot find '%s' in trash\n", name)
		os.Exit(1)
	}

	original := getOriginalPath(trashPath)
	if original == "" {
		fmt.Fprintf(os.Stderr, "recycle: cannot determine original path for '%s'\n", name)
		os.Exit(1)
	}

	parentDir := filepath.Dir(original)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "recycle: cannot create directory '%s': %v\n", parentDir, err)
		os.Exit(1)
	}

	if _, err := os.Lstat(original); err == nil {
		fmt.Fprintf(os.Stderr, "recycle: cannot restore — '%s' already exists\n", original)
		os.Exit(1)
	}

	if err := os.Rename(trashPath, original); err != nil {
		fmt.Fprintf(os.Stderr, "recycle: cannot restore '%s': %v\n", name, err)
		os.Exit(1)
	}

	fmt.Printf("Restored %s -> %s\n", name, original)
}

func cmdEmpty() {
	out, err := exec.Command("osascript", "-e", `tell application "Finder" to empty trash`).CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "recycle: cannot empty trash: %s\n", strings.TrimSpace(string(out)))
		os.Exit(1)
	}
	fmt.Println("Emptied trash")
}

func printUsage() {
	fmt.Print(`recycle — safe rm replacement

Usage:
  recycle [flags] <file ...>    Move files to trash
  recycle --list                Show trashed files
  recycle --restore <name>      Restore a file from trash
  recycle --empty               Permanently delete all trash

Flags:
  -r    Recursive (required for directories)
  -f    Force (ignore nonexistent files)

Trash location: ~/.Trash/ (macOS native — files appear in Finder)

Alias as rm:
  alias rm='recycle'
`)
}
