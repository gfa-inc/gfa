package syncx

import "sync"

type SyncFunc func()

func All(funcs ...SyncFunc) {
	var wg sync.WaitGroup

	for _, fn := range funcs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fn()
		}()
	}

	wg.Wait()
}
