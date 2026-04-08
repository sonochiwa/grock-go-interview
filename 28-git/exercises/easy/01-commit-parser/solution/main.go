package commit_parser

import (
	"errors"
	"strings"
)

var ErrInvalidFormat = errors.New("invalid commit message format")

type CommitInfo struct {
	Type        string
	Scope       string
	Description string
	Breaking    bool
}

var validTypes = map[string]bool{
	"feat": true, "fix": true, "docs": true, "style": true,
	"refactor": true, "perf": true, "test": true, "chore": true, "ci": true,
}

func ParseCommit(msg string) (*CommitInfo, error) {
	colonIdx := strings.Index(msg, ": ")
	if colonIdx < 0 {
		return nil, ErrInvalidFormat
	}

	prefix := msg[:colonIdx]
	desc := strings.TrimSpace(msg[colonIdx+2:])
	if desc == "" {
		return nil, ErrInvalidFormat
	}

	info := &CommitInfo{Description: desc}

	// Check breaking
	if strings.HasSuffix(prefix, "!") {
		info.Breaking = true
		prefix = prefix[:len(prefix)-1]
	}

	// Check scope
	if parenOpen := strings.Index(prefix, "("); parenOpen >= 0 {
		parenClose := strings.Index(prefix, ")")
		if parenClose < 0 || parenClose < parenOpen {
			return nil, ErrInvalidFormat
		}
		info.Type = prefix[:parenOpen]
		info.Scope = prefix[parenOpen+1 : parenClose]
	} else {
		info.Type = prefix
	}

	if !validTypes[info.Type] {
		return nil, ErrInvalidFormat
	}

	return info, nil
}
