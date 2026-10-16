// Package render renders Jinja2 templates, recursively, so that Jinja in Jinja is rendered too.
package render

import (
	"fmt"
	"io"

	"github.com/nikolalohinski/gonja/v2"
	"github.com/nikolalohinski/gonja/v2/exec"
)

// MaxIterations limits the number of render passes, protecting against templates that never stabilize.
const MaxIterations = 100

func init() {
	// gonja logs through logrus by default, which is noise for a CLI tool
	gonja.SetLoggerOutput(io.Discard)
}

// Render renders a Jinja2 template once with data as context.
func Render(tpl string, data map[string]any) (string, error) {
	t, err := gonja.FromString(tpl)
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}
	out, err := t.ExecuteToString(exec.NewContext(data))
	if err != nil {
		return "", fmt.Errorf("rendering template: %w", err)
	}
	return out, nil
}

// Recursive renders a template repeatedly until the output no longer changes.
// This allows values in data to contain Jinja expressions themselves.
func Recursive(tpl string, data map[string]any) (string, error) {
	prev := tpl
	for range MaxIterations {
		curr, err := Render(prev, data)
		if err != nil {
			return "", err
		}
		if curr == prev {
			return curr, nil
		}
		prev = curr
	}
	return "", fmt.Errorf("template did not stabilize after %d render passes", MaxIterations)
}
