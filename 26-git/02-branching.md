# Branching

## Merge vs Rebase

```
Merge (git merge feature):
  Создаёт merge commit с двумя parents

  A ← B ← C ← D ← M (main)
           ↑       ↗
           └── E ← F (feature)

  ✅ Сохраняет полную историю
  ✅ Non-destructive (безопасно)
  ❌ "Шумная" история (merge commits)

Rebase (git rebase main):
  Переписывает коммиты поверх target branch

  До:  A ← B ← C (main)
            ↑
            └── D ← E (feature)

  После: A ← B ← C (main) ← D' ← E' (feature)

  ✅ Линейная чистая история
  ✅ Нет merge commits
  ❌ Перезаписывает SHA (новые коммиты D', E')
  ❌ ОПАСНО для shared branches (rewrite public history!)

Золотое правило:
  НИКОГДА не rebase то, что уже push-нуто и используется другими!
  Rebase = только для ЛОКАЛЬНЫХ веток
```

## Fast-Forward

```
Ситуация: main не изменилась после создания feature

  A ← B (main) ← C ← D (feature)

git merge feature:
  Fast-forward: main просто передвигается на D
  Нет merge commit

  A ← B ← C ← D (main, feature)

git merge --no-ff feature:
  Принудительный merge commit (для видимости feature в истории)

  A ← B ← ─── M (main)
       ↑       ↗
       └── C ← D (feature)
```

## Squash

```
git merge --squash feature:
  Все коммиты feature → один commit в main
  Не создаёт merge commit, feature branch не linked

  Feature:  D ← E ← F (3 коммита)
  Main:     A ← B ← C ← G (1 squashed commit)

  G содержит все изменения D+E+F но как один коммит

Когда:
  ✅ Feature branch с "wip", "fix typo" — squash в чистый commit
  ❌ Long-running branch (теряется история)

Interactive rebase squash:
  git rebase -i HEAD~3
  # pick   abc1234 Add feature
  # squash def5678 Fix typo
  # squash ghi9012 Address review
  →  Один коммит с объединённым message
```

## Cherry-Pick

```
git cherry-pick <commit>:
  Скопировать один commit из другой ветки

  main:     A ← B ← C
  feature:  A ← B ← D ← E ← F

  git checkout main
  git cherry-pick E
  →  main: A ← B ← C ← E' (копия E)

  Новый SHA (E' ≠ E), но те же изменения

Когда:
  ✅ Hotfix: cherry-pick fix из develop в release
  ✅ Backport: cherry-pick feature в старую версию
  ❌ Много cherry-picks → лучше merge/rebase
```

## Merge Strategies

```
git merge -s <strategy>:

recursive (default, до Git 2.34):
  3-way merge: base + ours + theirs
  Автоматически разрешает простые конфликты

ort (default, Git 2.34+):
  "Ostensibly Recursive's Twin"
  Замена recursive, быстрее и корректнее
  Лучше обрабатывает rename detection

ours:
  git merge -s ours feature
  Результат = наша версия (игнорирует feature полностью)
  Merge commit создаётся (для истории)

octopus:
  Merge нескольких веток одновременно
  git merge feature1 feature2 feature3
  Не работает при конфликтах

Merge options (не путать со strategies):
  -X ours    — при конфликте брать нашу версию
  -X theirs  — при конфликте брать их версию
  -X patience — patience diff algorithm (лучше для рефакторинга)
```

## Conflict Resolution

```
Конфликт:
  <<<<<<< HEAD
  our code here
  =======
  their code here
  >>>>>>> feature

Алгоритм разрешения:
  1. git status — какие файлы в конфликте
  2. Открыть файл, найти маркеры <<<<<<<
  3. Понять оба изменения (git log --merge -p)
  4. Выбрать правильный вариант / комбинировать
  5. Удалить маркеры
  6. git add resolved_file.go
  7. git commit (или git rebase --continue)

Инструменты:
  git mergetool             # визуальный мерж (vimdiff, vscode)
  git diff --check          # найти оставшиеся маркеры
  git log --merge           # коммиты вызвавшие конфликт
  git diff :1:file :2:file  # base vs ours
  git diff :1:file :3:file  # base vs theirs
```

## Частые вопросы

**Q: merge vs rebase — когда что?**
A: Rebase — для СВОИХ локальных веток перед merge в main (линейная история). Merge — для shared branches и pull requests. Многие команды: squash merge для PRs.

**Q: Что делать если rebase пошёл не так?**
A: `git rebase --abort` — отменить и вернуться. Или `git reflog` → `git reset --hard HEAD@{N}` — откатить к состоянию до rebase.

**Q: Можно ли отменить merge?**
A: `git revert -m 1 <merge-commit>` — создать "anti-commit". НЕ `git reset` если уже push-нуто. Внимание: после revert merge, повторный merge той же ветки не подхватит уже reverted изменения — нужен `git revert <revert-commit>`.
