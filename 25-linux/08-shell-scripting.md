# Shell Scripting

## Bash Essentials

```bash
#!/bin/bash
set -euo pipefail    # ОБЯЗАТЕЛЬНО в production скриптах!
# -e: exit on error
# -u: error on undefined variable
# -o pipefail: pipe fails if ANY command fails

# Переменные
NAME="world"
echo "Hello, ${NAME}"
echo "PID: $$"          # текущий PID
echo "Args: $@"         # все аргументы
echo "Count: $#"        # кол-во аргументов
echo "Exit code: $?"    # код предыдущей команды

# Условия
if [[ -f "file.txt" ]]; then
    echo "file exists"
elif [[ -d "dir" ]]; then
    echo "directory exists"
fi

# Проверки:
# -f file exists (regular)
# -d directory exists
# -z string is empty
# -n string is not empty
# -eq, -ne, -lt, -gt (числа)
# ==, != (строки)

# Циклы
for f in *.go; do
    echo "Processing $f"
done

for i in {1..10}; do
    echo "$i"
done

while read -r line; do
    echo "$line"
done < input.txt

# Функции
deploy() {
    local version=$1    # local переменная
    echo "Deploying $version"
    return 0
}
deploy "v1.2.3"
```

## grep / awk / sed

```bash
# grep — поиск текста
grep "error" app.log                  # строки с "error"
grep -i "error" app.log               # case-insensitive
grep -r "TODO" ./src/                  # рекурсивно
grep -n "func main" *.go              # с номерами строк
grep -c "ERROR" app.log               # подсчёт совпадений
grep -v "DEBUG" app.log               # инвертировать (без DEBUG)
grep -E "error|warn|fatal" app.log    # regex (extended)
grep -A 3 "panic" app.log             # + 3 строки после
grep -B 2 "panic" app.log             # + 2 строки до
grep -l "TODO" *.go                   # только имена файлов

# awk — обработка табличных данных
awk '{print $1, $3}' file.txt         # 1-й и 3-й столбцы
awk -F: '{print $1}' /etc/passwd      # delimiter :
awk '$3 > 100 {print $0}' data.txt    # фильтр по значению
awk '{sum += $1} END {print sum}'      # сумма 1-го столбца
awk 'NR > 1 {print}'                   # пропустить header

# Примеры:
# Топ-10 IP по количеству запросов
awk '{print $1}' access.log | sort | uniq -c | sort -rn | head -10

# Среднее время ответа
awk '{sum+=$NF; n++} END {print sum/n}' access.log

# sed — stream editor
sed 's/old/new/' file.txt              # заменить первое
sed 's/old/new/g' file.txt             # заменить все
sed -i 's/old/new/g' file.txt          # in-place
sed '5d' file.txt                      # удалить 5-ю строку
sed '/^#/d' file.txt                   # удалить комментарии
sed -n '10,20p' file.txt               # строки 10-20
```

## Pipelines и Composition

```bash
# | (pipe): stdout → stdin
# Каждая команда = отдельный процесс

# Топ-5 самых больших файлов
find . -type f -exec du -h {} + | sort -rh | head -5

# Количество горутин в Go процессе
curl -s localhost:6060/debug/pprof/goroutine?debug=1 | head -1

# JSON processing с jq
curl -s api.example.com/users | jq '.[] | {name, email}'
curl -s api.example.com/users | jq 'length'   # подсчёт
curl -s api.example.com/users | jq '.[0].name' # первый user

# Параллельное выполнение
cat urls.txt | xargs -P 10 -I {} curl -s {}    # 10 параллельных curl

# Process substitution
diff <(ssh server1 cat /etc/config) <(ssh server2 cat /etc/config)
```

## Job Control

```bash
# Background / Foreground
./long-task &       # запустить в background
jobs                # список background jobs
fg %1               # вернуть job 1 в foreground
bg %1               # продолжить job 1 в background
kill %1             # убить job 1

# Ctrl+Z → suspend → bg → background
# Ctrl+C → SIGINT → terminate

# nohup: продолжить после закрытия терминала
nohup ./server > server.log 2>&1 &

# disown: отвязать от терминала
./server &
disown

# screen / tmux: persistent terminal sessions
tmux new -s deploy
# ... работаем ...
# Ctrl+B, D → detach
tmux attach -t deploy  # вернуться
```

## Полезные one-liners

```bash
# Мониторинг файла
tail -f app.log | grep --line-buffered "ERROR"

# Найти и убить процесс
pkill -f "myapp"
pgrep -af "myapp"

# Размер директорий
du -sh */ | sort -rh

# Свободное место
df -h

# Кто слушает порт
ss -tlnp | grep :8080

# Watch: повторять команду
watch -n 1 'ss -s'        # каждую секунду

# Скопировать с прогрессом
rsync -avP source/ dest/

# HTTP сервер из текущей директории
python3 -m http.server 8080

# Генерация пароля
openssl rand -base64 32
```
