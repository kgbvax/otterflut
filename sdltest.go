package main

import (
	"flag"
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
	"sync"
	"runtime/trace"
	"github.com/dustin/go-humanize"
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


const numSimUpdater = 1
const TargetFps =2
const PerformTrace = false

var pixelCntSli [numSimUpdater]int64

var pixelXXCnt int64
var totalPixelCnt int64

var pixels *[]uint32
var maxOffset uint32

var pixelsArr []byte
var sdlTexture *sdl.Texture
var renderer *sdl.Renderer
var allDisplay *sdl.Rect

var xrunning = true
var window *sdl.Window=nil


var frames uint64
var errorCnt int64
var outOfRangeErrorCnt int64

var serverQuit = make(chan int)

var globalWinUpdateLock = sync.Mutex{}


func printFps() {
	for isRunning() {
		time.Sleep(time.Second * 1)
		var sumPixelCount int64
		sumPixelCount=pixelXXCnt
		log.Printf("errors (out of range:%v parse:%v) frames=%v pixel: total=%v last=%v ", outOfRangeErrorCnt, errorCnt, atomic.LoadUint64(&frames),humanize.Comma(totalPixelCnt),humanize.Comma(sumPixelCount))
		pixelXXCnt=0
		totalPixelCnt+=sumPixelCount
		atomic.StoreUint64(&frames, 0)
	}
}



func checkError(err error) {
	if err != nil {
		log.Fatal(err.Error())
		os.Exit(1)
	}
}

func setPixel(x uint32, y uint32, color uint32) /* chan? */ {

	/*	//sdlcol:=sdl.Color{R: uint8((color & 0xff0000) >> 16),G: uint8((color & 0xff00) >> 8), B: uint8(color & 0xff), A: uint8((color&0xff000000)>>24) }
		gfx.PixelRGBA(ren,int32(x),int32(y),255,255,0,255) */
	offset:=y*W+x

	if offset<=maxOffset { //out of bounds
		(*pixels)[offset] = color //uint32((color & 0xff0000) >> 16) | uint32((color & 0xff00) >> 8) | uint32(color & 0xff)

	} else {
		//log.Printf("pixel out of range %v %v ",x,y)
		atomic.AddInt64(&outOfRangeErrorCnt, 1)
		return // ignore
	}

	atomic.AddInt64(&pixelXXCnt,1)
	//pixelXXCnt+=1
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
	globalWinUpdateLock.Lock()

	frames++
	//window.UpdateSurface()
	sdlTexture.Unlock()

	sdlTexture.Update(allDisplay,pixelsArr,int(W*4))

	renderer.Clear()
	renderer.Copy(sdlTexture,allDisplay,allDisplay)
	renderer.Present()

	//window.UpdateSurface()
	sdlTexture.Lock(allDisplay)

	globalWinUpdateLock.Unlock()
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


	numModes,err:=sdl.GetNumDisplayModes(0)
	for i:=0; i<numModes; i++ {
		mode,_:=sdl.GetDisplayMode(0,i)
		log.Printf("mode %vx%v@%v f:%v",mode.W,mode.H,mode.RefreshRate,mode.Format)
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
	allDisplay = &sdl.Rect{0, 0, int32(W), int32(H)}


	window, err = sdl.CreateWindow("otterflut", 0, 0,int32(W),int32(H),
		sdl.WINDOW_SHOWN|sdl.WINDOW_ALLOW_HIGHDPI|sdl.WINDOW_FULLSCREEN | sdl.WINDOW_OPENGL)


	log.Print("create renderer")
	renderer,err = sdl.CreateRenderer(window,-1,sdl.RENDERER_PRESENTVSYNC|sdl.RENDERER_ACCELERATED)
	checkErr(err)
	checkSdlError()

	info,err:=renderer.GetInfo()
	log.Printf("selected renderer: %v",info.Name)
	log.Printf("max texture size: %vx%v",info.MaxTextureWidth,info.MaxTextureHeight)


	log.Print("create texture")
	sdlTexture,err = renderer.CreateTexture(
		sdl.PIXELFORMAT_ARGB8888,
		sdl.TEXTUREACCESS_STREAMING,
		int32(W), int32(H))
    checkErr(err)
	checkSdlError()
	//sdlTexture.Lock(nil)

	maxOffset= W*H

	pixelsArr = make ([]byte,W*H*4) //the actual pixel buffer hidden in a golang array
	pixels=(*[]uint32)(unsafe.Pointer(&pixelsArr)) //wrangle into array of uint32

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
			err:=sdl.GetError()

			if err!=nil {
				log.Printf("sdl-event-loop: %v ",err)
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

			pfparse([]byte(element))
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
	err:=sdl.GetError()
	if err!=nil {
		log.Printf("sdl: %v",err)
		sdl.ClearError()
	}
}

func main() {
	if PerformTrace {
		f, err := os.Create("trace.out")
		if err != nil {
			panic(err)
		}
		defer f.Close()

		err = trace.Start(f)
		if err != nil {
			panic(err)
		}
		defer trace.Stop()
	}

	runtime.GOMAXPROCS(8+16 * runtime.NumCPU())
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	sdl.ClearError()

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
		pprof.StopCPUProfile()
		runtime.SetCPUProfileRate(2000)
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}

		defer pprof.StopCPUProfile()
	}
	if *memprofile != "" {
		go memProfileWriter()
	}

	windowInit()
	go printFps()

	ticker := time.NewTicker(1000 / TargetFps * time.Millisecond) //target 30fps
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
