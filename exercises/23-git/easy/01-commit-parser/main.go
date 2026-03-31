package commit_parser

import "errors"

var ErrInvalidFormat = errors.New("invalid commit message format")

type CommitInfo struct {
	Type        string
	Scope       string
	Description string
	Breaking    bool
}

// TODO: распарси conventional commit message
// "feat(auth): add login" → {Type:"feat", Scope:"auth", Description:"add login"}
// "fix: typo" → {Type:"fix", Scope:"", Description:"typo"}
// "feat!: breaking" → {Type:"feat", Breaking:true, Description:"breaking"}
func ParseCommit(msg string) (*CommitInfo, error) {
	return nil, ErrInvalidFormat
}
