package main

/*
#cgo CFLAGS: -I CL
#cgo arm64  LDFLAGS: -lOpenCL -L/usr/lib/aarch64-linux-gnu
#cgo darwin LDFLAGS: -framework OpenC
*/

import "gocl/cl"

const maxMsgLen = 32

func init() {

	// Host data
 	var A [][maxMsgLen]cl.CL_char //input array
	var X []cl.CL_int //output array
	var Y []cl.CL_int //output array
	var C []cl.CL_long // output array


}
