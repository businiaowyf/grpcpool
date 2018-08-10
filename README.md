# grpcpool
This package aims to provide an easy to use and lightweight GRPC connection pool.

Please note that the goal isn't to replicate the client-side load-balancing feature of the official grpc package: the goal is rather to have multiple connections established to one endpoint (which can be server-side load-balanced).

## Install
go get github.com/businiaowyf/grpcpool

## Get Started
```
package main

import (
    "log"

    "github.com/businiaowyf/grpcpool"
    "google.golang.org/grpc"
)

func main() {
    p, err := grpcpool.NewPool(
        func() (*grpc.ClientConn, error) {
            return grpc.Dial("localhost:50051", grpc.WithInsecure())
        }, 0, 3, 0, 0)
    if err != nil {
        log.Println(err)
        return
    }

    conn, err := p.Get()
    defer p.Put(conn, false)
}
```

If you use etcdv3 as a naming service, you can also use it like this:
```
p, err := grpcpool.NewPool(
    r := etcdv3.NewResolver("localhost:2379;localhost:22379;localhost:32379")
    resolver.Register(r)
    func() (*grpc.ClientConn, error) {
    return grpc.Dial(r.Scheme()+"://authority/"+serviceName, grpc.WithBalancerName("round_robin"), grpc.WithInsecure())
    }, 0, 3, 0, 0
)
```
Of course, you need to implement your own resolver.Builder.
