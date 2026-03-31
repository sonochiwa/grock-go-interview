# Git Troubleshooting

## Reset vs Revert vs Restore

```
git reset — передвинуть HEAD (изменить историю)
  --soft HEAD~1   → убрать commit, изменения staged
  --mixed HEAD~1  → убрать commit, изменения unstaged (default)
  --hard HEAD~1   → убрать commit И изменения (ОПАСНО!)

  ⚠️ Только для НЕ push-нутых коммитов!

git revert — создать "анти-commit" (безопасно)
  git revert <commit>  → новый commit, отменяющий изменения
  git revert -m 1 <merge-commit>  → revert merge commit

  ✅ Для push-нутых коммитов (не меняет историю)

git restore — восстановить файлы (Git 2.23+)
  git restore file.go              → откатить рабочую копию
  git restore --staged file.go     → unstage файл
  git restore --source=HEAD~3 file.go  → версия из commit
```

## Recover Lost Commits

```bash
# "Я случайно удалил коммиты / ветку / сделал reset --hard"

# 1. Reflog — журнал всех перемещений HEAD
git reflog
# a1b2c3d HEAD@{0}: reset: moving to HEAD~3
# d4e5f6g HEAD@{1}: commit: important feature   ← вот он!
# h7i8j9k HEAD@{2}: commit: another commit

# 2. Восстановить
git reset --hard d4e5f6g          # вернуть HEAD
# или
git checkout -b recovered d4e5f6g  # создать ветку

# "Я удалил ветку"
git reflog | grep "feature/"
git checkout -b feature/recovered <sha>

# "Я сделал force push и перезаписал remote"
# Если кто-то ещё имеет старую версию:
git push origin recovered-branch:main --force-with-lease

# Dangling commits (не в reflog):
git fsck --unreachable --no-reflogs
# unreachable commit a1b2c3d...
git show a1b2c3d
```

## Force Push Safety

```bash
# ❌ НИКОГДА:
git push --force origin main    # перезапишет чужую работу!

# ✅ Вместо:
git push --force-with-lease origin feature
# Проверяет что remote не изменился с момента последнего fetch
# Если кто-то push-нул после тебя → отказ → fetch → resolve

# Защита на уровне GitHub/GitLab:
# Settings → Branches → Branch protection rules:
#   ✅ Require pull request reviews
#   ✅ Require status checks (CI)
#   ✅ Do not allow force pushes
#   ✅ Require linear history (no merge commits)
```

## Типичные проблемы

```bash
# "Detached HEAD"
# HEAD указывает на commit, а не на branch
git checkout -b new-branch   # создать ветку и продолжить работу
# или
git switch -             # вернуться на предыдущую ветку

# "Changes would be overwritten"
# Есть uncommitted changes, которые конфликтуют
git stash                # сохранить изменения
git checkout main
git stash pop            # вернуть изменения

# "Merge conflict в binary файле"
git checkout --ours file.png     # взять нашу версию
git checkout --theirs file.png   # взять их версию
git add file.png

# Большой файл случайно закоммичен
# (уже push-нут → нужен BFG или git-filter-repo)
git filter-repo --strip-blobs-bigger-than 10M
# Или BFG Repo-Cleaner:
bfg --strip-blobs-bigger-than 10M
git push --force

# "Я коммитил в wrong branch"
git stash                    # если есть uncommitted
git log --oneline -3          # найти коммиты
git checkout correct-branch
git cherry-pick <sha1> <sha2> # перенести коммиты
git checkout wrong-branch
git reset --hard HEAD~2       # убрать оттуда

# ".gitignore не работает" (файл уже tracked)
git rm --cached file.log     # убрать из tracking (файл останется)
echo "*.log" >> .gitignore
git commit -m "chore: ignore log files"
```

## Полезные команды

```bash
# Кто последний менял строку
git blame -L 10,20 main.go

# Найти commit по содержимому
git log -S "functionName" --oneline       # добавление/удаление строки
git log -G "regex" --oneline              # regex поиск
git log --all --oneline -- path/to/file   # история файла

# Diff
git diff main..feature          # разница между ветками
git diff --stat main..feature   # только имена файлов
git diff --name-only HEAD~5     # файлы за 5 коммитов

# Очистка
git clean -fd          # удалить untracked files и dirs
git clean -fxd         # + ignored файлы (осторожно!)
git gc --aggressive    # garbage collection + pack

# Размер репозитория
git count-objects -vH
# size-pack: 125.00 MiB
```
