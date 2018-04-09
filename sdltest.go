package main

import (
	"github.com/veandco/go-sdl2/sdl"
	"time"
	"log"
	"strings"
	"flag"
	"os"
	"runtime/pprof"
	"io/ioutil"
	"runtime"
	"unsafe"
	"github.com/dustin/go-humanize"
)

type opix struct {
	//rev order?
	B byte
	G byte
	R byte
	A byte
}

var W uint32 = 1824
var H uint32 = 968

var lines []string

const numUpdater int = 4

var pixelcnt [numUpdater]int64

var pixels *[]uint32
var running = true
var window *sdl.Window

func printFps(frames *uint64) {
	for {
		time.Sleep(time.Second * 1)
		log.Printf("frames=%d\b", *frames)
		*frames = 0
	}
}

func printPixel() {
	runtime.LockOSThread()
	for  running==true {
		//start:=time.Now()
		time.Sleep(time.Second * 1)
		var total int64
	    for i:=0 ; i< numUpdater; i++ {
	    	total+=pixelcnt[i]
	    	log.Printf("u-%v %v",i,humanize.Comma(pixelcnt[i]))
	    	pixelcnt[i]=0
		}
		log.Printf("total %v",humanize.Comma( total))

	//	pixelPerMsec:= pixelCount / int64(time.Since(start) / time.Millisecond)
	//	log.Printf("px/s=%v",  humanize.Comma(pixelPerMsec*1000))
	//	*pixelcnt = 0
	}
	runtime.UnlockOSThread()
}

//stats

var frames uint64

func checkError(err error) {
	if err != nil {
		log.Fatal(err.Error())
		os.Exit(1)
	}
}

func setPixel(x uint32, y uint32, color uint32) /* chan? */ {
	if x >= W || y >= H {
		return // ignore
	}
 	/*	//sdlcol:=sdl.Color{R: uint8((color & 0xff0000) >> 16),G: uint8((color & 0xff00) >> 8), B: uint8(color & 0xff), A: uint8((color&0xff000000)>>24) }
		gfx.PixelRGBA(ren,int32(x),int32(y),255,255,0,255) */
	(*pixels)[y*W+x] = color //uint32((color & 0xff0000) >> 16) | uint32((color & 0xff00) >> 8) | uint32(color & 0xff)
}

//find next 'field' quickly ;-)
func nextNonWs(stri string, initialStart int) (int, int) {
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

//lookup table for hex digits
var hexval = [256]uint8{'0': 0, '1': 1, '2': 2, '3': 3, '4': 4, '5': 5,
	'6': 6, '7': 7, '8': 8, '9': 9, 'a': 10, 'A': 10, 'b': 11, 'B': 11, 'c': 12, 'C': 12, 'd': 13, 'D': 13,
	'e': 14, 'E': 14, 'f': 15, 'F': 15}

//quickly  parse a 3 byte hex number
func parseHex3(m string) uint32 {
	//MUL version
	return 0x100000*uint32(hexval[m[0]]) + 0x010000*uint32(hexval[m[1]]) + 0x001000*uint32(hexval[m[2]]) +
		0x000100*uint32(hexval[m[3]]) + 0x000010*uint32(hexval[m[4]]) + uint32(hexval[m[5]])
}

//quickly parse a 4 byte hex number
func parseHex4(m string) uint32 {
	//MUL version
	return 0x10000000*uint32(hexval[m[0]]) + 0x01000000*uint32(hexval[m[1]]) + 0x00100000*uint32(hexval[m[2]]) +
		0x00010000*uint32(hexval[m[3]]) + 0x00001000*uint32(hexval[m[4]]) + 0x00000100*uint32(hexval[m[5]]) +
		0x00000010*uint32(hexval[m[6]]) + uint32(hexval[m[7]])

}

// Swiftly parse an Uint32
// no bounds checks we don't care (at this point)
func parsUint(m string) uint32 {
	var n uint32
	l := len(m)
	for i := 0; i < l; i++ {
		n = n*10 + uint32(m[i]-'0')
	}
	return n
}

func pfparse(m string) {
	//elems := strings.Fields(m)

	//0 -> "PX"
	//1&2 -> x & y (dec)
	//3 -> Color(hex)
	if m[0] == 'P' { // we only test for the first "P" on purpose.

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
			//huh?
			return
		}
		setPixel(x, y, color)
	} //else TODO
}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")

func printInfo(sur *sdl.Surface, name string) {
	log.Print("foo")
	var formatName = "-"
	if sur == nil {
		log.Print("surface is nil")
		return
	}
	imgFormat := sur.Format

	if imgFormat != nil {
		format := uint((sur.Format).Format)
		formatName = sdl.GetPixelFormatName(format)
	} else {
		formatName = "Format is nil"
	}
	log.Printf("%v pixel format: %s\n", name, formatName)
	log.Printf("%v bytes per pixel: %v\n", name, (sur.Format).BytesPerPixel)
	log.Printf("%v bits per pixel: %v\n", name, (sur.Format).BitsPerPixel)
	log.Printf("%v size: %v x %v\n", name, sur.W, sur.H)
	log.Printf("%v pitch: %v\n", name, sur.Pitch)
}

func windowInit() {
	var err error

	sdl.SetHint("SDL_HINT_FRAMEBUFFER_ACCELERATION", "1")

	numdrv, _ := sdl.GetNumRenderDrivers()
	for i := 0; i < numdrv; i++ {
		var rinfo sdl.RendererInfo
		sdl.GetRenderDriverInfo(i, &rinfo)
		name := rinfo.Name
		log.Printf("available rendere: %v", name)
		if name == "metal" {
			log.Print("🤘!")
			sdl.SetHint("SDL_HINT_RENDER_DRIVER", "metal")
		}
	}

	if err = sdl.Init(sdl.INIT_EVENTS | sdl.INIT_TIMER | sdl.INIT_VIDEO); err != nil {
		panic(err)
	}
	displayBounds,err :=sdl.GetDisplayBounds(0)
	checkError(err)
	log.Printf("display: %v * %v",displayBounds.W, displayBounds.H)

	window, err = sdl.CreateWindow("otterflut", 0, 0,
		displayBounds.W, displayBounds.H, sdl.WINDOW_SHOWN|sdl.WINDOW_ALLOW_HIGHDPI|sdl.WINDOW_BORDERLESS|sdl.WINDOW_OPENGL)
	checkError(err)


	surface, err := window.GetSurface()
	checkError(err)

	printInfo(surface, "window")

	W = uint32(surface.W)
	H = uint32(surface.H)

	pixelsPtr := uintptr(surface.Data())
	pixelsSlice := struct {
		addr uintptr
		len  int
		cap  int
	}{pixelsPtr, int(W * H * 4), int(W * H * 4)}
	pixels = (*[]uint32)(unsafe.Pointer(&pixelsSlice))

}

func updateWin() {
	frames++
	window.UpdateSurface()
}

func sdlEventLoop() {
	for running==true {
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			//log.Print(event)
			switch event.(type) {
			case *sdl.QuitEvent:
				println("Quit")
				running = false
				if *memprofile != "" {
					f, err := os.Create(*memprofile)
					if err != nil {
						log.Fatal("could not create memory profile: ", err)
					}
					runtime.GC() // get up-to-date statistics
					if err := pprof.WriteHeapProfile(f); err != nil {
						log.Fatal("could not write memory profile: ", err)
					}
					f.Close()
				}
				break
			}
		}
	}
}

func updater(gridx int) {
	runtime.LockOSThread()
	for pixels==nil { //wait for bitmap to become available
		runtime.Gosched()
	}

	for ; running == true; {
		for _, element := range lines {
			pfparse(element)
			pixelcnt[gridx]++
		}
		//running=false
	}
	runtime.UnlockOSThread()
}

func main() {
	runtime.GOMAXPROCS(4 + runtime.NumCPU())


	bdata, err := ioutil.ReadFile("test.pxfl")
	checkError(err)
	s := string(bdata)
	lines = strings.Split(s, "\n")
	//log.Println(http.ListenAndServe("localhost:6060", nil))

	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}

		defer pprof.StopCPUProfile()
	}
	windowInit()
	go printPixel()
	go printFps(&frames)


	ticker := time.NewTicker(1000 / 30 * time.Millisecond) //target 30fps
	go func() {
		for  range ticker.C {
			 updateWin()
		}
	}()

	for i:=0 ; i< numUpdater; i++ {
		go updater(i)
	}


	sdlEventLoop()

}
