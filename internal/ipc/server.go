package ipc

import (
	"encoding/json"
	"errors"
	"net"
	"os"
	"time"
)

type Request struct {
	Cmd   string `json:"cmd"`
	App   string `json:"app"`
	Name  string `json:"name"`
	Limit int    `json:"limit"`
}

type Response struct {
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

func StartServer(cancelCh <-chan struct{}, socketPath string, handle func(Request) Response) error {
	// remove old socket
	_ = os.Remove(socketPath)

	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		return err
	}

	_ = os.Chmod(socketPath, 0600)

	go func() {
		<-cancelCh
		ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {

			select {
			case <-cancelCh:
				return nil
			default:
				// short pause to not spam errors
				time.Sleep(100 * time.Millisecond)
				// if error is important then return
				// else ignore
				// check "use of closed network connection"
				if errors.Is(err, net.ErrClosed) {
					return nil
				}
				// continue loop
				continue
			}
		}

		go func(c net.Conn) {
			defer c.Close()
			decoder := json.NewDecoder(c)
			encoder := json.NewEncoder(c)

			var req Request
			if err := decoder.Decode(&req); err != nil {
				encoder.Encode(Response{Success: false})
				return
			}
			resp := handle(req)
			_ = encoder.Encode(resp)
		}(conn)
	}
}
