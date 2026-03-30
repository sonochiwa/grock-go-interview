# Внутреннее устройство каналов

## Структура

```go
// runtime/chan.go
type hchan struct {
    qcount   uint           // текущее кол-во элементов в буфере
    dataqsiz uint           // размер буфера (0 для unbuffered)
    buf      unsafe.Pointer // ring buffer (циклический буфер)
    elemsize uint16         // размер элемента
    closed   uint32         // 1 если закрыт
    elemtype *_type         // тип элемента (для GC)
    sendx    uint           // индекс записи в ring buffer
    recvx    uint           // индекс чтения из ring buffer
    recvq    waitq          // очередь ожидающих получателей (sudog)
    sendq    waitq          // очередь ожидающих отправителей (sudog)
    lock     mutex          // внутренний мьютекс
}
```

```
Ring buffer (buffered channel, size=4):
         recvx        sendx
           ▼            ▼
┌────┬────┬────┬────┐
│ v0 │ v1 │    │    │
└────┴────┴────┴────┘
  qcount = 2
```

## Операции

### Send (ch <- v)

1. Lock hchan
2. Если есть ожидающий receiver в recvq → **копировать данные напрямую** в стек receiver, разбудить его (fast path)
3. Если есть место в buf → скопировать в buf[sendx], sendx++
4. Иначе → создать sudog, добавить в sendq, усыпить горутину

### Receive (v = <-ch)

1. Lock hchan
2. Если есть ожидающий sender в sendq → **копировать данные напрямую** из sender (fast path)
3. Если buf не пуст → взять из buf[recvx], recvx++
4. Иначе → создать sudog, добавить в recvq, усыпить горутину

### Close

1. Lock hchan
2. Установить closed = 1
3. Разбудить ВСЕХ ожидающих receivers (отдать zero value)
4. Разбудить ВСЕХ ожидающих senders (они запаникуют)

## Оптимизация: прямая копия

Для unbuffered каналов данные копируются **напрямую между стеками горутин**, минуя буфер. Это экономит одну операцию копирования.

## Частые вопросы на собеседованиях

**Q: Как канал блокирует горутину?**
A: Горутина создаёт sudog (описатель ожидания), добавляет в waitq канала, и отдаёт свой P scheduler'у (gopark).

**Q: Чем unbuffered канал отличается от buffered(1) под капотом?**
A: Unbuffered: нет ring buffer, данные копируются напрямую sender→receiver. Buffered(1): есть ring buffer на 1 элемент.

**Q: Есть ли lock-free каналы?**
A: Нет. hchan использует внутренний mutex. Были попытки lock-free, но не приняты из-за сложности.
