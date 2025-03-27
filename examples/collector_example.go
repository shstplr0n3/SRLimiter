func main() {
	collector := NewCollector()

	// multi-thread addition
	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(weight uint16) {
			defer wg.Done()
			collector.Add(&Load{priorityWeight: weight})
		}(uint16(i % 100))
	}

	wg.Wait()

	// multi-thread popping
	for collector.length > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if load := collector.Pop(); load != nil {
				// ...
			}
		}()
	}

	wg.Wait()
}
