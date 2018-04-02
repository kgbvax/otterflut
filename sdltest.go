package main

import (
	"github.com/veandco/go-sdl2/sdl"
	"unsafe"
	"time"
	"log"
	"strings"
	"strconv"
	"flag"
	"os"
	"runtime/pprof"
	"github.com/dustin/go-humanize"
	"io/ioutil"
)

type opix struct {
	A byte
	R byte
	G byte
	B byte
}

/*const W = 1920
const H = 1080*/
const W = 800
const H = 600


var allRect = sdl.Rect{0, 0, W, H}
var lines []string

var pixels = [W][H]opix{}
var running = true

func printFps(frames *uint32) {
	for {
		time.Sleep(time.Second * 1)
		log.Printf("frames=%d\b", *frames)
		*frames = 0
	}
}
func printPixel(pixelcnt *int64) {
	for {
		time.Sleep(time.Second * 1)
		log.Printf("px/s=%s\b", humanize.Comma(*pixelcnt))
		*pixelcnt = 0
	}
}

//stats
var pixelcnt int64
var frames int64

func checkError(err error) {
	if err != nil {
		log.Fatal(err.Error())
		os.Exit(1)
	}
}


func setPixel(x uint32, y uint32, color uint32) {
	pixelcnt++
	pix := opix{255,byte((color & 0xff0000) >> 16), byte((color & 0xff00) >> 8), byte((color & 0xff))}
	if x >= W || y >= H {
		return // ignore
	}
/*	println("--")
	println(x)
	println(y)
	println(pix.R)
	println(pix.G)
	println(pix.B) */
	pixels[x][y] = pix
}

func pfparse(m string) {
	elems := strings.Split(m, " ")

	//0 -> "PX"
	//1&2 -> x & y (dec)
	//3 -> Color(hex)
	x, err := strconv.Atoi(elems[1])
	checkError(err)
	y, err := strconv.Atoi(elems[2])
	checkError(err)
	color, err := strconv.ParseUint(elems[3], 16, 32)
	checkError(err)
	setPixel(uint32(x), uint32(y), uint32(color))

}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")

func flipper() {
	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}
	defer sdl.Quit()

	srd, err := sdl.CreateRGBSurfaceWithFormatFrom(unsafe.Pointer(&pixels), W, H, 32, W*3, sdl.PIXELFORMAT_ARGB8888)
	if err != nil {
		panic(err)
	}

	window, err := sdl.CreateWindow("test", 0, 0,
		W, H, sdl.WINDOW_SHOWN|sdl.WINDOW_ALLOW_HIGHDPI)
	if err != nil {
		panic(err)
	}
	defer window.Destroy()

	surface, err := window.GetSurface()

	sur:=srd

	log.Printf(" pixel format: %s\b",sdl.GetPixelFormatName(uint((sur.Format).Format)))
	log.Printf("bytes per pixel: %v\b",(sur.Format).BytesPerPixel)
	log.Printf("bits per pixel: %v\b",(sur.Format).BitsPerPixel)
	log.Printf("pitch: %v\b",sur.Pitch)
	for ; running == true; {


		if err != nil {
			panic(err)
		}
		srd.Blit(&allRect, surface, &allRect)

		window.UpdateSurface()
		frames++

		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch event.(type) {
			case *sdl.QuitEvent:
				println("Quit")
				running = false
				break
			}
		}
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

	bdata, err := ioutil.ReadFile("small.pxfl")
	checkError(err)
	s := string(bdata)
	lines = strings.Split(s, "\n")

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
	go updater()

	flipper()
}
