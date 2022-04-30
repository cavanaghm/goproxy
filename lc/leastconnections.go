package leastconnections

import (
	"errors"
	"net/http/httputil"
	"net/url"
	"sync"
)

//
type Handler interface {
	Next() (next *url.URL, done func())
}

type HandleProxy interface {
	NextProxy() (next *httputil.ReverseProxy, done func())
}

//De
type leastConnections struct {
	connections []connection
	mu          *sync.Mutex
}

type leastConnectionsProxy struct {
	connections []proxy
	mu          *sync.Mutex
}

type connection struct {
	url *url.URL
	cnt int
}

type proxy struct {
	proxy *httputil.ReverseProxy
	cnt   int
}

//Initialise the counter
// @param {[]*url.URL} an Array of url objects
// @returns {LeastConnections}
func LeastConnections(urls []*url.URL) (Handler, error) {
	if len(urls) == 0 {
		return nil, errors.New("no urls defined")
	}

	connections := make([]connection, len(urls))
	for i := range connections {
		connections[i] = connection{
			url: urls[i],
			cnt: 0,
		}
	}

	return &leastConnections{
		connections: connections,
		mu:          new(sync.Mutex),
	}, nil
}

func LeastConnectionsProxy(proxies []*httputil.ReverseProxy) (HandleProxy, error) {
	if len(proxies) == 0 {
		return nil, errors.New("no urls defined")
	}

	connections := make([]proxy, len(proxies))
	for i := range connections {
		connections[i] = proxy{
			proxy: proxies[i],
			cnt:   0,
		}
	}

	return &leastConnectionsProxy{
		connections: connections,
		mu:          new(sync.Mutex),
	}, nil
}

func (lc *leastConnections) Next() (*url.URL, func()) {
	var (
		min = -1 //Minimum connections before using new server
		idx int  //Server to use
	)

	lc.mu.Lock()

	for i, connection := range lc.connections {
		if min == -1 || connection.cnt < min {
			min = connection.cnt
			idx = i
		}
	}
	lc.connections[idx].cnt++

	lc.mu.Unlock()

	var done bool
	//Returns the URL to use and a function to de-increment once connection closed
	return lc.connections[idx].url, func() {
		lc.mu.Lock()
		if !done {
			lc.connections[idx].cnt--
			done = true
		}
		lc.mu.Lock()
	}
}

func (lc *leastConnectionsProxy) NextProxy() (*httputil.ReverseProxy, func()) {
	var (
		min = -1 //Minimum connections before using new server
		idx int  //Server to use
	)

	lc.mu.Lock()

	for i, connection := range lc.connections {
		if min == -1 || connection.cnt < min {
			min = connection.cnt
			idx = i
		}
	}
	lc.connections[idx].cnt++

	lc.mu.Unlock()

	var done bool
	//Returns the URL to use and a function to de-increment once connection closed
	return lc.connections[idx].proxy, func() {
		lc.mu.Lock()
		if !done {
			lc.connections[idx].cnt--
			done = true
		}
		lc.mu.Unlock()
	}
}
