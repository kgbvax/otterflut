package main

import (
	"flag"
	"github.com/dustin/go-humanize"
	"github.com/veandco/go-sdl2/sdl"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"sync/atomic"
	"time"
	"unsafe"
	"io/ioutil"
	"strings"
	"math/rand"
	_ "net/http/pprof"
	"strconv"
)

type opix struct {
	//rev order?
	B byte
	G byte
	R byte
	A byte
}

//uint to save on sign manipulation in hot loop
var W uint32 = 800
var H uint32 = 600

var lines []string


const numSimUpdater int = 4

var pixelCntSli [numSimUpdater]int64

var pixelXXCnt int64
var totalPixelCnt int64

var pixels *[]uint32
var pixelsArr []byte
var sdlTexture *sdl.Texture
var renderer *sdl.Renderer

var xrunning bool = true
var window *sdl.Window=nil

var frames uint64
var errorCnt int64

var serverQuit chan int = make(chan int)

func printFps() {

	for isRunning() {
		time.Sleep(time.Second * 1)
		log.Printf("frames=%v\b", atomic.LoadUint64(&frames))

		atomic.StoreUint64(&frames, 0)
	}
	log.Print("Exit printFps")
}

func printPixel() {
	runtime.LockOSThread()
	for isRunning() {
		time.Sleep(time.Second * 1)
		var sumPixelCount int64
		for i:=0 ;i<numSimUpdater;i++ {
			sumPixelCount+=pixelCntSli[i]
			pixelCntSli[i]=0
		}
		//pixelCount := atomic.LoadInt64(&sumPixelCount)
		log.Printf("%v", humanize.Comma(sumPixelCount))

		//atomic.StoreInt64(&pixelCnt, 0)
		atomic.AddInt64(&totalPixelCnt, sumPixelCount)
	}
	runtime.UnlockOSThread()
}

//stats

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
	//MUL version, compiles to shifts
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

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")

func printSurfaceInfo(sur *sdl.Surface, name string) {
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

func updateWin() {
	atomic.AddUint64(&frames, 1)
	//window.UpdateSurface()
	sdlTexture.Unlock()

	sdlTexture.Update(nil,pixelsArr,int(W*4))
	renderer.Clear()
	renderer.Copy(sdlTexture,nil,nil)
	renderer.Present()
	window.UpdateSurface()
	sdlTexture.Lock(nil)

}

func windowInit() {
	var err error

	platform := sdl.GetPlatform()
	log.Printf("platform: %v",platform)
	switch platform {
	case "Mac OS X":
	    sdl.SetHint("SDL_HINT_FRAMEBUFFER_ACCELERATION", "metal")
	    sdl.SetHint("SDL_HINT_RENDER_DRIVER", "metal") //this fails on older OSX versions, I don't care

	case "Linux":
		//OpenGLES2
		//sdl.GLSetAttribute(sdl.GL_CONTEXT_PROFILE_MASK,sdl.GL_CONTEXT_PROFILE_ES)
		//sdl.GLSetAttribute(sdl.GL_CONTEXT_MAJOR_VERSION,2)
		//sdl.GLSetAttribute(sdl.GL_CONTEXT_MINOR_VERSION,0)
	}



	numdrv, _ := sdl.GetNumRenderDrivers()
	for i := 0; i < numdrv; i++ {
		var rinfo sdl.RendererInfo
		sdl.GetRenderDriverInfo(i, &rinfo)
		name := rinfo.Name
		log.Printf("available renderer driver: %v", name)
	}

	if err = sdl.Init(sdl.INIT_VIDEO); err != nil {
		panic(err)
	}
	checkSdlError()

	//sdl.DisableScreenSaver()

	//checkSdlError()

	displayBounds, err := sdl.GetDisplayBounds(0)
	checkErr(err)
	log.Printf("display: %vx%v", displayBounds.W, displayBounds.H)

	W=uint32(displayBounds.W)
	H=uint32(displayBounds.H)


	window, err = sdl.CreateWindow("otterflut", 0, 0,int32(W),int32(H),
		sdl.WINDOW_SHOWN|sdl.WINDOW_ALLOW_HIGHDPI|sdl.WINDOW_BORDERLESS )


	log.Print("create renderer")
	renderer,err = sdl.CreateRenderer(window,-1,sdl.RENDERER_PRESENTVSYNC|sdl.RENDERER_ACCELERATED)
	checkErr(err)
	checkSdlError()

	info,err:=renderer.GetInfo()
	log.Printf("selected renderer: %v",info.Name)
	log.Printf("max texgure size: %vx%v",info.MaxTextureWidth,info.MaxTextureHeight)


	log.Print("create texture")
	sdlTexture,err = renderer.CreateTexture(
		sdl.PIXELFORMAT_ARGB8888,
		sdl.TEXTUREACCESS_TARGET,
		int32(W), int32(H))
    checkErr(err)
	checkSdlError()
	sdlTexture.Lock(nil)

	pixelsArr = make ([]byte,W*H*4) //the actual pixel buffer hidden in a golang array
	pixels=(*[]uint32)(unsafe.Pointer(&pixelsArr))

}

func isRunning() bool {
	return xrunning
}

func stopRunning() {
	if isRunning() {
		xrunning = false
		serverQuit <- 1
		if window != nil {
			log.Print("window destroy")
			window.Destroy()
		}
	}
}



func sdlEventLoop() {
	for isRunning() {
		event := sdl.WaitEventTimeout(20)
		if event!= nil {
			//log.Print(event)
			switch event.(type) {
			case *sdl.QuitEvent:
				log.Print("SDL Quit")
				stopRunning()
				return
			}
		} else {
			error:=sdl.GetError()

			if error!=nil {
				log.Printf("sdl-event-loop: ",error)
				sdl.ClearError()
			}
		}
	}
}

func updateSim(gridx int) {
	for pixels == nil { //wait for bitmap to become available
		runtime.Gosched()
	}

	for isRunning() {
		for _, element := range lines {
			pfparse(element)
			pixelCntSli[gridx]+=1
		}
		time.Sleep(time.Duration(rand.Int63n(10)) * time.Millisecond) // some random delay
	}
	log.Printf("Exit updateSim %v", gridx)

}

//take 10 memory profiles every 5 seconds
func memProfileWriter() {
	for i := 0; i < 10; i++ {
		time.Sleep(5 * time.Second)
		memprofileFn := "memprofile.pprof." + strconv.Itoa(i)
		f, err := os.Create(memprofileFn)
		if err != nil {
			log.Fatal(err)
		}
		pprof.WriteHeapProfile(f)
		f.Close()
	}
}

func checkSdlError() {
	error:=sdl.GetError()

	if error!=nil {
		log.Printf("sdl: ",error)
		sdl.ClearError()
	}
}

func main() {
	runtime.GOMAXPROCS(16 + runtime.NumCPU())
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	sdl.ClearError()

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
		pprof.StopCPUProfile()

		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}

		defer pprof.StopCPUProfile()
	}
	if *memprofile != "" {
		go memProfileWriter()
	}

	windowInit()
	go printPixel()
	go printFps()

	ticker := time.NewTicker(1000 / 30 * time.Millisecond) //target 30fps
	go func() {
		for range ticker.C {
			if isRunning() {
				updateWin()
			} else {
				log.Print("Exit Window update ticker")
				return //the end
			}
		}
	}()

	go Server(serverQuit)

	//simulated messages
	for i := 0; i < numSimUpdater; i++ {
		go updateSim(i)
	}

	sdlEventLoop()

	(*window).Destroy()

}
