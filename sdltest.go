package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	_ "net/http/pprof"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/dustin/go-humanize"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
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

const numSimUpdater = 2
const TargetFps = 3.75
const PerformTrace = false

var pixelXXCnt int64
var totalPixelCnt int64

var pixels *[]uint32
var maxOffset uint32

var pixelsArr []byte
var sdlTexture *sdl.Texture
var renderer *sdl.Renderer
var allDisplay *sdl.Rect
var font *ttf.Font

var xrunning = true
var window *sdl.Window = nil

var frames uint64
var errorCnt int64
var outOfRangeErrorCnt int64

var serverQuit = make(chan int)

var globalWinUpdateLock = sync.Mutex{}

//status line related state
var statsMsg = "Please stand by."
var statusTextTexture *sdl.Texture
var statusTextRect *sdl.Rect

func updateStatsDisplay() {
	for isRunning() {
		time.Sleep(time.Second * 1)
		var sumPixelCount int64
		sumPixelCount = atomic.LoadInt64(&pixelXXCnt)

		statsMsg = fmt.Sprintf("OTTERFLUT IP: %v, PORT:%v\nSTATS ERR:out-of-range:%v parse:%v FPS=%v MSG:total=%v last=%v ", findMyIp(), port, outOfRangeErrorCnt, errorCnt, atomic.LoadUint64(&frames), humanize.Comma(totalPixelCnt), humanize.Comma(sumPixelCount))

		log.Print(statsMsg)

		totalPixelCnt += sumPixelCount
		atomic.StoreInt64(&pixelXXCnt, 0)
		atomic.StoreUint64(&frames, 0)
		statusTextTexture =nil //invalidate surface, needs to be re-generated

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
	offset := y*W + x

	if offset <= maxOffset { //out of bounds
		(*pixels)[offset] = color //uint32((color & 0xff0000) >> 16) | uint32((color & 0xff00) >> 8) | uint32(color & 0xff)

	} else {
		//log.Printf("pixel out of range %v %v ",x,y)
		atomic.AddInt64(&outOfRangeErrorCnt, 1)
		return // ignore
	}
}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")
var tracecall = flag.String("trace", "", "write trace profile to `file`")

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

	var err error

	globalWinUpdateLock.Lock()
	defer globalWinUpdateLock.Unlock()


	sdlTexture.Unlock()
	defer sdlTexture.Lock(allDisplay)

	sdlTexture.Update(allDisplay, pixelsArr, int(W*4))
	renderer.Clear()

	renderer.Copy(sdlTexture, allDisplay, allDisplay)

	if statusTextTexture == nil {
		var statusTextSurface *sdl.Surface
		if statusTextSurface, err = font.RenderUTF8BlendedWrapped(statsMsg, sdl.Color{255, 255, 0, 200}, int(allDisplay.W)); err != nil {
			log.Printf( "Failed to render text: %s\n", err)
			return
		}
		defer statusTextSurface.Free()
		statusTextRect = &sdl.Rect{0,0,statusTextSurface.W,statusTextSurface.H}
		statusTextTexture, err = renderer.CreateTextureFromSurface(statusTextSurface)
		checkErr(err)
	}

	renderer.Copy(statusTextTexture, statusTextRect ,statusTextRect)

	renderer.Present()
	frames++
}

func windowInit() {
	var err error

	platform := sdl.GetPlatform()
	workingDir, err := os.Getwd()

	log.Printf("platform: %v CWD:%v", platform, workingDir)

	err = ttf.Init()
	checkErr(err)

	font, err = ttf.OpenFont("Inconsolata-Regular.ttf", 24)
	checkErr(err)

	numModes, err := sdl.GetNumDisplayModes(0)
	for i := 0; i <= numModes; i++ {
		mode, _ := sdl.GetDisplayMode(0, i)
		log.Printf("mode %vx%v@%v f:%v", mode.W, mode.H, mode.RefreshRate, mode.Format)
	}

	rendererIndex := -1
	numdrv, err := sdl.GetNumRenderDrivers()
	for i := 0; i < numdrv; i++ {
		var rinfo sdl.RendererInfo
		sdl.GetRenderDriverInfo(i, &rinfo)
		rendererName := rinfo.Name
		log.Printf("available renderer driver: #%v %v, flags:%b ", i, rendererName, rinfo.Flags)
		if platform == "Mac OS X" && rendererName == "metal" {
			rendererIndex = i
		}
	}

	if err = sdl.Init(sdl.INIT_VIDEO); err != nil {
		panic(err)
	}
	checkSdlError()

	//checkSdlError()
	displayBounds, err := sdl.GetDisplayBounds(0)

	checkErr(err)
	log.Printf("display: %vx%v", displayBounds.W, displayBounds.H)

	W = uint32(displayBounds.W)
	H = uint32(displayBounds.H)
	allDisplay = &sdl.Rect{0, 0, int32(W), int32(H)}

	window, err = sdl.CreateWindow("otterflut", 0, 0, int32(W), int32(H),
		sdl.WINDOW_SHOWN|sdl.WINDOW_ALLOW_HIGHDPI|sdl.WINDOW_FULLSCREEN|sdl.WINDOW_OPENGL)

	log.Print("create renderer")
	renderer, err = sdl.CreateRenderer(window, rendererIndex,  sdl.RENDERER_PRESENTVSYNC | sdl.RENDERER_ACCELERATED)
	checkErr(err)
	checkSdlError()

	info, err := renderer.GetInfo()
	log.Printf("selected renderer: %v", info.Name)
	log.Printf("max texture size: %vx%v", info.MaxTextureWidth, info.MaxTextureHeight)

	sdlTexture, err = renderer.CreateTexture(
		sdl.PIXELFORMAT_ARGB8888,
		sdl.TEXTUREACCESS_STREAMING,
		int32(W), int32(H))
	checkErr(err)
	checkSdlError()
	//sdlTexture.Lock(nil)

	maxOffset = W * H

	pixelsArr = make([]byte, W*H*4)                  //the actual pixel buffer hidden in a golang array
	pixels = (*[]uint32)(unsafe.Pointer(&pixelsArr)) //wrangle into array of uint32

	sdl.DisableScreenSaver()
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
		if event != nil {
			//log.Print(event)
			switch event.(type) {
			case *sdl.QuitEvent:
				log.Print("SDL Quit")
				stopRunning()
				return
			}
		} else {
			err := sdl.GetError()

			if err != nil {
				log.Printf("sdl-event-loop: %v ", err)
				sdl.ClearError()
			}
		}
	}
}

func updateSim(gridx int) {
	for pixels == nil { //wait for bitmap to become available
		runtime.Gosched()
	}

	numLines := len(lines)
	for isRunning() {
		for _, element := range lines {
			pfparse([]byte(element))

		}
		atomic.AddInt64(&pixelXXCnt, int64(numLines))

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
	err := sdl.GetError()
	if err != nil {
		log.Printf("sdl: %v", err)
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

	runtime.GOMAXPROCS(8 + 16*runtime.NumCPU())
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	sdl.ClearError()

	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)

		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}

		runtime.SetCPUProfileRate(200)
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}

		defer pprof.StopCPUProfile()
	}
	if *memprofile != "" {
		go memProfileWriter()
	}

	windowInit()
	go updateStatsDisplay()

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
	if numSimUpdater > 0 {
		bdata, err := ioutil.ReadFile("test.pxfl")
		checkError(err)
		s := string(bdata)
		lines = strings.Split(s, "\n")

		for i := 0; i < numSimUpdater; i++ {
			go updateSim(i)
		}
	}

	sdlEventLoop()

	(*window).Destroy()
}
