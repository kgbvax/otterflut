package main

import (
	"log"
	"net"
	"runtime"
	"io"
	"bytes"
	"strings"
	"fmt"
	"os"
	"sync/atomic"
	"os/signal"
	"syscall"
	"github.com/mailru/easygo/netpoll"
	"io/ioutil"
)

var port = "1234"
var connLimit = 1024

const SOCKET_READ_BUFFER_SZ = 1024 * 1024
const SOCKER_READ_CHUNK_SZ = 512 * 1024 // keep in mind that we may need this for thousands of connections

const SINGLE_PIXEL_LL = 18 //PX nnn nnn rrggbb_
const READ_PIXEL_B = 10
const readChunkSize = SINGLE_PIXEL_LL * READ_PIXEL_B

const lockThread = false

// Or as a kind user on reddit refactored:
func checkErr(err error) {
	if err != nil {
		log.Fatal("ERROR:", err)
	}
}

func handlePolledEv(buffer []byte) {
	var messagesProcessedInChunk int64 = 0

	offset := 0
	n := len(buffer)

	for {
		nlAt := bytes.IndexByte(buffer[offset:n], '\n')
		if nlAt > 0 { // -1 not found and 0 (zero length) to be ignored
			//log.Printf("offset: %v, nlAt:%v n:%v",offset,nlAt,n)
			msg := buffer[offset : offset+nlAt] //without NL

			//log.Printf("process >>%v<<", string(msg))
			if len(msg) > 0 {

				if msg[0] == 'P' {
					pfparse(msg)
					messagesProcessedInChunk++

				} else { // not a PX
					/*	s_msg := string(msg)
						s2 := strings.ToLower(s_msg)
						if strings.Contains(s2, "size") {
							_, err := conn.Write([]byte(fmt.Sprintf("SIZE %v %v\n", W, H)))
							//TODO check err
						}
						if strings.Contains(s2, "help") {
							hostname, _ := os.Hostname()
							HELP := "OTTERFLUT (github.com/kgbvax/otterflut) on " + hostname + " (" + runtime.GOARCH + "/" + runtime.Version() +
								")\nCommands:\n" +
								"'PX x y rrggbb' set a pixel, where x,y = decimal postitive integer and colors are hex, hex values need leading zeros\n" +
								"'HELP' - this text\n" +
								"'SIZE' - get canvas size ,responds with 'SIZE X Y'\n" +
								"\nReading pixels is not supported, alpha is not implemented yet\n"
							_, err := conn.Write([]byte(HELP))
							//TODO check err
					*/
				}
			}
			offset += nlAt + 1
		} else { // nothing more
			break
		}
	}
	atomic.AddInt64(&pixelXXCnt, messagesProcessedInChunk)

}

// Handles incoming requests.
// Handles closing of the connection.
func handleConnection(conn *net.TCPConn) {
	if lockThread {
		runtime.LockOSThread()
	} //uh oh, one thread per connection is not that great ;-)

	// Defer all close logic.
	// Using a closure makes it easy to group logic as well as execute serially
	// and avoid the deferred LIFO exec order.
	defer func() {
		// Since handleConnection is run in a go routine,
		// it manages the closing of our net.Conn.
		if lockThread {
			runtime.UnlockOSThread()
		}
		conn.Close()
	}()

	var buffer = make([]byte, SOCKER_READ_CHUNK_SZ)

	for { // forever: read from socket and process contents
		var messagesProcessedInChunk int64

		n, err := conn.Read(buffer)

		//log.Printf("readn %v", n)
		if err != nil {
			//log.Printf("error reading: %v", err)
			if err == io.EOF {
				log.Printf("connection broken")
				return
			}
		} else {
			offset := 0

			for {
				nlAt := bytes.IndexByte(buffer[offset:n], '\n')
				if nlAt > 0 { // -1 not found and 0 (zero length) to be ignored
					//log.Printf("offset: %v, nlAt:%v n:%v",offset,nlAt,n)
					msg := buffer[offset : offset+nlAt] //without NL

					//log.Printf("process >>%v<<", string(msg))
					if len(msg) > 0 {

						if msg[0] == 'P' {
							pfparse(msg)
							messagesProcessedInChunk++

						} else { // not a PX
							s_msg := string(msg)
							s2 := strings.ToLower(s_msg)
							if strings.Contains(s2, "size") {
								_, err = conn.Write([]byte(fmt.Sprintf("SIZE %v %v\n", W, H)))
								//TODO check err
							}
							if strings.Contains(s2, "help") {
								hostname, _ := os.Hostname()
								HELP := "OTTERFLUT (github.com/kgbvax/otterflut) on " + hostname + " (" + runtime.GOARCH + "/" + runtime.Version() +
									")\nCommands:\n" +
									"'PX x y rrggbb' set a pixel, where x,y = decimal postitive integer and colors are hex, hex values need leading zeros\n" +
									"'HELP' - this text\n" +
									"'SIZE' - get canvas size ,responds with 'SIZE X Y'\n" +
									"\nReading pixels is not supported, alpha is not implemented yet\n"
								_, err = conn.Write([]byte(HELP))
								//TODO check err
							}
						}
						offset += nlAt + 1
					} else { // zero length msg
						break
					}
				} else { //no NL found
					break //TODO reshuffle buffer and continue to read MORE

				}
			}
			atomic.AddInt64(&pixelXXCnt, messagesProcessedInChunk)
		}
	}
}

// acceptConns uses the semaphore channel on the counter to rate limit.
// New connections get sent on the returned channel.
func acceptConns(srv *net.TCPListener) <-chan *net.TCPConn {
	conns := make(chan *net.TCPConn, 42)

	go func() {
		for isRunning() {
			conn, err := srv.AcceptTCP()

			if err != nil {
				fmt.Fprintf(os.Stderr, "Error accepting connection: %v\n", err)
				continue
			}
			conn.SetReadBuffer(SOCKET_READ_BUFFER_SZ)
			conn.SetNoDelay(false)
			desc, err := netpoll.Handle(conn, netpoll.EventRead|netpoll.EventEdgeTriggered)
			if err != nil {
				// handle error
			}
			poller, err := netpoll.New(nil)
			if err != nil {
				// handle error
			}

			// Get netpoll descriptor with EventRead|EventEdgeTriggered.
			descriptor := netpoll.Must(netpoll.HandleRead(conn))

			poller.Start(descriptor, func(ev netpoll.Event) {
				if ev&netpoll.EventReadHup != 0 {
					poller.Stop(desc)
					conn.Close()
					return
				}

				buffer, err := ioutil.ReadAll(conn)
				handlePolledEv(buffer)
				if err != nil {
					// handle error
					log.Printf("err %v", err) //TODO "handle"
				}
			})

			conns <- conn
		}
	}()

	return conns
}

func Server(quit chan int) { //todo add mechanism to terminate, channel?
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hostname, _ := os.Hostname()
	log.Printf("my hostname: %v", hostname)
	addrs, _ := net.LookupHost(hostname)
	log.Printf("my ips: %v", addrs)

	service := ":" + port
	tcpAddr, err := net.ResolveTCPAddr("tcp", service)
	checkError(err)

	srv, err := net.ListenTCP("tcp", tcpAddr)
	checkErr(err)

	defer srv.Close()

	// Listen for termination signals.
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)

	// Receive new connections on an unbuffered channel.
	conns := acceptConns(srv)

	for {
		select {
		case conn := <-conns:
			go handleConnection(conn)

		case <-quit:
			log.Print("Server quit.")
			srv.Close()

		case <-sig:
			log.Print("Shutting down server.")
			// Add a leading new line since the signal escape sequence prints on stdout.
			stopRunning()
			srv.Close()

			return
		}
	}
}
