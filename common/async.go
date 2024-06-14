package common

type Promise[T any] struct {
	ResultCh chan T
	ErrorCh  chan error
}

func (p Promise[T]) Wait() (T, error) {
	err := <-p.ErrorCh
	res := <-p.ResultCh

	return res, err
}

func (p Promise[T]) Close() {
	close(p.ErrorCh)
	close(p.ResultCh)
}

func Async[T any](fn func() (T, error)) Promise[T] {
	p := Promise[T]{
		ResultCh: make(chan T, 1),
		ErrorCh:  make(chan error, 1),
	}

	go func() {
		defer p.Close()

		res, err := fn()
		p.ResultCh <- res
		p.ErrorCh <- err
	}()

	return p
}
