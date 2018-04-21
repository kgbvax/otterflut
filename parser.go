package main

import "sync/atomic"

//lookup table for hex digits
var hexval = [256]uint8{'0': 0, '1': 1, '2': 2, '3': 3, '4': 4, '5': 5,
	'6': 6, '7': 7, '8': 8, '9': 9, 'a': 10, 'A': 10, 'b': 11, 'B': 11, 'c': 12, 'C': 12, 'd': 13, 'D': 13,
	'e': 14, 'E': 14, 'f': 15, 'F': 15}

//quickly  parse a 3 byte hex number
func parseHex3(m []byte) uint32 {
	//MUL version, compiles to shifts
	return 0x100000*uint32(hexval[m[0]]) + 0x010000*uint32(hexval[m[1]]) + 0x001000*uint32(hexval[m[2]]) +
		0x000100*uint32(hexval[m[3]]) + 0x000010*uint32(hexval[m[4]]) + uint32(hexval[m[5]])
}

//quickly parse a 4 byte hex number
func parseHex4(m []byte) uint32 {
	//MUL version
	return 0x10000000*uint32(hexval[m[0]]) + 0x01000000*uint32(hexval[m[1]]) + 0x00100000*uint32(hexval[m[2]]) +
		0x00010000*uint32(hexval[m[3]]) + 0x00001000*uint32(hexval[m[4]]) + 0x00000100*uint32(hexval[m[5]]) +
		0x00000010*uint32(hexval[m[6]]) + uint32(hexval[m[7]])

}

//find next 'field' 'quickly' ;-)
func nextNonWs(stri []byte, initialStart int) (int, int) {
	i := initialStart
	length := len(stri)

	// Skip spaces in the front of the input.
	for ; i < length && stri[i] == ' '; i++ {
	}
	start := i

	// now find the end, ie the next space
	for ; i < length && stri[i] != ' '; i++ {
	}

	return start, i
}


// Swiftly parse an Uint32
// no bounds checks we don't care (at this point)
func parsUint(m []byte) uint32 {
	var n uint32

	/*	l := len(m)
	for i := 0; i < l; i++ {
		n = n*10 + uint32(m[i]-'0')
	} */
	for _, v := range m {
		n = n*10 + uint32(v-'0')
	}
	return n
}

func pfparse(m []byte) {

	var color uint32

	start, end := nextNonWs(m, 3)
	x := parsUint(m[start:end])

	start, end = nextNonWs(m, end)
	y := parsUint(m[start:end])

	start, end = nextNonWs(m, end)
	hexstr := m[start:end]

	if len(hexstr) == 6 {
		color = parseHex3(hexstr)
	} else if len(hexstr) == 8 {
		color = parseHex4(hexstr)
	} else {
		atomic.AddInt64(&errorCnt, 1)
		return
	}
	setPixel(x, y, color)
}
