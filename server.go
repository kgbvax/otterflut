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
		for {
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

	log.Printf("handle connection")
	scanner := bufio.NewScanner(conn)
	var s string
	for scanner.Scan() {
		s = scanner.Text()

		if len(s) > 1 && s[0] == 'P' { // we only test for the first "P" on purpose.
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
					"'PX x y rrggb' where x.y = dec integer and colors are hex\n" +
					"'HELP' - this text\n" +
					"'SIZE' - get canvas size ,responds with 'SIZE X Y'\n"
				conn.Write([]byte(HELP))

			}
		}
	}

	// If a failure to read input occurs,
	// it's probably my bad.
	// Fail and figure it out if so!
	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading: %v", err) // TODO don't just kick the bucket
	}
}

func Server() {
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
		case <-sig:
			// Add a leading new line since the signal escape sequence prints on stdout.
			running = false
			srv.Close()
			fmt.Printf("\nShutting down server.\n")
			return
		}
	}
}
