package grpcpool

import (
	"errors"
	"log"
	"sync"
	"time"

	"google.golang.org/grpc"
)

var (
	ErrParam  = errors.New("grpc pool: params error")
	ErrClosed = errors.New("grpc pool: pool closed")
	ErrCreate = errors.New("grpc pool: create conn error")
	ErrFull   = errors.New("grpc pool: pool full")
)

type Pool struct {
	conns            chan *poolConn
	mu               sync.RWMutex
	factory          func() (*grpc.ClientConn, error)
	idleTimeout      time.Duration
	lifetimeDuration time.Duration
}

type poolConn struct {
	Conn       *grpc.ClientConn
	createTime time.Time
	UpdateTime time.Time
}

func NewPool(factory func() (*grpc.ClientConn, error), len int, cap int, idleTimeout time.Duration, lifetimeDuration time.Duration) (*Pool, error) {
	if len < 0 || cap <= 0 || len > cap {
		return nil, ErrParam
	}

	pool := &Pool{
		conns:            make(chan *poolConn, cap),
		factory:          factory,
		idleTimeout:      idleTimeout,
		lifetimeDuration: lifetimeDuration,
	}

	for i := 0; i < len; i++ {
		c, err := factory()
		if err != nil {
			return nil, err
		}
		pool.conns <- &poolConn{
			Conn:       c,
			createTime: time.Now(),
			UpdateTime: time.Now(),
		}
	}

	return pool, nil
}

func (p *Pool) Get() (*poolConn, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.conns == nil {
		return nil, ErrClosed
	}

	select {
	case conn, ok := <-p.conns:
		if !ok {
			return nil, ErrClosed
		}
		if p.idleTimeout > 0 && conn.UpdateTime.Add(p.idleTimeout).Before(time.Now()) {
			log.Println("idle timeout, recreate")
			conn.Conn.Close()
			return p.createNewConn()
		}
		if p.lifetimeDuration > 0 && conn.createTime.Add(p.lifetimeDuration).Before(time.Now()) {
			log.Println("reach lifetime, recreate")
			conn.Conn.Close()
			return p.createNewConn()
		}
		log.Println("get a connection")
		conn.UpdateTime = time.Now()
		return conn, nil
	default:
		log.Println("new a connection")
		return p.createNewConn()
	}
}

func (p *Pool) createNewConn() (*poolConn, error) {
	c, err := p.factory()
	if err != nil {
		return nil, ErrCreate
	}
	conn := &poolConn{
		Conn:       c,
		createTime: time.Now(),
		UpdateTime: time.Now(),
	}
	return conn, nil
}

func (p *Pool) Put(conn *poolConn, forceClose bool) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.conns == nil {
		return ErrClosed
	}

	if conn == nil {
		return nil
	}

	if forceClose {
		conn.Conn.Close()
		return nil
	}

	select {
	case p.conns <- conn:
		return nil
	default:
		log.Println("pool is full, close conn")
		conn.Conn.Close()
		return ErrFull
	}
}

func (p *Pool) Len() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.conns)
}

func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	close(p.conns)
	for conn := range p.conns {
		conn.Conn.Close()
	}
	p.conns = nil
	return nil
}
