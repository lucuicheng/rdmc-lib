package main

import (
	"fmt"
	"sync"
)

type Queue struct {
	items []int
	lock  sync.Mutex
	cond  *sync.Cond
}

func NewQueue() *Queue {
	queue := &Queue{}
	queue.cond = sync.NewCond(&queue.lock)
	return queue
}

func (q *Queue) Enqueue(item int) {
	q.lock.Lock()
	defer q.lock.Unlock()

	q.items = append(q.items, item)
	q.cond.Signal()
}

func (q *Queue) Dequeue() (int, error) {
	q.lock.Lock()
	defer q.lock.Unlock()

	for len(q.items) == 0 {
		q.cond.Wait()
	}

	item := q.items[0]
	q.items = q.items[1:]

	return item, nil
}

func (q *Queue) IsEmpty() bool {
	q.lock.Lock()
	defer q.lock.Unlock()

	return len(q.items) == 0
}

func main() {
	queue := NewQueue()

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		queue.Enqueue(1)
	}()

	go func() {
		defer wg.Done()
		queue.Enqueue(2)
	}()

	go func() {
		defer wg.Done()
		queue.Enqueue(3)
	}()

	wg.Wait()
	fmt.Println("Is queue empty?", queue.IsEmpty())

	//// 等待所有Enqueue操作完成后再执行Dequeue和IsEmpty操作
	//wg.Add(2)
	//
	//go func() {
	//	defer wg.Done()
	//	item, _ := queue.Dequeue()
	//	fmt.Println("Dequeued item:", item)
	//}()
	//
	//go func() {
	//	defer wg.Done()
	//	fmt.Println("Is queue empty?", queue.IsEmpty())
	//}()

	//wg.Wait()
}
