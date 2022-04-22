package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/elitah/fast-io"
	"github.com/elitah/utils/exepath"
	"github.com/elitah/websocket/websocket"

	"github.com/xtaci/smux"
)

func main() {
	//
	var workdir string
	//
	var port int
	//
	var mutex sync.RWMutex
	//
	flag.StringVar(&workdir, "w", exepath.GetExeDir(), "work path")
	//
	flag.IntVar(&port, "p", 8230, "listen port")
	//
	flag.Parse()
	//
	m := make(map[string]*smux.Session)
	//
	if l, err := net.ListenTCP("tcp4", &net.TCPAddr{
		Port: port,
	}); nil == err {
		//
		go func() {
			//
			for {
				//
				if conn, err := l.Accept(); nil == err {
					//
					if c, err := smux.Client(conn, smux.DefaultConfig()); nil == err {
						//
						key := conn.RemoteAddr().String()
						//
						fmt.Println("new:", key)
						//
						mutex.Lock()
						//
						if _c, ok := m[key]; ok {
							//
							if !_c.IsClosed() {
								//
								_c.Close()
							}
							//
							delete(m, key)
						}
						//
						m[key] = c
						//
						mutex.Unlock()
						//
						continue
					} else {
						//
						fmt.Println("smux.Client:", err)
					}
					//
					conn.Close()
				} else {
					//
					break
				}
			}
		}()
	} else {
		//
		fmt.Println(err)
		//
		return
	}
	//
	go func() {
		//
		for {
			//
			mutex.Lock()
			//
			for key, value := range m {
				//
				if value.IsClosed() {
					//
					delete(m, key)
				}
			}
			//
			mutex.Unlock()
			//
			time.Sleep(time.Second)
		}
	}()
	//
	ws := websocket.NewServer(nil)
	//
	wc := websocket.NewClient()
	//
	ws.AddAuthHandler(func(v *websocket.Values, args interface{}) bool {
		//
		if r, ok := args.(*http.Request); ok {
			//
			if key := r.FormValue("key"); "" != key {
				//
				v.KVSet("key", key)
				//
				return true
			}
		}
		//
		return false
	})
	//
	ws.AddRawConnectionHandler(func(conn *websocket.Conn) {
		//
		key := conn.KVGetString("key")
		//
		if _conn, err := wc.Dial(fmt.Sprintf("ws://%s/ws", key)); nil == err {
			//
			fmt.Println("ok")
			//
			fast_io.FastCopy(conn, _conn)
			//
			_conn.Close()
		} else {
			//
			var s *smux.Session
			//
			fmt.Println("websocket.Dial:", err)
			//
			mutex.RLock()
			//
			if _s, ok := m[key]; ok {
				//
				s = _s
			}
			//
			mutex.RUnlock()
			//
			if nil != s {
				//
				s.Close()
			}
		}
		//
		conn.Close()
	})
	//
	wc.SetNetDial(func(network, addr string) (net.Conn, error) {
		//
		var s *smux.Session
		//
		mutex.RLock()
		//
		if _s, ok := m[addr]; ok {
			//
			s = _s
		}
		//
		mutex.RUnlock()
		//
		if nil != s {
			//
			if stream, err := s.OpenStream(); nil == err {
				//
				return stream, nil
			} else {
				//
				fmt.Println("smux.Session.OpenStream:", err)
			}
		} else {
			//
			fmt.Println("stream not found")
		}
		//
		return nil, io.EOF
	})
	//
	http.ListenAndServe(":11080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//
		switch r.URL.Path {
		case "/":
			//
			if data, err := ioutil.ReadFile(
				filepath.Join(
					workdir,
					"index.html",
				),
			); nil == err {
				//
				w.Header().Set("Content-Type", "text/html")
				//
				w.WriteHeader(http.StatusOK)
				//
				w.Write(data)
				//
				return
			} else {
				//
				fmt.Println("readfile:", err)
			}
			//
			http.NotFound(w, r)
			//
			return
		case "/list":
			//
			var list []string
			//
			mutex.RLock()
			//
			for key, _ := range m {
				//
				list = append(list, key)
			}
			//
			mutex.RUnlock()
			//
			if data, err := json.Marshal(list); nil == err {
				//
				w.Header().Set("Content-Type", "application/json")
				//
				w.WriteHeader(http.StatusOK)
				//
				w.Write(data)
				//
				return
			} else {
				//
				fmt.Println("json.Marshal:", err)
			}
			//
			http.NotFound(w, r)
			//
			return
		case "/ws":
			//
			ws.Upgrade(w, r, r)
			//
			return
		default:
			//
			if data, err := ioutil.ReadFile(
				filepath.Join(
					workdir,
					r.URL.Path,
				),
			); nil == err {
				//
				if ct := mime.TypeByExtension(filepath.Ext(r.URL.Path)); "" != ct {
					//
					w.Header().Set("Content-Type", ct)
				}
				//
				w.WriteHeader(http.StatusOK)
				//
				w.Write(data)
				//
				return
			} else {
				//
				fmt.Println("readfile:", err)
			}
			//
			http.NotFound(w, r)
			//
			return
		}
	}))
}
