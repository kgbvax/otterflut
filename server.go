package main

import (
	"os"
	"net"
	"log"
	"os/signal"
	"syscall"
	"fmt"
	"bufio"
	"time"
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
func acceptConns(srv net.Listener, counter *Counter) <-chan net.Conn {
	conns := make(chan net.Conn)

	go func() {
		for {
			conn, err := srv.Accept()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error accepting connection: %v\n", err)
				continue
			}

			select {
			case counter.Sem <- 1:
				conns <- conn
			default:
				fmt.Fprintf(conn, "Server busy.")
				conn.Close()
			}
		}
	}()

	return conns
}


// Handles incoming requests.
// Handles closing of the connection.
func handleConnection(conn net.Conn, counter *Counter) {
	// Defer all close logic.
	// Using a closure makes it easy to group logic as well as execute serially
	// and avoid the deferred LIFO exec order.
	defer func() {
		// Since handleConnection is run in a go routine,
		// it manages the closing of our net.Conn.
		conn.Close()
		// Once our connection is closed,
		// we can drain a value from our semaphore
		// to free up a space in the connection limit.
		<-counter.Sem
	}()

	scanner := bufio.NewScanner(conn)
	var s string
	for scanner.Scan() {
		s = scanner.Text()
		log.Printf(s)

		/* From here on out, we have a valid input. */
		// Safely increment total counter.
		counter.Inc()

	}

	// If a failure to read input occurs,
	// it's probably my bad.
	// Fail and figure it out if so!
	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading: %v", err)
	}
}

func Server() {
	hostname,_ := os.Hostname()
	log.Printf("my hostname: %v",hostname)
	addrs,_:= net.LookupHost(hostname)
	log.Printf("my ips: %v",addrs)

	addr:=addrs[0] //TODO

	srv, err := net.Listen("tcp", addr+":"+port)
	checkErr(err)
	defer srv.Close()

	counter := NewCounter(connLimit)

	// Listen for termination signals.
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)

	// Set up intervals
	go counter.RunOutputInterval(5*time.Second)

	// Receive new connections on an unbuffered channel.
	conns := acceptConns(srv, counter)
	for {
		select {
		case conn := <-conns:
			go handleConnection(conn, counter)
		case <-sig:
			// Add a leading new line since the signal escape sequence prints on stdout.
			fmt.Printf("\nShutting down server.\n")
			counter.Close()
			os.Exit(0)
		}
	}
}
