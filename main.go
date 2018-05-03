package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	_ "net/http/pprof"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/go-gl/gl/v3.2-compatibility/gl"
	"github.com/dustin/go-humanize"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
	"math/rand"
	"unsafe"
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

const numSimUpdater = 1 //0=disable
var targetFps time.Duration = 30 //0=disable
const performTrace = false

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


//status line related state
var statsMsg = "ಠ_ಠ Please stand by."
var statusTextTexture *sdl.Texture
var statusTextRect *sdl.Rect
const lockTexture=false

var texture uint32

func init() {
	runtime.LockOSThread()
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func updateStatsDisplay() {
	for isRunning() {
		time.Sleep(time.Second * 1)
		var sumPixelCount int64
		sumPixelCount = atomic.LoadInt64(&pixelXXCnt)

		statsMsg = fmt.Sprintf("OTTERFLUT IP: %v, PORT:%v\nSTATS ERR:oor:%v parse:%v FPS:%v CONN:%v MSG:total=%v last=%v ", findMyIp(), port, outOfRangeErrorCnt, errorCnt, atomic.LoadUint64(&frames), currentConnections, humanize.Comma(totalPixelCnt), humanize.Comma(sumPixelCount))

		log.Print(statsMsg)

		totalPixelCnt += sumPixelCount
		atomic.StoreInt64(&pixelXXCnt, 0)
		atomic.StoreUint64(&frames, 0)

	//	textTextureUpdateLock.Lock()
		if statusTextTexture != nil { //invalidate surface, needs to be re-generated, todo race condition here, needs lock
			x:=statusTextTexture
			statusTextTexture=nil
			x.Destroy()
		}
		//textTextureUpdateLock.Unlock()
	}
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err.Error())
		os.Exit(1)
	}
}


func setPixel(x uint32, y uint32, color uint32) /* chan? */ {

	offset := y*W + x

	if x < W && y < H { //not out of bounds
	//	log.Printf("set pixel at %v %v %v",offset,x,y)

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

func updateSim(gridx int) {
	for pixels == nil { //wait for bitmap to become available
		runtime.Gosched()
	}

	numLines := len(lines)
	for isRunning() {
		for _, element := range lines {
			pfparse([]byte(element))
			time.Sleep(time.Duration(rand.Int63n(500)) * time.Nanosecond) // some random delay

		}
		atomic.AddInt64(&pixelXXCnt, int64(numLines))

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

func main() {
	runtime.GOMAXPROCS(8 + runtime.NumCPU())


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

	if err := glfw.Init(); err != nil {
		log.Fatalln("failed to initialize glfw:", err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 2)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	monitor:=glfw.GetPrimaryMonitor()

	W=uint32(monitor.GetVideoMode().Width)/2
	H=uint32(monitor.GetVideoMode().Height)/2

	log.Printf("monitor '%v' %v x %v",monitor.GetName(), W,H )
	window, err := glfw.CreateWindow(int(W), int(H), "Otterflut", nil, nil)
	if err != nil {
		panic(err)
	}
	window.MakeContextCurrent()


	if err := gl.Init(); err != nil {
		panic(err)
	}


	buf := createImageBuffer(int(W),int(H))
	pixels = (*[]uint32)(unsafe.Pointer(&buf)) //wrangle into array of uint32


	defer gl.DeleteTextures(1, &texture)

	maxOffset=W*H
	setupScene()

	go updateStatsDisplay()


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

	//start the TCP server
	go Server(serverQuit)

	if targetFps!=0  {
		ticker := time.NewTicker(1000 / targetFps * time.Millisecond) //target 30fps
		 func() {
			 for range ticker.C { //main event loop
				 if !window.ShouldClose() {
					 drawScene()
					 frames++

					 window.SwapBuffers()
					 glfw.PollEvents()
				 } else {
					 log.Print("Exit Window update ticker")
					 return //the boom! //todo needs proper cleanup
				 }
			 }
		}()
	}

}



func createImageBuffer( width int,  height int)  []uint32 {
	buffer:=make ([]uint32,W*H)
	return buffer
}

func makeTexture() uint32 {
	var texture uint32
	gl.Enable(gl.TEXTURE_2D)
	gl.GenTextures(1, &texture)
	gl.BindTexture(gl.TEXTURE_2D, texture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		int32(W),
		int32(H),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(*pixels))


	return texture
}

func setupScene() {
	//gl.Enable(gl.DEPTH_TEST)

	gl.ClearColor(0.0, 0.0, 0.0, 0.0)
	gl.ClearDepth(1)
	gl.DepthFunc(gl.LEQUAL)

	gl.MatrixMode(gl.PROJECTION)
	gl.LoadIdentity()
	gl.Frustum(-1, 1, -1, 1, 1.0, 10.0)
	gl.MatrixMode(gl.MODELVIEW)
	gl.LoadIdentity()
}

func destroyScene() {
}

func drawScene() {
	gl.Clear(gl.COLOR_BUFFER_BIT)

	gl.MatrixMode(gl.MODELVIEW)
	gl.LoadIdentity()
	gl.Translatef(0, 0, -0.0000001) //I have no idea what I am doing ;-)
	gl.Rotatef(0, 0, 0, 0)

	var texture = makeTexture()

	gl.BindTexture(gl.TEXTURE_2D, texture)
	gl.Color4f(1, 1, 1, 1)

	gl.Begin(gl.QUADS)
	gl.TexCoord2f(0, 0)
	gl.Vertex3f(-1, -1, -1)
	gl.TexCoord2f(1, 0)
	gl.Vertex3f(1, -1, -1)
	gl.TexCoord2f(1, 1)
	gl.Vertex3f(1, 1, -1)
	gl.TexCoord2f(0, 1)
	gl.Vertex3f(-1, 1, -1)
	gl.End()
}

