package main

import (
	"github.com/veandco/go-sdl2/sdl"

	"time"
	"log"
	"strings"

	"flag"
	"os"
	"runtime/pprof"
	"github.com/dustin/go-humanize"
	"io/ioutil"
	"runtime"
	"unsafe"
)

type opix struct {
	//rev order?
	B byte
	G byte
	R byte
	A byte
}

const W = 1440
const H = 900

const PX uint16 = uint16('P'<<8) | uint16('X')

/*const W = 800
const H = 600*/
var ren *sdl.Renderer

var allRect = sdl.Rect{0, 0, W, H}
var lines []string

var pixels = [W * H]opix{}
var running = true

func printFps(frames *uint64) {
	for {
		time.Sleep(time.Second * 1)
		log.Printf("frames=%d\b", *frames)
		*frames = 0
	}
}

func printPixel(pixelcnt *int64) {
	runtime.LockOSThread()
	for {
		time.Sleep(time.Second * 1)
		log.Printf("px/s=%s\b", humanize.Comma(*pixelcnt))
		*pixelcnt = 0
	}
}

//stats
var pixelcnt int64
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

	pixelcnt++

	/*	//sdlcol:=sdl.Color{R: uint8((color & 0xff0000) >> 16),G: uint8((color & 0xff00) >> 8), B: uint8(color & 0xff), A: uint8((color&0xff000000)>>24) }
		gfx.PixelRGBA(ren,int32(x),int32(y),255,255,0,255) */

	pixels[y*W+x] = opix{A: 255, R: byte((color & 0xff0000) >> 16), G: byte((color & 0xff00) >> 8), B: byte(color & 0xff)}
}

//find next 'field' quickly ;-)
func nextNonWs(stri string, initial_start int) (int, int) {
	i := initial_start
	len := len(stri)

	// Skip spaces in the front of the input.
	for i < len && stri[i] == ' ' {
		i++
	}
	start := i

	// now find the end, ie the next space
	for i < len && stri[i] != ' ' {
		i++
	}

	return start, i
}

//lookup table for hex digits
var hexval = [256]uint8{'0': 0, '1': 1, '2': 2, '3': 3, '4': 4, '5': 5,
	'6': 6, '7': 7, '8': 8, '9': 9, 'a': 10, 'A': 10, 'b': 11, 'B': 11, 'c': 12, 'C': 12, 'd': 13, 'D': 13,
	'e': 14, 'E': 14, 'f': 15, 'F': 15}

//quickyla parse a 3 byte hex number
func parseHex3(m string) uint32 {

	//MUL version
	return 0x100000*uint32(hexval[m[0]]) + 0x010000*uint32(hexval[m[1]]) + 0x001000*uint32(hexval[m[2]]) +
		0x000100*uint32(hexval[m[3]]) + 0x000010*uint32(hexval[m[4]]) + uint32(hexval[m[5]])

	//Shift version
	/* return uint32(hexval[m[0]])<<20 + uint32(hexval[m[1]]) <<16 + uint32(hexval[m[2]])<<12 +
		 uint32(hexval[m[3]])<<8 +  uint32(hexval[m[4]])<<4 + uint32(hexval[m[5]])*/

}

//quickly parse a 4 byte hex number
func parseHex4(m string) uint32 {

	return 0x10000000*uint32(hexval[m[0]]) + 0x01000000*uint32(hexval[m[1]]) + 0x00100000*uint32(hexval[m[2]]) +
		0x00010000*uint32(hexval[m[3]]) + 0x00001000*uint32(hexval[m[4]]) + 0x00000100*uint32(hexval[m[5]]) +
		0x00000010*uint32(hexval[m[6]]) + uint32(hexval[m[7]])

}

//quickly parse an uint
//non digit input will lead to 0 as result
/*
func parsUint(m string) uint32 {
	var n uint32
	for _, ch := range []byte(m) {
		ch -= '0'
		if ch > 9 || ch<0{
			return 0
		}
		n = n*10 + uint32(ch)
	}
	return n
}
 */

//no bounds checks we don't care (at this point)
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

func printInfo(sur *sdl.Surface) {
	log.Printf("pixel format: %s\n", sdl.GetPixelFormatName(uint((sur.Format).Format)))
	log.Printf("bytes per pixel: %v\n", (sur.Format).BytesPerPixel)
	log.Printf("bits per pixel: %v\n", (sur.Format).BitsPerPixel)
	log.Printf("size: %v x %v\n", sur.W, sur.H)
	log.Printf("pitch: %v\n", sur.Pitch)
}

func flipper() {
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

	if err := sdl.Init(sdl.INIT_EVENTS | sdl.INIT_TIMER | sdl.INIT_VIDEO); err != nil {
		panic(err)
	}
	defer sdl.Quit()



	window, err := sdl.CreateWindow("test", 0, 0,
		W, H, sdl.WINDOW_SHOWN|sdl.WINDOW_ALLOW_HIGHDPI|sdl.WINDOW_RESIZABLE|sdl.WINDOW_OPENGL)
	checkError(err)
	defer window.Destroy()

	srd, err := sdl.CreateRGBSurfaceWithFormatFrom(unsafe.Pointer(&pixels), W, H, 24, 4*W, sdl.PIXELFORMAT_ARGB8888)
	if err != nil {
		panic(err)
	}
	/*

	ren,err := sdl.CreateRenderer(window,-1,0)

	defer ren.Destroy()
	checkError(err)
	renderer_info,err:=  ren.GetInfo()

	checkError(err)
	ren.Clear()
	log.Printf(" renderer name: %s\b",renderer_info.Name)

	//texture,err := renderer.CreateTexture( sdl.PIXELFORMAT_RGB24, sdl.TEXTUREACCESS_STREAMING, W, H)
	//checkError(err)
	*/

	surface, err := window.GetSurface()
	checkError(err)
	printInfo(surface)
	printInfo(srd)

	for ; running == true; {

		//		var pixeldata []byte= C.GoBytes(unsafe.Pointer(&pixels[0][0]),W*H)
		//	texture.Lock(&allRect)
		//	tex,_:=renderer.CreateTextureFromSurface(srd)
		//	renderer.SetRenderTarget(tex)
		//		texture.Unlock()
		//		ren.Present()
		srd.Blit(nil, surface, nil)

		frames++
		window.UpdateSurface()
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
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
		sdl.Delay(0)
	}
}

func updater() {
	for ; running == true; {
		for _, element := range lines {
			pfparse(element)
		}
		//running=false
	}
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

	go printPixel(&pixelcnt)
	go printFps(&frames)
	go updater()

	flipper()
}
