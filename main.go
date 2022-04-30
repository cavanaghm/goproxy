package main

import (
	"crypto/tls"
	lc "lc"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

var (
	hostTarget = map[string][]string{
		"pi.192.168.1.136:8080": {"192.168.1.136/admin", "192.168.1.136/admin"},
		"pi.sideme.xyz":         {"http://192.168.1.136/admin"},
		"test.sideme.xyz":       {"http://localhost:3000/a", "http://localhost:3000/b", "http://localhost:3000/c"},
		"pi.localhost:8080":     {"http://192.168.1.136/admin"},
		"app.192.168.1.136":     {"192.168.1.136"},
	}
	hostProxy map[string]lc.HandleProxy = map[string]lc.HandleProxy{}
)

func main() {
	httpsMux := http.NewServeMux()

	server := &http.Server{
		Addr:           ":8080",
		Handler:        httpsMux,
		MaxHeaderBytes: 1 << 20,
		//Handler: http.HandlerFunc(handler),
	}

	httpsMux.Handle("/", logger(handler))

	httpListener := &http.Server{
		Addr:    ":8000",
		Handler: http.HandlerFunc(redirect),
	}

	go httpListener.ListenAndServe()
	log.Fatal(server.ListenAndServeTLS("./cert.pem", "./key.pem"))
}

func redirect(res http.ResponseWriter, req *http.Request) {
	http.Redirect(res, req, "https://"+req.Host, http.StatusFound)
}

func handler(res http.ResponseWriter, req *http.Request) {
	host := req.Host

	if target, ok := hostProxy[host]; ok {

		proxy, done := target.NextProxy()

		proxy.ServeHTTP(res, req)
		defer done()
		return
	}

	if target, ok := hostTarget[host]; ok {
		proxies := make([]*httputil.ReverseProxy, len(hostTarget[host]))
		for i := range hostTarget[host] {
			remoteUrl, err := url.Parse(target[i])
			if err != nil {
				log.Println("Parse target failed:", err)
				return
			}
			proxy := httputil.NewSingleHostReverseProxy(remoteUrl)
			proxy.Transport = &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			proxies[i] = proxy
		}

		nextServer, err := lc.LeastConnectionsProxy(proxies)
		if err != nil {
			log.Fatal("panic")
		}

		hostProxy[host] = nextServer

		proxy, done := nextServer.NextProxy()

		proxy.ServeHTTP(res, req)
		defer done()
		return
	}
	res.WriteHeader(http.StatusForbidden)
	res.Write([]byte("403: Host forbidden " + host))
}

func logger(next http.HandlerFunc) http.HandlerFunc {
	fn := func(res http.ResponseWriter, req *http.Request) {
		t1 := time.Now()
		next.ServeHTTP(res, req)
		t2 := time.Now()
		log.Printf("[%s] %q %v\n", req.Method, req.URL.String(), t2.Sub(t1))
	}

	return http.HandlerFunc(fn)
}
