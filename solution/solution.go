package solution

import (
	"math/bits"
	"math/rand"
	"sync"
	"sync/atomic"
	"unsafe"
)

// Часть A: Атомарный счётчик

type AtomicCounter struct {
	value int64
}

func NewAtomicCounter() *AtomicCounter {
	return &AtomicCounter{value: 0}
}

func (c *AtomicCounter) Increment() {
	atomic.AddInt64(&c.value, 1)
}

func (c *AtomicCounter) Get() int64 {
	return atomic.LoadInt64(&c.value)
}

type CASCounter struct {
	value int64
}

func NewCASCounter() *CASCounter {
	return &CASCounter{value: 0}
}

func (c *CASCounter) Increment() {
	for {
		old := atomic.LoadInt64(&c.value)
		new := old + 1
		if atomic.CompareAndSwapInt64(&c.value, old, new) {
			return
		}
	}
}

func (c *CASCounter) Get() int64 {
	return atomic.LoadInt64(&c.value)
}

// Часть B: Litmus-тест

type LitmusResult struct {
	BothZero int64
	R1Only   int64
	R2Only   int64
	BothOne  int64
}

const cacheLineSize = 64

type PaddedInt64 struct {
	Value int64
	Pad   [cacheLineSize - unsafe.Sizeof(int64(0))]byte
}

func Litmus(n int, seqcst bool) LitmusResult {
	var result LitmusResult

	for round := 0; round < n; round++ {
		var x, y int64
		var r1, r2 int64

		var flag1, flag2 PaddedInt64

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			delay := rand.Intn(20)
			for i := 0; i < delay; i++ {
			}
			atomic.StoreInt64(&flag1.Value, 1)
			for atomic.LoadInt64(&flag2.Value) == 0 {
			}
			if seqcst {
				atomic.StoreInt64(&x, 1)
				r1 = atomic.LoadInt64(&y)
			} else {
				x = 1
				r1 = y
			}
		}()

		go func() {
			defer wg.Done()
			delay := rand.Intn(20)
			for i := 0; i < delay; i++ {
			}
			atomic.StoreInt64(&flag2.Value, 1)
			for atomic.LoadInt64(&flag1.Value) == 0 {
			}
			if seqcst {
				atomic.StoreInt64(&y, 1)
				r2 = atomic.LoadInt64(&x)
			} else {
				y = 1
				r2 = x
			}
		}()

		wg.Wait()

		if r1 == 0 && r2 == 0 {
			result.BothZero++
		} else if r1 == 1 && r2 == 0 {
			result.R1Only++
		} else if r1 == 0 && r2 == 1 {
			result.R2Only++
		} else {
			result.BothOne++
		}
	}

	return result
}

// Часть C: Кольцевой буфер SPSC

type SPSCQueue struct {
	data     []int64
	capacity int64
	mask     int64
	head     int64
	_        [cacheLineSize - unsafe.Sizeof(int64(0))]byte
	tail     int64
}

func NewSPSCQueue(capacity int) *SPSCQueue {
	cap := 1 << bits.Len(uint(capacity-1))
	if cap < capacity {
		cap <<= 1
	}
	return &SPSCQueue{
		data:     make([]int64, cap),
		capacity: int64(cap),
		mask:     int64(cap - 1),
		head:     0,
		tail:     0,
	}
}

func (q *SPSCQueue) Capacity() int {
	return int(q.capacity)
}

func (q *SPSCQueue) TryPush(value int64) bool {
	tail := q.tail
	head := atomic.LoadInt64(&q.head)
	if tail-head >= q.capacity {
		return false
	}
	q.data[tail&q.mask] = value
	q.tail = tail + 1
	return true
}

func (q *SPSCQueue) TryPop() (int64, bool) {
	head := q.head
	tail := atomic.LoadInt64(&q.tail)
	if head >= tail {
		return 0, false
	}
	value := q.data[head&q.mask]
	q.head = head + 1
	return value, true
}

func Answer() uint64 {
	return 42
}
