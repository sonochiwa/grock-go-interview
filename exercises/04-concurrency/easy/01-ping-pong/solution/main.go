package ping_pong

// PingPong запускает 2 горутины, которые обмениваются сообщениями "ping" и "pong"
// через каналы. Возвращает слайс из n сообщений в порядке отправки.
func PingPong(n int) []string {
	if n <= 0 {
		return []string{}
	}

	pingCh := make(chan struct{})
	pongCh := make(chan struct{})
	result := make([]string, 0, n)
	done := make(chan struct{})

	// Горутина ping
	go func() {
		for i := 0; i < n; i += 2 {
			<-pingCh
			result = append(result, "ping")
			if i+1 < n {
				pongCh <- struct{}{}
			}
		}
		close(done)
	}()

	// Горутина pong
	go func() {
		for i := 1; i < n; i += 2 {
			<-pongCh
			result = append(result, "pong")
			if i+1 < n {
				pingCh <- struct{}{}
			}
		}
		if n == 1 {
			close(done)
		}
	}()

	// Запускаем первый ping
	pingCh <- struct{}{}
	<-done

	return result
}
