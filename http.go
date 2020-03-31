// 分布式缓存需要实现节点间通信
package dcache

import (
	"context"
	"dcache/cachepb"
	"dcache/consistenthash"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const defaultBasePath = "/_cache/"
const defaultReplicas = 3

type HTTPPool struct {
	self     string //记录自身地址
	basePath string // 通信地址前缀

	peers   *consistenthash.Map
	mu      sync.Mutex
	getters map[string]PeerGetter
}

func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Cache Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpect path..." + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)
	// /<basepath>/<groupname>/<key> required
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName, key := parts[0], parts[1]
	group := GetGroup(groupName)
	if group == nil {
		// 本机没有这个缓存group
		http.Error(w, "no such group", http.StatusNotFound)
		return
	}

	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	body, err := proto.Marshal(&cachepb.Response{Value: view.ByteSlice()})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream") // 二进制流
	w.Write(body)
}

var _ cachepb.GroupCacheServer = (*HTTPPool)(nil)

func (p *HTTPPool) Get(ctx context.Context, req *cachepb.Request) (*cachepb.Response, error) {
	groupName := req.GetGroup()
	group := GetGroup(groupName)
	if group == nil {
		// 本机没有这个缓存group
		return nil, errors.New("no such group")
	}
	view, err := group.Get(req.GetKey())
	if err != nil {
		return nil, err
	}
	return &cachepb.Response{Value: view.ByteSlice()}, err
}

type httpGetter struct {
	baseURL string
}

var _ PeerGetter = (*httpGetter)(nil)

func (h *httpGetter) Get(in *cachepb.Request, out *cachepb.Response) error {
	u := fmt.Sprintf("%v%v/%v", h.baseURL, url.QueryEscape(in.GetGroup()), url.QueryEscape(in.GetKey()))
	log.Println("get remote dcache url", u)
	res, err := http.Get(u)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned: %v", res.StatusCode)
	}

	bytes, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return fmt.Errorf("reading response bidy: %v", err)
	}

	if err = proto.Unmarshal(bytes, out); err != nil {
		return fmt.Errorf("decoding response body: %v", err)
	}
	return nil

}

type rpcGetter struct {
	baseRPCAddr string
}

var _ PeerGetter = (*rpcGetter)(nil)

func (r *rpcGetter) Get(in *cachepb.Request, out *cachepb.Response) error {
	log.Println("get remote dcache rpc address ", r.baseRPCAddr)
	conn, err := grpc.Dial(r.baseRPCAddr)
	if err != nil {
		return nil
	}
	defer conn.Close()
	cli := cachepb.NewGroupCacheClient(conn)
	out, err = cli.Get(context.Background(), in) // rpc 调用
	return err
}

func (p *HTTPPool) Set(t GetterType, peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	if p.getters == nil {
		p.getters = make(map[string]PeerGetter, len(peers))
	}
	switch t {
	case HttpGetter:
		for _, peer := range peers {
			p.getters[peer] = &httpGetter{baseURL: peer + p.basePath}
		}
	case RpcGetter:
		for _, peer := range peers {
			p.getters[peer] = &rpcGetter{baseRPCAddr: peer}
		}
	}

}

func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// peer=="" 表示一个key都没有,表示不是自己的数据不接收peer != p.self
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		return p.getters[peer], true
	}
	return nil, false
}

var _ PeerPicker = (*HTTPPool)(nil)
