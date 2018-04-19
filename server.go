package main

import (
	"os"
	"net"
	"log"
	"os/signal"
	"syscall"
	"fmt"
	"bufio"
	"strings"
	"runtime"
)

var port string = "1234"
var connLimit int = 1024

const SINGLE_PIXEL_LL = 18 //PX nnn nnn rrggbb_
const READ_PIXEL_B = 10
const   readChunkSize = SINGLE_PIXEL_LL * READ_PIXEL_B


// Or as a kind user on reddit refactored:
func checkErr(err error) {
	if err != nil {
		log.Fatal("ERROR:", err)
	}
}

// acceptConns uses the semaphore channel on the counter to rate limit.
// New connections get sent on the returned channel.
func acceptConns(srv net.Listener) <-chan net.Conn {
	conns := make(chan net.Conn)

	go func() {
		for isRunning() {

			conn, err := srv.Accept()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error accepting connection: %v\n", err)
				continue
			}

			conns <- conn
		}
	}()

	return conns
}

func xScan() {

}
// Handles incoming requests.
// Handles closing of the connection.
func handleConnection(conn net.Conn) {
	// Defer all close logic.
	// Using a closure makes it easy to group logic as well as execute serially
	// and avoid the deferred LIFO exec order.
	defer func() {
		// Since handleConnection is run in a go routine,
		// it manages the closing of our net.Conn.
		conn.Close()
	}()

	buffered:=bufio.NewReaderSize(conn,8192)

	scanner := bufio.NewScanner(buffered)
	var s string

	for scanner.Scan() { //TODO this most likely needs tuning
		s = scanner.Text()

		if len(s) > 1 && //we can save the len test if this becomes a gurantee of the "scanning"
			s[0] == 'P' { // we only test for the first "P" on purpose.
			pfparse(s)
			//log.Printf("parsed %v",s)
		} else {
			s2 := strings.ToLower(s)
			if strings.Contains(s2, "size") {
				conn.Write([]byte(fmt.Sprintf("SIZE %v %v\n", W, H)))
			}
			if strings.Contains(s, "help") {
				hostname, _ := os.Hostname()
				HELP := "OTTERFLUT (github.com/kgbvax/otterflut) on " + hostname + " (" + runtime.GOARCH + "/" + runtime.Version() +
					")\nCommands:\n" +
					"'PX x y rrggbb' set a pixel, where x,y = decimal postitive integer and colors are hex, hex values need leading zeros\n" +
					"'HELP' - this text\n" +
					"'SIZE' - get canvas size ,responds with 'SIZE X Y'\n"+
					"\nReading pixels is not supported, alpha is not implemented yet\n"
				conn.Write([]byte(HELP))

			}
		}
	}

	// If a failure to read input occurs,
	// it's probably my bad.
	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading: %v", err) // TODO don't just kick the bucket
	}
}

func Server(quit chan int) { //todo add mechanism to terminate, channel?
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hostname, _ := os.Hostname()
	log.Printf("my hostname: %v", hostname)
	addrs, _ := net.LookupHost(hostname)
	log.Printf("my ips: %v", addrs)

	srv, err := net.Listen("tcp", ":"+port)
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
