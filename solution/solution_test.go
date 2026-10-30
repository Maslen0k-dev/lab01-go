package solution

import (
	"os"
	"strings"
	"sync"
	"testing"
)

func isRaceDetectorEnabled() bool {
	for _, arg := range os.Args {
		if strings.Contains(arg, "-test.race") || strings.Contains(arg, "-race") {
			return true
		}
	}
	return false
}

func TestAtomicCounter(t *testing.T) {
	counter := NewAtomicCounter()
	var wg sync.WaitGroup

	numGoroutines := 8
	incrementsPerGoroutine := 1000

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				counter.Increment()
			}
		}()
	}

	wg.Wait()

	expected := int64(numGoroutines * incrementsPerGoroutine)
	actual := counter.Get()

	if actual != expected {
		t.Errorf("Expected %d, got %d", expected, actual)
	}
}

func TestCASCounter(t *testing.T) {
	counter := NewCASCounter()
	var wg sync.WaitGroup

	numGoroutines := 8
	incrementsPerGoroutine := 1000

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				counter.Increment()
			}
		}()
	}

	wg.Wait()

	expected := int64(numGoroutines * incrementsPerGoroutine)
	actual := counter.Get()

	if actual != expected {
		t.Errorf("Expected %d, got %d", expected, actual)
	}
}

func TestLitmus(t *testing.T) {
	n := 10000

	t.Run("seqcst", func(t *testing.T) {
		result := Litmus(n, true)
		if result.BothZero != 0 {
			t.Errorf("seqcst: both_zero должен быть 0, получен %d", result.BothZero)
		}
		total := result.BothZero + result.R1Only + result.R2Only + result.BothOne
		if total != int64(n) {
			t.Errorf("seqcst: сумма исходов %d != %d", total, n)
		}
	})

	t.Run("relaxed", func(t *testing.T) {
		if isRaceDetectorEnabled() {
			t.Skip("relaxed режим содержит намеренную гонку данных, пропускаем под -race")
		}
		result := Litmus(n, false)
		total := result.BothZero + result.R1Only + result.R2Only + result.BothOne
		if total != int64(n) {
			t.Errorf("relaxed: сумма исходов %d != %d", total, n)
		}
		if result.BothZero == 0 {
			t.Errorf("relaxed: both_zero должен быть > 0")
		}
		t.Logf("relaxed: both_zero=%d, r1_only=%d, r2_only=%d, both_one=%d",
			result.BothZero, result.R1Only, result.R2Only, result.BothOne)
	})
}

func TestSPSC(t *testing.T) {
	capacity := 8
	items := 1000
	q := NewSPSCQueue(capacity)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := int64(1); i <= int64(items); i++ {
			for !q.TryPush(i) {
			}
		}
	}()

	var sum int64
	var lastValue int64
	ordered := true

	go func() {
		defer wg.Done()
		for i := 0; i < items; i++ {
			var value int64
			var ok bool
			for {
				value, ok = q.TryPop()
				if ok {
					break
				}
			}
			sum += value
			if value <= lastValue {
				ordered = false
			}
			lastValue = value
		}
	}()

	wg.Wait()

	expectedSum := int64(items * (items + 1) / 2)
	if sum != expectedSum {
		t.Errorf("sum = %d, expected %d", sum, expectedSum)
	}
	if !ordered {
		t.Error("order violated")
	}
}
