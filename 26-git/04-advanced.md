# Advanced Git

## git bisect

```
Бинарный поиск коммита, который сломал что-то

git bisect start
git bisect bad                    # текущий = сломан
git bisect good v1.0.0            # v1.0.0 = работал

# Git checkout'ит середину → ты проверяешь
git bisect good    # этот коммит OK
git bisect bad     # этот коммит сломан
# ... повторяй до нахождения виновника

git bisect reset   # вернуться к исходному состоянию

# Автоматический bisect (с тестом):
git bisect start HEAD v1.0.0
git bisect run go test ./pkg/auth/...
# Git автоматически найдёт первый failing commit!

# 1000 коммитов → ~10 шагов (log2(1000) ≈ 10)
```

## git worktree

```
Несколько рабочих директорий из одного репозитория

git worktree add ../hotfix-branch hotfix/urgent
# Создаёт ../hotfix-branch с checkout hotfix/urgent

# Теперь можно работать в двух ветках одновременно
# без stash/commit текущей работы!

cd ../hotfix-branch
# ... fix bug ...
git commit -m "fix: urgent production issue"
git push

cd ../main-repo
# ... продолжаем работу без потерь

git worktree list     # все worktrees
git worktree remove ../hotfix-branch   # удалить

Когда полезно:
  ✅ Hotfix пока работаешь над feature
  ✅ Code review: checkout PR в отдельную директорию
  ✅ Параллельная работа над разными ветками
  ✅ CI: build разных веток одновременно
```

## git stash

```bash
git stash                 # сохранить uncommitted изменения
git stash push -m "WIP: auth feature"   # с сообщением
git stash list            # все stash'и
git stash pop             # применить и удалить последний
git stash apply stash@{2} # применить конкретный (не удалять)
git stash drop stash@{0}  # удалить
git stash show -p         # показать diff

# Stash только tracked файлов (default)
git stash push -u         # + untracked files
git stash push -a         # + ignored files

# Stash конкретных файлов
git stash push -m "config changes" -- config.yaml

# Создать ветку из stash
git stash branch feature-from-stash stash@{0}
```

## Interactive Rebase

```bash
git rebase -i HEAD~5    # последние 5 коммитов

# Откроется редактор:
pick a1b2c3d Add user model
pick d4e5f6g Add user repository
pick h7i8j9k WIP: user handler         ← squash
pick l0m1n2o Fix typo in handler        ← fixup (squash без message)
pick p3q4r5s Add user tests

# Команды:
# pick   — оставить как есть
# reword — изменить commit message
# edit   — остановиться для amend
# squash — слить с предыдущим (объединить messages)
# fixup  — слить с предыдущим (выбросить message)
# drop   — удалить коммит
# reorder — перетащить строку (изменить порядок)

# Результат: 3 чистых коммита вместо 5

# fixup shortcut (при создании):
git commit --fixup=a1b2c3d      # создать fixup! коммит
git rebase -i --autosquash HEAD~5  # auto-squash fixup! коммитов
```

## git rerere

```
rerere = "reuse recorded resolution"
Запоминает как ты разрешил конфликт и применяет автоматически

git config rerere.enabled true

Сценарий:
  1. Merge → конфликт → разрешаешь
  2. git rerere записывает разрешение
  3. Rebase → тот же конфликт → автоматически разрешается!

Полезно при:
  - Частый rebase на main
  - Long-running feature branches
  - Повторяющиеся конфликты

git rerere status     # текущие записанные конфликты
git rerere diff       # показать как было разрешено
```

## Submodules

```bash
# Репозиторий внутри репозитория
git submodule add https://github.com/org/lib.git vendor/lib
git commit -m "chore: add lib submodule"

# Clone с submodules
git clone --recurse-submodules https://github.com/org/project.git

# Обновить submodule
cd vendor/lib
git pull origin main
cd ../..
git add vendor/lib
git commit -m "chore: update lib to latest"

# Обновить все submodules
git submodule update --remote --merge

Проблемы:
  ❌ Забывают init/update после clone
  ❌ Сложный workflow
  ❌ Привязка к конкретному commit

Альтернативы:
  Go: go modules (встроенный dependency management)
  Monorepo: всё в одном репозитории
```

## Частые вопросы

**Q: Как отменить последний commit (не push-нутый)?**
A: `git reset --soft HEAD~1` — убрать commit, оставить изменения staged. `git reset --mixed HEAD~1` — убрать commit, изменения unstaged. `git reset --hard HEAD~1` — убрать commit И изменения.

**Q: Как изменить старый commit message?**
A: `git rebase -i HEAD~N` → reword. Для последнего: `git commit --amend`.

**Q: Как найти кто написал эту строку?**
A: `git blame file.go` — автор каждой строки. `git log -p -S "functionName"` — когда добавлена/удалена строка.
