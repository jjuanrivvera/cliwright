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
// echoes the secret in plaintext and hangs on long pastes (API keys are often long JWTs).
func promptSecret(cmd *cobra.Command, label string) (string, error) {
	fmt.Fprint(cmd.ErrOrStderr(), label)
	if f, ok := cmd.InOrStdin().(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		b, err := term.ReadPassword(int(f.Fd()))
		fmt.Fprintln(cmd.ErrOrStderr())
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(b)), nil
	}
	return promptLine(cmd, "")
}
