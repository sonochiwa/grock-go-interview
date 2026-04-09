# Filesystem

## Inode

```
Inode = metadata файла (НЕ имя, НЕ данные)

Содержит:
  - File type (regular, directory, symlink, socket, pipe)
  - Permissions (rwxrwxrwx)
  - Owner (UID, GID)
  - Size
  - Timestamps: atime, mtime, ctime
  - Hard link count
  - Pointers to data blocks (direct, indirect, double indirect)

  НЕ содержит: имя файла! Имя → inode mapping хранится в directory.

Проверка:
  ls -i file.txt        # показать inode number
  stat file.txt         # полная info
  df -i                 # использование inodes (можно исчерпать!)

Проблема: "No space left on device" при свободном месте
  → Закончились inodes (много мелких файлов)
  → df -i → IUse% = 100%
```

## Hard Links vs Soft Links

```
Hard Link:
  Ещё одно имя для того же inode
  ln file.txt file_link.txt
  - Оба имени равноправны (нет "оригинала")
  - Удаление одного не удаляет данные (пока link count > 0)
  - Нельзя на директорию (кроме . и ..)
  - Нельзя на другую файловую систему
  - stat: Links: 2

  file.txt ─────→ [inode 12345] → data blocks
  file_link.txt ─→ [inode 12345] → data blocks

Soft Link (Symbolic):
  Указатель на путь (как ярлык)
  ln -s /path/to/file.txt link.txt
  - Отдельный inode с типом symlink
  - Данные symlink = путь к target
  - Может на директорию
  - Может на другую файловую систему
  - Может быть "битым" (target удалён)

  link.txt [inode 99999: "/path/to/file.txt"]
           → file.txt [inode 12345] → data blocks
```

## Permissions

```
rwxrwxrwx = owner | group | others
  r (4) = read
  w (2) = write
  x (1) = execute (для dir: доступ в директорию)

Числовой формат:
  chmod 755 file  = rwxr-xr-x
  chmod 644 file  = rw-r--r--
  chmod 600 file  = rw------- (secrets!)

Special bits:
  SUID (4xxx): выполнить от имени owner
    chmod 4755 /usr/bin/passwd → запускается как root
  SGID (2xxx): выполнить от имени group (для dir: наследовать group)
  Sticky (1xxx): в /tmp — удалять только свои файлы
    chmod 1777 /tmp

umask:
  umask 022 → новые файлы: 644 (666-022), директории: 755 (777-022)
  umask 077 → новые файлы: 600, директории: 700
```

## Ключевые директории

```
/proc — виртуальная FS, информация о ядре и процессах
  /proc/cpuinfo          — CPU info
  /proc/meminfo          — Memory info
  /proc/loadavg          — Load average
  /proc/<pid>/status     — Статус процесса
  /proc/<pid>/fd/        — File descriptors
  /proc/<pid>/maps       — Memory mappings
  /proc/<pid>/limits     — Ресурсные лимиты
  /proc/sys/             — Kernel parameters (sysctl)

/dev — устройства
  /dev/null   — чёрная дыра
  /dev/zero   — источник нулей
  /dev/random — криптографический random (blocking)
  /dev/urandom — криптографический random (non-blocking)
  /dev/sda    — диск
  /dev/pts/   — pseudo-terminals

/sys — sysfs (hardware info, kernel objects)
/var/log — логи (syslog, auth.log, kern.log)
/etc — конфигурация
/tmp — временные файлы (очищается при reboot)
```

## Файловые системы

```
ext4:
  Default в большинстве Linux
  Journaling (write-ahead log)
  Max file: 16 TB, Max FS: 1 EB
  Extents (continuous blocks)

XFS:
  Оптимизирован для больших файлов
  Параллельный allocation (хорош для многопоточности)
  Default в RHEL/CentOS

tmpfs:
  Файловая система в RAM
  mount -t tmpfs -o size=1G tmpfs /mnt/ramdisk
  Быстро, но теряется при reboot

overlayfs:
  Слоёная FS (используется Docker)
  Lower layer (read-only) + Upper layer (read-write)
  Copy-on-Write: запись → копия в upper layer

Монтирование:
  mount /dev/sdb1 /mnt/data           # монтировать
  mount -o ro /dev/sdb1 /mnt/data     # read-only
  umount /mnt/data                     # размонтировать
  findmnt                              # все mount points
  lsblk                                # block devices
```

## lsof

```bash
lsof -p <pid>           # все открытые файлы процесса
lsof -i :8080           # кто слушает порт 8080
lsof -i tcp             # все TCP соединения
lsof +D /var/log        # кто использует файлы в директории
lsof /var/log/syslog    # кто открыл этот файл
lsof -u myuser          # файлы пользователя

# Удалённый файл всё ещё занимает место?
lsof | grep deleted     # процесс держит deleted file → fd → место не освобождается
# Решение: restart процесса или truncate через fd:
# : > /proc/<pid>/fd/<fd_number>
```
