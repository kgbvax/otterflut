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
)

var port = "1234"
var connLimit = 1024
var totalBytes int64

const socketReadBufferSz = 256 * 1024
const socketReadChunkSz = 16 * 1024 // keep in mind that we may need this for thousands of connections

const SINGLE_PIXEL_LL = 18 //PX nnn nnn rrggbb_
const READ_PIXEL_B = 10
const readChunkSize = SINGLE_PIXEL_LL * READ_PIXEL_B

const lockThread = false
const dontProcessPX =true  //for testing purposes
const usePoll=false

// Or as a kind user on reddit refactored:
func checkErr(err error) {
	if err != nil {
		log.Fatal("ERROR:", err)
	}
}

func handleBuffer(buffer []byte, conn *net.TCPConn) {
//	log.Printf("handle buffer")
	n := len(buffer)
	totalBytes+=int64(n)
	offset := 0
	var messagesProcessedInChunk int64 = 0

	for {
		nlAt := bytes.IndexByte(buffer[offset:n], '\n')
		if nlAt > 0 { // -1 not found and 0 (zero length) to be ignored
			//log.Printf("offset: %v, nlAt:%v n:%v",offset,nlAt,n)
			msg := buffer[offset : offset+nlAt] //without NL

			//log.Printf("process >>%v<<", string(msg))
			if len(msg) > 0 {

				if msg[0] == 'P' {
					if dontProcessPX {
						pfparse(msg)
					}
					messagesProcessedInChunk++

				} else { // not a PX
					s_msg := string(msg)
					s2 := strings.ToLower(s_msg)
					if strings.Contains(s2, "size") {
						_, _ = conn.Write([]byte(fmt.Sprintf("SIZE %v %v\n", W, H)))
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
						_, _ = conn.Write([]byte(HELP))
						//TODO check err

					}
				}
				offset += nlAt + 1
			} else { // nothing more
				break
			}
		} else {
			break
		}

	}
	atomic.AddInt64(&pixelXXCnt, messagesProcessedInChunk)

}

func handlePolledEv(conn *net.TCPConn) {

	var buffer = make([]byte, socketReadChunkSz)
	_, err := conn.Read(buffer)
	if err == nil {
		handleBuffer(buffer, conn)
	} else {
		log.Printf("error reading %v", err)
	}
}

// Handles incoming requests.
// Handles closing of the connection.
func handleXXXConnection(conn *net.TCPConn) {
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

	var buffer = make([]byte, socketReadChunkSz)

	for { // forever: read from socket and process contents

		n, err := conn.Read(buffer)

		//log.Printf("readn %v", n)
		if err != nil {
			//log.Printf("error reading: %v", err)
			if err == io.EOF {
				log.Printf("connection broken")
				return
			}
		} else {
			handleBuffer(buffer,conn)
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
			conn.SetReadBuffer(socketReadBufferSz)
			conn.SetNoDelay(false)
			if usePoll {
				desc, err := netpoll.Handle(conn, netpoll.EventRead|netpoll.EventEdgeTriggered)
				if err != nil {
					// handle error
					log.Printf("poll error %v", err)
				}

				poller, err := netpoll.New(nil)
				if err != nil {
					// handle error
					log.Printf("poll error %v", err)
				}

				// Get netpoll descriptor with #|EventEdgeTriggered.
				//descriptor := netpoll.Must(netpoll.HandleRead(conn))
				log.Printf("poller start, %v", desc)
				poller.Start(desc,
					func(ev netpoll.Event) {
						if ev&netpoll.EventReadHup != 0 {
							poller.Stop(desc)
							conn.Close()
							return
						}

						handlePolledEv(conn)
						if err != nil {
							// handle error
							log.Printf("err %v", err) //TODO "handle"
						}
					})
			} else {
				go handleXXXConnection(conn)
			}
			conns <- conn
		}
	}()

	return conns
}

func findMyIp() string {
	addrs, err := net.InterfaceAddrs()
	addresses := ""

	if err != nil {
		panic(err)
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && !ipnet.IP.IsLinkLocalUnicast() { // todo filter v6 temporary
			addresses += " " + ipnet.IP.String()
		}
	}
	return addresses
}

func Server(quit chan int) { //todo add mechanism to terminate, channel?
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hostname, _ := os.Hostname()
	log.Printf("my hostname: %v", hostname)

	log.Printf("my ips: %v", findMyIp())

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
	_ = acceptConns(srv)

	for {
		select {
		/*case conn := <-conns:
			go handleConnection(conn) */

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
