# Commit Message Parser

Реализуй парсер conventional commits:

- `CommitInfo{Type, Scope, Description string, Breaking bool}`
- `ParseCommit(msg string) (*CommitInfo, error)` — распарсить commit message

Формат: `<type>(<scope>): <description>` или `<type>: <description>`
Breaking: `!` перед `:` (например `feat(api)!: remove v1`)

Валидные types: feat, fix, docs, style, refactor, perf, test, chore, ci
