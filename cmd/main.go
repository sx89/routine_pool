package cmd

import (
	"fmt"
	"github.com/eapache/queue"
	routinePool "routine-pool/core"
	"runtime"
	"sync"
	"time"
)

var (
	wp    *routinePool.Pool
	q     queue.Queue
	group sync.WaitGroup
)

func createPool() *routinePool.Pool {
	conf := &routinePool.PoolConfig{
		MaxWorkers:     1024,
		MaxIdleWorkers: 512,
		MinIdleWorkers: 256,
		KeepAlive:      30 * time.Second,
	}
	p, err := routinePool.NewWorkerPool(2048, conf)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	if err := p.Start(); err != nil {
		fmt.Println(err)
		return nil
	}
	return p
}

// TestTask .
type TestTask struct {
	name string
}

// Run .
func (t *TestTask) Run() *[]byte {
	// fmt.Println(t.name)
	res := []byte(t.name)
	time.Sleep(100 * time.Millisecond)
	return &res
}

func producer(id int) {
	for j := 0; j < 10000; j++ {
		ft := routinePool.NewFutureTask(&TestTask{
			name: fmt.Sprintf("t:p %d, id %d", id, j),
		})
		for {
			err := wp.Submit(ft)
			if err == nil {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		//q.Put(ft)
		q.Add(ft)
		group.Add(1)
	}
	group.Done()
}

func consumer(id int) {
	for {
		//inter, _ := q.Poll(1, 10*time.Second)
		inter := q.Get(1)
		if inter == nil {
			continue
		}
		//task := inter[0].(*routinePool.FutureTask)
		task := inter.(*routinePool.FutureTask)
		// task.Wait(3 * time.Second)
		res, err := task.Wait(1 * time.Second)
		if err != nil {
			fmt.Printf("c: %d; t: %+v err: %+v \n", id, inter, err)
		} else {
			fmt.Printf("c: %d; t: %+v err: %+v res: %+v \n", id, inter, err, string(*res))
		}

		group.Done()
	}
}

func main() {

	runtime.GOMAXPROCS(8)
	wp = createPool()
	// consumer
	for i := 0; i < 10; i++ {
		go consumer(i)
	}

	// producer
	for i := 0; i < 10; i++ {
		group.Add(1)
		go producer(i)
	}

	group.Wait()
}
