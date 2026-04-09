# Git Internals

## Object Model

```
Git = content-addressable filesystem
Каждый объект идентифицируется SHA-1 хешем его содержимого

4 типа объектов:

1. Blob — содержимое файла (без имени!)
   SHA1 = hash(content)

2. Tree — директория (список blob + tree)
   SHA1 = hash(entries)
   100644 blob a1b2c3  main.go
   100644 blob d4e5f6  go.mod
   040000 tree f7g8h9  pkg/

3. Commit — snapshot + metadata
   SHA1 = hash(tree + parent + author + message)
   tree a1b2c3
   parent d4e5f6
   author Alice <alice@mail.com> 1234567890 +0300
   committer Alice <alice@mail.com> 1234567890 +0300

   Initial commit

4. Tag (annotated) — именованный указатель на commit
   object d4e5f6
   type commit
   tag v1.0.0
   tagger Alice <alice@mail.com>

   Release v1.0.0
```

## DAG (Directed Acyclic Graph)

```
Коммиты образуют DAG (направленный ациклический граф):

A ← B ← C ← D (main)
         ↑
         └── E ← F (feature)

Каждый commit указывает на parent(s):
  - Обычный commit: 1 parent
  - Merge commit: 2+ parents
  - Initial commit: 0 parents

Branch = указатель (ref) на commit
  .git/refs/heads/main = SHA of commit D
  .git/refs/heads/feature = SHA of commit F

HEAD = указатель на текущий branch (или commit для detached)
  .git/HEAD = "ref: refs/heads/main"
```

## Хранение объектов

```
.git/objects/
├── a1/b2c3d4...    # blob
├── d4/e5f6g7...    # tree
├── f7/g8h9i0...    # commit
├── info/
└── pack/
    ├── pack-abc123.idx    # index (offsets)
    └── pack-abc123.pack   # packed objects (delta compression)

Loose objects:
  Каждый объект = отдельный zlib-сжатый файл
  .git/objects/a1/b2c3d4e5f6...
  Путь: первые 2 символа SHA → директория, остальные → файл

Pack files (git gc):
  Множество объектов в одном файле
  Delta compression: хранит diff от похожего объекта
  Значительно экономит место
  git gc / git repack — создают pack files

Проверка:
  git cat-file -t <sha>    # тип объекта
  git cat-file -p <sha>    # содержимое объекта
  git cat-file -s <sha>    # размер

  git rev-parse HEAD        # SHA текущего commit
  git log --oneline --graph # визуальный DAG
```

## Reflog

```
Reflog = журнал ВСЕХ перемещений HEAD и веток
Хранится ТОЛЬКО ЛОКАЛЬНО, 90 дней по умолчанию

git reflog
# a1b2c3d HEAD@{0}: commit: add feature
# d4e5f6g HEAD@{1}: checkout: moving from main to feature
# h7i8j9k HEAD@{2}: commit: fix bug
# l0m1n2o HEAD@{3}: reset: moving to HEAD~1

Спасение жизни:
  # "Я случайно сделал reset --hard!"
  git reflog               # найти SHA до reset
  git reset --hard HEAD@{2}  # вернуться

  # "Я удалил ветку!"
  git reflog               # найти последний commit ветки
  git checkout -b recovered HEAD@{5}

  # "Я испортил rebase!"
  git reflog
  git reset --hard HEAD@{N}  # до rebase

Reflog references:
  HEAD@{0}     — текущий HEAD
  HEAD@{1}     — предыдущая позиция HEAD
  main@{2}     — позиция main 2 действия назад
  HEAD@{1.hour.ago}  — HEAD час назад
```

## Index (Staging Area)

```
Index = .git/index

Промежуточное состояние между working dir и commit

Working Dir    →  Index (Stage)  →  Repository
              git add           git commit

Index содержит:
  - Путь файла
  - SHA blob объекта
  - File mode, timestamps
  - Stage number (для merge conflicts: 1=base, 2=ours, 3=theirs)

git ls-files --stage    # содержимое index
# 100644 a1b2c3 0 main.go     (stage 0 = normal)
# 100644 d4e5f6 1 conflict.go  (stage 1 = base)
# 100644 f7g8h9 2 conflict.go  (stage 2 = ours)
# 100644 j0k1l2 3 conflict.go  (stage 3 = theirs)
```

## Частые вопросы

**Q: Почему Git быстрый?**
A: 1) Snapshot-based, не diff-based (не нужно пересчитывать). 2) SHA-1 для дедупликации (одинаковый файл = один blob). 3) Pack files + delta compression. 4) Всё локально (нет сети для commit/log/diff).

**Q: Git хранит diff'ы или полные файлы?**
A: Полные файлы (blob = content). Но в pack files использует delta compression (diff от похожего blob). Логически — snapshot. Физически — оптимизировано.

**Q: Что такое SHA-1 collision?**
A: Два разных файла с одним SHA-1. Теоретически возможно (SHAttered attack, 2017). Git переходит на SHA-256 (с Git 2.42+, opt-in). На практике: не проблема для большинства репозиториев.
