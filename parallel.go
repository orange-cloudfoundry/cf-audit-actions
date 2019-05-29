package main

import (
	"context"
	"golang.org/x/sync/semaphore"
	"runtime"
	"sync"
)

var sem *semaphore.Weighted
var waitgroup sync.WaitGroup
var parrallelError = make(chan error, 1000)

func initParallel() {
	parallel := options.Parallel
	if parallel <= 0 {
		parallel = runtime.GOMAXPROCS(0)
	}
	sem = semaphore.NewWeighted(int64(parallel))
}

func RunParallel(meta interface{}, f func(meta interface{}) error) {
	waitgroup.Add(1)
	go func(meta interface{}) {
		defer waitgroup.Done()
		err := sem.Acquire(context.TODO(), 1)
		if err != nil {
			parrallelError <- err
			return
		}
		defer sem.Release(1)
		err = f(meta)
		if err != nil {
			parrallelError <- err
		}
	}(meta)
}

func WaitParallel() []error {
	waitgroup.Wait()
	errors := make([]error, 0)

	for {
		select {
		case err := <-parrallelError:
			errors = append(errors, err)
		default:
			return errors
		}
	}
	return errors
}
