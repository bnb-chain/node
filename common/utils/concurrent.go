package utils

import "sync"

func ConcurrentExecuteAsync(concurrency int, producer func(), consumer func(), postConsume func()) {
	var wg sync.WaitGroup
	wg.Add(concurrency)
	consumerWrapper := func() {
		defer wg.Done()
		consumer()
	}

	for i := 0; i < concurrency; i++ {
		go consumerWrapper()
	}

	go func() {
		producer()
		wg.Wait()
		postConsume()
	}()
}

func ConcurrentExecuteSync(concurrency int, producer func(), consumer func()) {
	var wg sync.WaitGroup
	wg.Add(concurrency)
	consumerWrapper := func() {
		defer wg.Done()
		consumer()
	}

	for i := 0; i < concurrency; i++ {
		go consumerWrapper()
	}

	producer()
	wg.Wait()
}
