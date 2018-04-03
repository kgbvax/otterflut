package main

import (
	"os"
	"net"
	"log"
)


func main() {
	hostname,_ := os.Hostname()
	log.Printf("my hostname: %v",hostname)
	addrs,_:= net.LookupHost(hostname)
	log.Printf("my ips: %v",addrs)


}
