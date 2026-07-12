package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// promptLine prints label to stderr and reads one line from stdin (trimmed). It reads one
// byte at a time rather than buffering, so successive prompts on the same reader don't lose
// input a buffered reader would have read ahead and discarded. Use this for non-secret input
// (base URL, y/n confirmations) — never fmt.Scanln, which echoes and stalls on long pastes.
func promptLine(cmd *cobra.Command, label string) (string, error) {
	fmt.Fprint(cmd.ErrOrStderr(), label)
	r := cmd.InOrStdin()
	var b strings.Builder
	buf := make([]byte, 1)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			if buf[0] == '\n' {
				break
			}
			b.WriteByte(buf[0])
		}
		if err != nil {
			if b.Len() == 0 {
				return "", err
			}
			break
		}
	}
	return strings.TrimSpace(b.String()), nil
}

// promptSecret reads a secret (token, API key, password, OAuth code) WITHOUT echoing when
// stdin is a terminal, so it never lands in scrollback; on a pipe it falls back to a normal
// line read so scripts still work. ALWAYS read secrets through this — never fmt.Scanln, which
// echoes the secret in plaintext.
func promptSecret(cmd *cobra.Command, label string) (string, error) {
	fmt.Fprint(cmd.ErrOrStderr(), label)
	if f, ok := cmd.InOrStdin().(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		s, err := readSecretRaw(f)
		fmt.Fprintln(cmd.ErrOrStderr()) // raw mode doesn't echo the Enter; end the prompt line
		if err != nil {
			return "", err
		}
		return sanitizeSecret(s), nil
	}
	return promptLine(cmd, "")
}

// readSecretRaw reads one line from f in raw, no-echo mode until CR/LF, with no line-length
// limit. term.ReadPassword reads in CANONICAL mode, whose line buffer is capped at MAX_CANON
// (1024 bytes on macOS): pasting a longer secret (a ~970-char JWT, say) fills the buffer and the
// terminal BLOCKS further input — the "prompt hangs until Ctrl-C" bug. Raw mode has no such
// limit, so long pasted keys read cleanly. Ctrl-C cancels; Backspace/DEL edits.
func readSecretRaw(f *os.File) (string, error) {
	fd := int(f.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return "", err
	}
	defer func() { _ = term.Restore(fd, oldState) }()

	var buf []byte
	chunk := make([]byte, 256)
	for {
		n, readErr := f.Read(chunk)
		for i := 0; i < n; i++ {
			switch c := chunk[i]; c {
			case '\r', '\n':
				return string(buf), nil
			case 3: // Ctrl-C
				return "", fmt.Errorf("cancelled")
			case 127, 8: // DEL / Backspace
				if len(buf) > 0 {
					buf = buf[:len(buf)-1]
				}
			default:
				buf = append(buf, c)
			}
		}
		if readErr != nil {
			if len(buf) == 0 {
				return "", readErr
			}
			return string(buf), nil
		}
	}
}

// sanitizeSecret strips terminal bracketed-paste markers (ESC[200~ … ESC[201~) and trims
// surrounding whitespace. With bracketed paste enabled, a raw read (unlike the shell's line
// editor) receives those wrappers around pasted text; left in, they corrupt a pasted key so it
// fails auth with a 401. Stripping them fixes the common "typing works, pasting fails".
func sanitizeSecret(s string) string {
	s = strings.ReplaceAll(s, "\x1b[200~", "")
	s = strings.ReplaceAll(s, "\x1b[201~", "")
	return strings.TrimSpace(s)
}
