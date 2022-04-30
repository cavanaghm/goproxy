package main

import (
	"log"
	"net/http"
	"time"

	config "reverseProxy/config"
)

var hostProxy = config.Bootstrap()

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
