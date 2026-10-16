package render

import "testing"

func TestRecursive(t *testing.T) {
	data := map[string]any{
		"name":    "etcd",
		"version": "v1",
		"src":     "{{ name }}-{{ version }}",
	}
	out, err := Recursive("{{ src }}/{{ undefined }}{% for x in missing %}{{ x }}{% endfor %}", data)
	if err != nil {
		t.Fatal(err)
	}
	if out != "etcd-v1/" {
		t.Errorf("got %q", out)
	}
}

func TestRecursiveDoesNotLoopForever(t *testing.T) {
	data := map[string]any{"a": "x{{ a }}"}
	if _, err := Recursive("{{ a }}", data); err == nil {
		t.Error("expected an error for a template that never stabilizes")
	}
}
