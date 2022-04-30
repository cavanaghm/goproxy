package config

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	lc "reverseProxy/lc"
)

type Proxy struct {
	Name    string   `json:name`
	Listen  string   `json:listen`
	Balance string   `json:balance`
	Targets []string `json:targets`
}

func Bootstrap() map[string]lc.HandleProxy {
	content, err := ioutil.ReadFile("./config/index.json")
	if err != nil {
		log.Fatal("Error reading config file: ", err)
	}

	var Servers []Proxy
	err = json.Unmarshal(content, &Servers)
	if err != nil {
		log.Fatal("JSON error: ", err)
	}

	for i := range Servers {
		fmt.Println(Servers[i].Name)
		fmt.Println(Servers[i].Listen)
		createProxy(Servers[i])
	}
	return HostProxy
}

var HostProxy map[string]lc.HandleProxy = map[string]lc.HandleProxy{}

func createProxy(Server Proxy) {
	//Create slice to store the targets
	proxies := make([]*httputil.ReverseProxy, len(Server.Targets))

	for i := range Server.Targets {
		url, err := url.Parse(Server.Targets[i])
		if err != nil {
			log.Fatal("Parse target failed for: ", err)
		}

		proxy := httputil.NewSingleHostReverseProxy(url)
		proxy.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}

		proxies[i] = proxy
	}

	if Server.Balance == "leastConnections" {
		nextServer, err := lc.LeastConnectionsProxy(proxies)
		if err != nil {
			log.Fatal("Failed to create least connections proxy")
		}

		HostProxy[Server.Listen] = nextServer
	}
	return
}
