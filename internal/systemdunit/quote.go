// Package systemdunit safely formats values and command lines for systemd units.
package systemdunit

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// Quote formats a directive value using systemd.syntax quoting and escapes
// percent specifiers so paths and descriptions are reproduced literally.
func Quote(value string) (string, error) {
	if err := validateText(value); err != nil {
		return "", err
	}
	return quote(strings.ReplaceAll(value, "%", "%%")), nil
}

// Command formats argv for an ExecStart-style directive. Unlike a shell
// command, systemd parses these arguments itself; dollar signs must therefore
// be doubled to prevent environment expansion.
func Command(args ...string) (string, error) {
	if len(args) == 0 {
		return "", errors.New("systemd command has no executable")
	}
	quoted := make([]string, len(args))
	for index, arg := range args {
		if err := validateText(arg); err != nil {
			return "", fmt.Errorf("argument %d: %w", index, err)
		}
		arg = strings.ReplaceAll(arg, "%", "%%")
		arg = strings.ReplaceAll(arg, "$", "$$")
		quoted[index] = quote(arg)
	}
	return strings.Join(quoted, " "), nil
}

func quote(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}

func validateText(value string) error {
	for _, char := range value {
		if unicode.IsControl(char) {
			return fmt.Errorf("value contains control character %U", char)
		}
	}
	return nil
}
