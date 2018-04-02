package main

import (
	"github.com/veandco/go-sdl2/sdl"

	"time"
	"log"
	"strings"
	"strconv"
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
/*const W = 800
const H = 600*/
var  ren *sdl.Renderer


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
	pix := opix{A:255, R:byte((color & 0xff0000) >> 16), G:byte((color & 0xff00) >> 8),B: byte((color & 0xff))}

	pixels[y*W+x] = pix

}

var asciiSpace = [256]uint8{'\t': 1, '\n': 1, '\v': 1, '\f': 1, '\r': 1, ' ': 1}



func nextNonWs(stri string, initial_start int) (start int,end int ) {
	i := initial_start
	len:=len(stri)
	// Skip spaces in the front of the input.
	for i < len && asciiSpace[stri[i]] != 0 {
		i++
	}
	start=i

	for i < len {
		if asciiSpace[stri[i]] == 0 {
			i++
			continue
		} else {
			break
		}
	}
	return start,i
}

func pfparse(m string) {
	 //elems := strings.Fields(m)

	//0 -> "PX"
	//1&2 -> x & y (dec)
	//3 -> Color(hex)
	if m[0]=='P' && m[1]=='X' && m[2]==' ' {
		var x,y  int
		var color uint64
		var err error
		var start int=3
		var end int

		start=3

		start,end = nextNonWs(m,start)
		if x, err = strconv.Atoi(m[start:end]); err != nil {
		//	log.Print(err)
			return
		}

		start,end = nextNonWs(m,end)
		if y, err = strconv.Atoi(m[start:end]); err != nil {
		//	log.Print(err)
			return
		}


		start,end = nextNonWs(m,end)
		if color, err = strconv.ParseUint(m[start:end], 16, 32); err !=nil {
			return
		}
 		setPixel(uint32(x), uint32(y), uint32(color))
	}

}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")

func printInfo(sur *sdl.Surface) {
	log.Printf("pixel format: %s\n",sdl.GetPixelFormatName(uint((sur.Format).Format)))
	log.Printf("bytes per pixel: %v\n",(sur.Format).BytesPerPixel)
	log.Printf("bits per pixel: %v\n",(sur.Format).BitsPerPixel)
	log.Printf("size: %v x %v\n",sur.W,sur.H)
	log.Printf("pitch: %v\n",sur.Pitch)
}

func flipper() {
	if err := sdl.Init(sdl.INIT_EVENTS|sdl.INIT_TIMER); err != nil {
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
		srd.Blit(&allRect,surface,&allRect)
		frames++
		window.UpdateSurface()
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch event.(type) {
			case *sdl.QuitEvent:
				println("Quit")
				running = false
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
	runtime.GOMAXPROCS(4+runtime.NumCPU())

	bdata, err := ioutil.ReadFile("test.pxfl")
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
	go printFps(&frames)
	go updater()

	flipper()
}
