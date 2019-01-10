package main

import "net"

func main() {
	newConns := make(chan net.Conn, 128)
	deadConns := make(chan net.Conn, 128)
	publishes := make(chan []byte, 128)
	conns := make(map[net.Conn]bool)
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				panic(err)
			}
			newConns <- conn
		}
	}()
	for {
		select {
		case conn := <-newConns:
			conns[conn] = true
			go func() {
				buf := make([]byte, 1024)
				for {
					nbyte, err := conn.Read(buf)
					if err != nil {
						deadConns <- conn
						break
					} else {
						fragment := make([]byte, nbyte)
						copy(fragment, buf[:nbyte])
						publishes <- fragment
					}
				}
			}()
		case deadConn := <-deadConns:
			_ = deadConn.Close()
			delete(conns, deadConn)
		case publish := <-publishes:
			for conn, _ := range conns {
				go func(conn net.Conn) {
					totalWritten := 0
					for totalWritten < len(publish) {
						writtenThisCall, err := conn.Write(publish[totalWritten:])
						if err != nil {
							deadConns <- conn
							break
						}
						totalWritten += writtenThisCall
					}
				}(conn)
			}
		}
	}
	listener.Close()
}
