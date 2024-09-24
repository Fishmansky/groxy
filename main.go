package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/netip"
	"net/url"
	"time"
)

type GoProxy struct {
	server     http.Server
	listenAddr netip.AddrPort
	proxyAddr  netip.AddrPort
	timeout    time.Duration
}

func NewGoProxy(l string, p string, t time.Duration) *GoProxy {
	lp, err := netip.ParseAddrPort(l)
	if err != nil {
		log.Fatal(err)
	}
	pp, err := netip.ParseAddrPort(p)
	if err != nil {
		log.Fatal(err)
	}
	return &GoProxy{
		listenAddr: lp,
		proxyAddr:  pp,
		timeout:    t,
	}
}

func (g *GoProxy) proxyReqHandler(w http.ResponseWriter, r *http.Request) {
	client := &http.Client{}
	payload := bytes.NewBufferString("")
	if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" && len(r.Form) > 0 {
		// add form values
		err := r.ParseForm()
		if err != nil {
			log.Fatal(err)
		}
		form := url.Values{}
		for k, v := range r.Form {
			log.Println(k, v)
			form.Set(k, v[0])
		}
		payload = bytes.NewBufferString(form.Encode())
	}
	var req *http.Request
	var err error
	if payload != nil {
		req, err = http.NewRequest(r.Method, "http://"+g.proxyAddr.String()+r.RequestURI, payload)
	} else {
		req, err = http.NewRequest(r.Method, "http://"+g.proxyAddr.String()+r.RequestURI, nil)
	}
	if err != nil {
		log.Fatal(err)
	}
	// add cookies
	log.Println(r.Cookies())
	for _, c := range r.Cookies() {
		log.Printf("Adding cookie %s to request", c.Name)
		req.AddCookie(c)
	}
	// add headers
	for h := range r.Header {
		req.Header.Add(h, r.Header.Get(h))
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	for _, c := range resp.Cookies() {
		log.Printf("Adding cookie %s to response", c.Name)
		http.SetCookie(w, c)
	}
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "text/plain"
	}
	w.Header().Set("Content-Type", ct)
	log.Printf("%s %s %d %s\n", r.Host, r.Method, resp.StatusCode, r.URL)
	w.Write(body)
}

func (g *GoProxy) Start() {
	http.HandleFunc("/", g.proxyReqHandler)
	log.Fatal(g.server.ListenAndServe())
}

func main() {
	gp := NewGoProxy("127.0.0.1:80", "127.0.0.1:8080", 30)
	gp.Start()
}
