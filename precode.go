package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// Generator генерирует последовательность чисел 1,2,3 и т.д. и
// отправляет их в канал ch. При этом после записи в канал для каждого числа
// вызывается функция fn. Она служит для подсчёта количества и суммы
// сгенерированных чисел.
func Generator(ctx context.Context, ch chan<- int64, fn func(int64)) {
	// 1. Функция Generator
	var count atomic.Int64
	count.Store(1)
	for {
		select {
		case <-ctx.Done():
			close(ch)
			return
		default:
			count.Add(1)
			ch <- count.Load()
			fn(count.Load())
		}
	}
}

// Worker читает число из канала in и пишет его в канал out.
func Worker(in <-chan int64, out chan<- int64) {
	// 2. Функция Worker
	for {
		v, ok := <-in
		if !ok {
			//Закрыть канал и выйти
			close(out)
			return
		}
		//Пишем в канал out
		out <- v
		//пауза на 1 млсек
		time.Sleep(1 * time.Millisecond)
	}
}

func main() {
	chIn := make(chan int64)

	// 3. Создание контекста
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	// через 1 секунды вызываем cancel для отмены операции
	defer cancel()

	// для проверки будем считать количество и сумму отправленных чисел
	var inputSum atomic.Int64   // сумма сгенерированных чисел
	var inputCount atomic.Int64 // количество сгенерированных чисел

	// генерируем числа, считая параллельно их количество и сумму
	go Generator(ctx, chIn, func(i int64) {
		inputSum.Add(i)
		inputCount.Add(1)
	})

	const NumOut = 5 // количество обрабатывающих горутин и каналов
	// outs — слайс каналов, куда будут записываться числа из chIn
	outs := make([]chan int64, NumOut)
	for i := 0; i < NumOut; i++ {
		// создаём каналы и для каждого из них вызываем горутину Worker
		outs[i] = make(chan int64)
		go Worker(chIn, outs[i])
	}

	// amounts — слайс, в который собирается статистика по горутинам
	amounts := make([]int64, NumOut)
	// chOut — канал, в который будут отправляться числа из горутин `outs[i]`
	chOut := make(chan int64, NumOut)

	var wg sync.WaitGroup

	// 4. Собираем числа из каналов outs
	var i int64
	for i = 0; i < int64(len(outs)); i++ {
		wg.Add(1)
		go func(in <-chan int64, i int64) {
			defer wg.Done()
			amounts[i]++
		}(outs[i], i)
	}

	// var i int64
	// i = -1
	// for {
	// 	wg.Add(1)
	// 	i++
	// 	go func(in <-chan int64, i int64) {
	// 		defer wg.Done()
	// 		//amounts[i]++
	// 		v := <-in
	// 		chOut <- v
	// 		// ждём завершения работы всех горутин для outs
	// 		wg.Wait()
	// 		// закрываем результирующий канал
	// 		close(chOut)
	// 	}(outs[i], i)
	// }

	//Переделать эту функцию на atomic

	var count int64 // количество чисел результирующего канала
	var sum int64   // сумма чисел результирующего канала

	// 5. Читаем числа из результирующего канала

	for v := range chOut {
		count++
		sum = +v
	}

	fmt.Println("Количество чисел", inputCount.Load(), count)
	fmt.Println("Сумма чисел", inputSum.Load(), sum)
	fmt.Println("Разбивка по каналам", amounts)

	// проверка результатов
	if inputSum.Load() != sum {
		log.Fatalf("Ошибка: суммы чисел не равны: %d != %d\n", inputSum.Load(), sum)
	}
	if inputCount.Load() != count {
		log.Fatalf("Ошибка: количество чисел не равно: %d != %d\n", inputCount.Load(), count)
	}
	for _, v := range amounts {
		inputCount.Add(-1 * v)
	}
	if inputCount.Load() != 0 {
		log.Fatalf("Ошибка: разделение чисел по каналам неверное\n")
	}
}
