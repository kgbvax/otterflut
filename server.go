package main

import (
	"os"
	"net"
	"log"
	"os/signal"
	"syscall"
	"fmt"

	"runtime"
	"strings"
	"io"
)

var port = "1234"
var connLimit = 1024

const SOCKET_READ_BUFFER_SZ = 1024 *1024
const SOCKER_READ_CHUNK_SZ  = 512  *1024 // keep in mind that we may need this for thousands of connections

const SINGLE_PIXEL_LL = 18 //PX nnn nnn rrggbb_
const READ_PIXEL_B = 10
const readChunkSize = SINGLE_PIXEL_LL * READ_PIXEL_B

// Or as a kind user on reddit refactored:
func checkErr(err error) {
	if err != nil {
		log.Fatal("ERROR:", err)
	}
}




//reading gameplan
//read into buffer up to X
//traverse buffer looking for \n
//found: print slice
//  continue scanning
//end reached and not \n:  copy "remains" to beginning of buffer (<- sucks) and read again
// continue scanning: if no \n found: pathological, discard
//
//options:
// * can we read into a ring-buffer, would save copies?
// * are "bytes" faster than strings?

/* scan slice for \n, returns index of \n or -1 if not found */
func findNl(buf []byte) int {
	for i, v := range buf {  //classic "for" loop instead of range is not faster
		if v == '\n' {
			return i
		}
	}
	return -1
}

// Handles incoming requests.
// Handles closing of the connection.
func handleConnection(conn *net.TCPConn) {
	runtime.LockOSThread() //uh oh, one thread per connection is not that great ;-)


	// Defer all close logic.
	// Using a closure makes it easy to group logic as well as execute serially
	// and avoid the deferred LIFO exec order.
	defer func() {
		// Since handleConnection is run in a go routine,
		// it manages the closing of our net.Conn.
		runtime.UnlockOSThread()
		conn.Close()
	}()

	var buffer = make([]byte, SOCKER_READ_CHUNK_SZ)

	for { //TODO this most likely needs tuning


		n, err := conn.Read(buffer)

		//log.Printf("readn %v", n)
		if err != nil {
			log.Printf("error reading: %v", err)
			if err == io.EOF {
				log.Printf("connection broken")
				return
			}
		} else {
			offset := 0

			for {
				nlAt := findNl(buffer[offset:n]) //search in slice from (last) start to
				if nlAt > 0 { // -1 not found and 0 (zero length) to be ignored
					//log.Printf("offset: %v, nlAt:%v n:%v",offset,nlAt,n)
					msg := buffer[offset : offset+nlAt]  //without NL


					//log.Printf("process >>%v<<", string(msg))
					if len(msg) > 0 {
						if msg[0] == 'P' {
							 pfparse(msg)
						} else {
							s_msg:=string(msg)
							s2 := strings.ToLower(s_msg)
							if strings.Contains(s2, "size") {
								_,err= conn.Write([]byte(fmt.Sprintf("SIZE %v %v\n", W, H)))
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
								_,err=conn.Write([]byte(HELP))
								//TODO check err
							}
						}
						offset += nlAt + 1
					} else {  // zero length msg
						break
					}
				} else { //no NL found
					break //TODO reshuffle buffer and continue to read MORE
				}
			}
		}
	}
}


// acceptConns uses the semaphore channel on the counter to rate limit.
// New connections get sent on the returned channel.
func acceptConns(srv *net.TCPListener) <-chan *net.TCPConn {
	conns := make(chan *net.TCPConn,42)

	go func() {
		for isRunning() {
			conn, err := srv.AcceptTCP()

			if err != nil {
				fmt.Fprintf(os.Stderr, "Error accepting connection: %v\n", err)
				continue
			}
			conn.SetReadBuffer(SOCKET_READ_BUFFER_SZ)
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

	service := ":"+port
	tcpAddr, err := net.ResolveTCPAddr("tcp", service)
	checkError(err)

	srv,err := net.ListenTCP("tcp",tcpAddr)
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
