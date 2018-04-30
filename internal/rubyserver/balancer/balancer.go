package balancer

// In this package we manage a global pool of addresses for gitaly-ruby,
// accessed via the gitaly-ruby:// scheme. The interface consists of the
// AddAddress and RemoveAddress methods. RemoveAddress returns a boolean
// indicating whether the address was removed; this is intended to give
// back-pressure against repeated process restarts.
//
// The gitaly-ruby:// scheme exists because that is the way we can
// interact with the internal client-side loadbalancer of grpc-go. A URL
// for this scheme would be gitaly-ruby://foobar. For gitaly-ruby://
// URL's, the host and port are ignored. So gitaly-ruby://foobar is
// actually a working, valid address.
//
// Strictly speaking this package implements a gRPC 'Resolver'. This
// resolver feeds address list updates to a gRPC 'balancer' which
// interacts with the gRPC client connection machinery. A resolver
// consists of a Builder which returns Resolver instances. Our Builder
// manages the address pool and notifies its Resolver instances of
// changes, which they then propagate into the gRPC library.
//

import (
	"google.golang.org/grpc/resolver"
)

var (
	lbBuilder = newBuilder()
)

func init() {
	resolver.Register(lbBuilder)
}

// AddAddress adds the address of a gitaly-ruby instance to the load
// balancer.
func AddAddress(a string) {
	lbBuilder.addAddress <- a
}

// RemoveAddress removes the address of a gitaly-ruby instance from the
// load balancer. Returns false if the pool is too small to remove the
// address.
func RemoveAddress(addr string) bool {
	ok := make(chan bool)
	lbBuilder.removeAddress <- addressRemoval{ok: ok, addr: addr}
	return <-ok
}

type addressRemoval struct {
	addr string
	ok   chan<- bool
}

type addressUpdate struct {
	addrs []resolver.Address
	next  chan struct{}
}

type builder struct {
	addAddress     chan string
	removeAddress  chan addressRemoval
	addressUpdates chan addressUpdate
}

func newBuilder() *builder {
	b := &builder{
		addAddress:     make(chan string),
		removeAddress:  make(chan addressRemoval),
		addressUpdates: make(chan addressUpdate),
	}
	go b.monitor()

	return b
}

// Scheme is the name of the address scheme that makes gRPC select this resolver.
const Scheme = "gitaly-ruby"

func (*builder) Scheme() string { return Scheme }

// Build ignores its resolver.Target argument. That means it does not
// care what "address" the caller wants to resolve. We always resolve to
// the current list of address for local gitaly-ruby processes.
func (b *builder) Build(_ resolver.Target, cc resolver.ClientConn, _ resolver.BuildOption) (resolver.Resolver, error) {
	// JV: Normally I would delete this but this is very poorly documented,
	// and I don't want to have to look up the magic words again. In case we
	// ever want to do round-robin.
	// cc.NewServiceConfig(`{"LoadBalancingPolicy":"round_robin"}`)

	return newGitalyResolver(cc, b.addressUpdates), nil
}

// monitor serves address list requests and handles address updates.
func (b *builder) monitor() {
	addresses := make(map[string]struct{})
	notify := make(chan struct{})

	for {
		au := addressUpdate{next: notify}
		for a := range addresses {
			au.addrs = append(au.addrs, resolver.Address{Addr: a})
		}

		select {
		case b.addressUpdates <- au:
			if len(au.addrs) == 0 {
				panic("builder monitor sent empty address update")
			}
		case addr := <-b.addAddress:
			addresses[addr] = struct{}{}
			notify = broadcast(notify)
		case removal := <-b.removeAddress:
			_, addressKnown := addresses[removal.addr]
			if !addressKnown || len(addresses) <= 1 {
				removal.ok <- false
				break
			}

			delete(addresses, removal.addr)
			removal.ok <- true
			notify = broadcast(notify)
		}
	}
}

// broadcast returns a fresh channel because we can only close them once
func broadcast(ch chan struct{}) chan struct{} {
	close(ch)
	return make(chan struct{})
}

// gitalyResolver propagates address list updates to a
// resolver.ClientConn instance
type gitalyResolver struct {
	clientConn resolver.ClientConn

	started        chan struct{}
	done           chan struct{}
	resolveNow     chan struct{}
	addressUpdates chan addressUpdate
}

func newGitalyResolver(cc resolver.ClientConn, auCh chan addressUpdate) *gitalyResolver {
	r := &gitalyResolver{
		started:        make(chan struct{}),
		done:           make(chan struct{}),
		resolveNow:     make(chan struct{}),
		addressUpdates: auCh,
		clientConn:     cc,
	}
	go r.monitor()

	// Don't return until we have sent at least one address update. This is
	// meant to avoid panics inside the grpc-go library.
	<-r.started

	return r
}

func (r *gitalyResolver) ResolveNow(resolver.ResolveNowOption) {
	r.resolveNow <- struct{}{}
}

func (r *gitalyResolver) Close() {
	close(r.done)
}

func (r *gitalyResolver) monitor() {
	notify := r.sendUpdate()
	close(r.started)

	for {
		select {
		case <-notify:
			notify = r.sendUpdate()
		case <-r.resolveNow:
			notify = r.sendUpdate()
		case <-r.done:
			return
		}
	}
}

func (r *gitalyResolver) sendUpdate() chan struct{} {
	au := <-r.addressUpdates
	r.clientConn.NewAddress(au.addrs)
	return au.next
}
