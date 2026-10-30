package main

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"pp/lab/solution"
)

func usage() {
	fmt.Fprint(os.Stderr, "Использование: harness <команда> [аргументы]\n\nРабота №1 должна поддерживать команды:\n  info\n  counter <T> <N> <relaxed|seqcst> [--cas]\n  litmus <N> <relaxed|seqcst>\n  spsc <capacity> <items>\n  bench-counter <T> <N> <ordering>\n  bench-spsc <capacity> <items>\n")
}

func arg(rest []string, index int, what string, fallback uint64) (uint64, error) {
	if index >= len(rest) {
		return fallback, nil
	}
	value, err := strconv.ParseUint(rest[index], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("аргумент %s: ожидалось целое число, получено %q", what, rest[index])
	}
	return value, nil
}

func run(args []string) error {
	if len(args) == 0 {
		usage()
		return fmt.Errorf("не указана команда")
	}

	command, rest := args[0], args[1:]

	switch command {
	case "info":
		fmt.Println(`{"lab":1,"language":"Go","cache_line":64}`)

	case "counter":
		if len(rest) < 3 {
			usage()
			return fmt.Errorf("counter: недостаточно аргументов")
		}

		threads, err := arg(rest, 0, "threads", 1)
		if err != nil {
			return err
		}

		increments, err := arg(rest, 1, "increments", 1)
		if err != nil {
			return err
		}

		ordering := rest[2]
		if ordering != "relaxed" && ordering != "seqcst" {
			return fmt.Errorf("counter: порядок должен быть relaxed или seqcst")
		}

		useCAS := false
		if len(rest) > 3 && rest[3] == "--cas" {
			useCAS = true
		}

		var total int64
		var wg sync.WaitGroup

		for i := uint64(0); i < threads; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if useCAS {
					counter := solution.NewCASCounter()
					for j := uint64(0); j < increments; j++ {
						counter.Increment()
					}
					atomic.AddInt64(&total, counter.Get())
				} else {
					counter := solution.NewAtomicCounter()
					for j := uint64(0); j < increments; j++ {
						counter.Increment()
					}
					atomic.AddInt64(&total, counter.Get())
				}
			}()
		}

		wg.Wait()

		expected := int64(threads * increments)
		fmt.Printf(`{"total":%d,"expected":%d,"ordering":"%s","cas":%t}`+"\n",
			total, expected, ordering, useCAS)

	case "litmus":
		if len(rest) < 2 {
			usage()
			return fmt.Errorf("litmus: недостаточно аргументов")
		}

		iterations, err := arg(rest, 0, "iterations", 1)
		if err != nil {
			return err
		}

		ordering := rest[1]
		if ordering != "relaxed" && ordering != "seqcst" {
			return fmt.Errorf("litmus: порядок должен быть relaxed или seqcst")
		}

		seqcst := ordering == "seqcst"
		result := solution.Litmus(int(iterations), seqcst)

		fmt.Printf(`{"iterations":%d,"both_zero":%d,"r1_only":%d,"r2_only":%d,"both_one":%d}`+"\n",
			iterations, result.BothZero, result.R1Only, result.R2Only, result.BothOne)

	case "spsc":
		if len(rest) < 2 {
			usage()
			return fmt.Errorf("spsc: недостаточно аргументов")
		}

		capacity, err := arg(rest, 0, "capacity", 1)
		if err != nil {
			return err
		}

		items, err := arg(rest, 1, "items", 1)
		if err != nil {
			return err
		}

		q := solution.NewSPSCQueue(int(capacity))

		var wg sync.WaitGroup
		wg.Add(2)

		start := time.Now()

		go func() {
			defer wg.Done()
			for i := int64(1); i <= int64(items); i++ {
				for !q.TryPush(i) {
				}
			}
		}()

		var sum int64
		var popped int64
		var lastValue int64
		ordered := true

		go func() {
			defer wg.Done()
			for i := int64(0); i < int64(items); i++ {
				var value int64
				var ok bool
				for {
					value, ok = q.TryPop()
					if ok {
						break
					}
				}
				sum += value
				popped++
				if value <= lastValue {
					ordered = false
				}
				lastValue = value
			}
		}()

		wg.Wait()

		elapsed := time.Since(start).Seconds()
		actualCapacity := int64(q.Capacity())

		fmt.Printf(`{"popped":%d,"sum":%d,"ordered":%t,"capacity":%d,"seconds":%f}`+"\n",
			popped, sum, ordered, actualCapacity, elapsed)

	case "bench-counter":
		if len(rest) < 3 {
			usage()
			return fmt.Errorf("bench-counter: недостаточно аргументов")
		}

		threads, err := arg(rest, 0, "threads", 1)
		if err != nil {
			return err
		}

		increments, err := arg(rest, 1, "increments", 1)
		if err != nil {
			return err
		}

		ordering := rest[2]
		if ordering != "relaxed" && ordering != "seqcst" {
			return fmt.Errorf("bench-counter: порядок должен быть relaxed или seqcst")
		}

		var wg sync.WaitGroup
		start := time.Now()

		for i := uint64(0); i < threads; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				counter := solution.NewAtomicCounter()
				for j := uint64(0); j < increments; j++ {
					counter.Increment()
				}
			}()
		}

		wg.Wait()

		elapsed := time.Since(start).Seconds()
		ops := threads * increments
		opsPerSec := float64(ops) / elapsed

		fmt.Printf(`{"ops":%d,"seconds":%f,"ops_per_sec":%f}`+"\n", ops, elapsed, opsPerSec)

	case "bench-spsc":
		if len(rest) < 2 {
			usage()
			return fmt.Errorf("bench-spsc: недостаточно аргументов")
		}

		capacity, err := arg(rest, 0, "capacity", 1)
		if err != nil {
			return err
		}

		items, err := arg(rest, 1, "items", 1)
		if err != nil {
			return err
		}

		q := solution.NewSPSCQueue(int(capacity))

		var wg sync.WaitGroup
		wg.Add(2)

		start := time.Now()

		go func() {
			defer wg.Done()
			for i := int64(1); i <= int64(items); i++ {
				for !q.TryPush(i) {
				}
			}
		}()

		go func() {
			defer wg.Done()
			for i := int64(0); i < int64(items); i++ {
				for {
					_, ok := q.TryPop()
					if ok {
						break
					}
				}
			}
		}()

		wg.Wait()

		elapsed := time.Since(start).Seconds()
		itemsPerSec := float64(items) / elapsed

		fmt.Printf(`{"items":%d,"seconds":%f,"items_per_sec":%f}`+"\n", items, elapsed, itemsPerSec)

	default:
		usage()
		return fmt.Errorf("неизвестная команда: %s", command)
	}

	return nil
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "ошибка: %v\n", err)
		os.Exit(2)
	}
}
