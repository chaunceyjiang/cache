package main

import (
	"dcache"
	"flag"
	"fmt"
	"log"
	"net/http"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func createGroup() *dcache.Group {
	return dcache.NewGroup("source", 2<<10, dcache.GetterFunc(func(key string) ([]byte, error) {
		log.Println("[cache] search key", key)
		if v, ok := db[key]; ok {
			return []byte(v), nil
		}
		return nil, fmt.Errorf("%s not exist", key)
	}))
}

func startCacheServer(addr string, addrs []string, dc *dcache.Group) {
	peers := dcache.NewHTTPPool(addr)
	// HTTPPool 即实现了ServeHTTP，又实现了PeerPicker
	peers.Set(addrs...)
	dc.RegisterPeers(peers)
	log.Println("dcache is running ad ", addr)
	log.Fatalln(http.ListenAndServe(addr[7:], peers))
}

func startAPIServer(addr string, dc *dcache.Group) {
	http.Handle("/api", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		view, err := dc.Get(key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(view.ByteSlice())
	}))
	log.Println("frontend server is running at ", addr)
	log.Fatalln(http.ListenAndServe(addr[7:], nil))
}
func main() {

	var port int
	var api bool
	flag.IntVar(&port, "port", 8001, "Geecache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()
	apiAddr := "http://localhost:9999"
	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}
	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}
	dc := createGroup()
	if api {
		go startAPIServer(apiAddr, dc)
	}
	startCacheServer(addrMap[port], addrs, dc)
}
