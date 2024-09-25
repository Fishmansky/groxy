package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

type GoProxy struct {
	server     *http.Server
	listenAddr string
	proxyAddr  string
	timeout    time.Duration
}

func NewGoProxy(l string, p string, t time.Duration) *GoProxy {
	return &GoProxy{
		listenAddr: l,
		proxyAddr:  p,
		timeout:    t,
	}
}

type ProxyReqHandler struct {
	listenAddr string
	proxyAddr  string
}

func NewProxyReqHandler(l string, p string) *ProxyReqHandler {
	return &ProxyReqHandler{
		listenAddr: l,
		proxyAddr:  p,
	}
}

func (p *ProxyReqHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	client := &http.Client{}
	payload := bytes.NewBufferString("")
	r.ParseForm()
	if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" && len(r.Form) > 0 {
		// add form values
		form := url.Values{}
		for k, v := range r.Form {
			form.Set(k, v[0])
		}
		payload = bytes.NewBufferString(form.Encode())
	}
	var req *http.Request
	var err error
	if payload != nil {
		// add payload to body if payload exist
		req, err = http.NewRequest(r.Method, p.proxyAddr+r.RequestURI, payload)
	} else {
		req, err = http.NewRequest(r.Method, p.proxyAddr+r.RequestURI, nil)
	}
	if err != nil {
		log.Fatal(err)
	}
	// add cookies to request
	for _, c := range r.Cookies() {
		req.AddCookie(c)
	}
	// add headers to request
	for h := range r.Header {
		req.Header.Add(h, r.Header.Get(h))
	}
	// make request
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// read body
	body, err := io.ReadAll(resp.Body)
	// add cookies to response
	for _, c := range resp.Cookies() {
		http.SetCookie(w, c)
	}
	// add headers to response
	for h := range resp.Header {
		w.Header().Set(h, resp.Header.Get(h))
	}
	log.Printf("%s %s %d %s\n", r.Host, r.Method, resp.StatusCode, r.URL)
	w.Write(body)
}

func (g *GoProxy) Start() {
	p := NewProxyReqHandler(g.listenAddr, g.proxyAddr)
	s := &http.Server{
		Addr:    g.listenAddr,
		Handler: p,
	}
	g.server = s
	log.Fatal(g.server.ListenAndServe())
}

func main() {
	gp := NewGoProxy("127.0.0.1:8081", "http://127.0.0.1:8080", 30)
	gp.Start()
}
