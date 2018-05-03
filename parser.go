package main

import (
	"sync/atomic"
)

type parser interface {
	 pfparse(m []byte)
}

//lookup table for hex digits
var hexval32 = [256]uint32{'0': 0, '1': 1, '2': 2, '3': 3, '4': 4, '5': 5,
	'6': 6, '7': 7, '8': 8, '9': 9, 'a': 0xA, 'A': 0xA, 'b': 0xB, 'B': 0xB, 'c': 0xC, 'C': 0xC, 'd': 0xD, 'D': 0xD,
	'e': 0xE, 'E': 0xE, 'f': 0xF, 'F': 0xF}

//quickly  parse a 3 byte hex number
/*
func parseHex3(m []byte) uint32 {
	//MUL version, compiles to shifts
	return 0x100000*hexval32[m[0]] + 0x010000*hexval32[m[1]] + 0x001000*hexval32[m[2]] +
		0x000100*hexval32[m[3]] + 0x000010*hexval32[m[4]] + hexval32[m[5]]
} */

//quickly  parse a 3 byte hex number
//RGB to BGR included

func parseHex3ToBGR(m []byte) uint32 {
	//MUL version, compiles to shifts
	// BB GG RR
	return 0x100000*hexval32[m[4]] + 0x010000*hexval32[m[5]] +
		0x001000*hexval32[m[2]] + 0x000100*hexval32[m[3]] +
		0x000010*hexval32[m[0]] + hexval32[m[1]]
}

//quickly parse a 4 byte hex number
func parseHex4(m []byte) uint32 {
	//MUL version
	return 0x10000000*hexval32[m[0]] + 0x01000000*hexval32[m[1]] + 0x00100000*hexval32[m[2]] +
		0x00010000*hexval32[m[3]] + 0x00001000*hexval32[m[4]] + 0x00000100*hexval32[m[5]] +
		0x00000010*hexval32[m[6]] + hexval32[m[7]]

}

//find next 'field' 'quickly' ;-)
func nextNonWs(stri []byte, initialStart int) (int, int) {

	length := len(stri)
	const SPACE byte = ' '
	i:=initialStart

	// Skip spaces in the front of the input.
	for ; i < length && stri[i] == SPACE; i++ {
	}
	start := i

	// now find the end, ie the next space
	for ; i < length && stri[i] != SPACE; i++ {
	}

	return start, i
}

// Parse an Uint
// no bounds checks we don't care (at this point), works for 0..9999
// loop-less edition, mucho rapido
func parsUint(m []byte) uint32 {
	l := len(m)
	switch l {
	case 3: //assumed to be the  most likely case
		return uint32(
			100*int(m[0]-'0') +
				10*int(m[1]-'0') +
				int(m[2]-'0'))
	case 4:
		return uint32(
			1000*int(m[0]-'0') +
				100*int(m[1]-'0') +
				10*int(m[2]-'0') +
				int(m[3]-'0'))
	case 2:
		return uint32(
			10*int(m[0]-'0') +
				int(m[1]-'0'))
	case 1:  //least likely case
		return uint32(m[0] - '0')
	}

	//todo increment error count
	return 0
}

//Parse & performf a "PX" line
func pfparse(m []byte) {

	var color uint32


	start, end := nextNonWs(m, 3)
	x := parsUint(m[start:end])
	//log.Printf("e1: %v %v %v",string(m[start:end]),start,end)

	start, end = nextNonWs(m, end)
	y := parsUint(m[start:end])
	//log.Printf("e2: %v %v %v",string(m[start:end]),start,end)

	start, end = nextNonWs(m, end)
	//log.Printf("c: %v %v %v",string(m[start:end]),start,end)

	hexstr := m[start:end]
	switch len(hexstr) {
		case 6:
			color = parseHex3ToBGR(hexstr)
		case 8:
			color = parseHex4(hexstr)
		default:
			wrongLen:=len(hexstr)
			if wrongLen <6 { //to short, could try to fix this by padding with leading 0s
				//TODO maybe later or maybe never
			}
			//log.Printf("pfparse err >%v< >%v<",string(m),hexstr)
			atomic.AddInt64(&errorCnt, 1)
			return
	}

	setPixel(x, y, color)
}
