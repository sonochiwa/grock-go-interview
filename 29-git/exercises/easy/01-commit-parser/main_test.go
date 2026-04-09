package commit_parser

import (
	"errors"
	"testing"
)

func TestParseWithScope(t *testing.T) {
	c, err := ParseCommit("feat(auth): add JWT login")
	if err != nil {
		t.Fatal(err)
	}
	if c.Type != "feat" {
		t.Errorf("Type = %q", c.Type)
	}
	if c.Scope != "auth" {
		t.Errorf("Scope = %q", c.Scope)
	}
	if c.Description != "add JWT login" {
		t.Errorf("Description = %q", c.Description)
	}
	if c.Breaking {
		t.Error("should not be breaking")
	}
}

func TestParseWithoutScope(t *testing.T) {
	c, err := ParseCommit("fix: handle nil pointer")
	if err != nil {
		t.Fatal(err)
	}
	if c.Type != "fix" || c.Scope != "" || c.Description != "handle nil pointer" {
		t.Errorf("unexpected: %+v", c)
	}
}

func TestParseBreaking(t *testing.T) {
	c, err := ParseCommit("feat(api)!: remove v1 endpoints")
	if err != nil {
		t.Fatal(err)
	}
	if !c.Breaking {
		t.Error("should be breaking")
	}
	if c.Type != "feat" || c.Scope != "api" {
		t.Errorf("unexpected: %+v", c)
	}
}

func TestParseBreakingNoScope(t *testing.T) {
	c, err := ParseCommit("refactor!: rewrite core")
	if err != nil {
		t.Fatal(err)
	}
	if !c.Breaking || c.Type != "refactor" {
		t.Errorf("unexpected: %+v", c)
	}
}

func TestInvalidType(t *testing.T) {
	_, err := ParseCommit("invalid: something")
	if !errors.Is(err, ErrInvalidFormat) {
		t.Errorf("expected ErrInvalidFormat, got %v", err)
	}
}

func TestInvalidFormat(t *testing.T) {
	cases := []string{"", "no colon", "feat", "feat:", "feat: "}
	for _, msg := range cases {
		_, err := ParseCommit(msg)
		if err == nil {
			t.Errorf("expected error for %q", msg)
		}
	}
}

func TestAllTypes(t *testing.T) {
	types := []string{"feat", "fix", "docs", "style", "refactor", "perf", "test", "chore", "ci"}
	for _, typ := range types {
		c, err := ParseCommit(typ + ": description")
		if err != nil {
			t.Errorf("type %q: %v", typ, err)
		}
		if c.Type != typ {
			t.Errorf("Type = %q, want %q", c.Type, typ)
		}
	}
}
