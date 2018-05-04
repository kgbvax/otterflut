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
	"github.com/dustin/go-humanize"
	"unsafe"
	"math/rand"
)

//uint32 to save on sign manipulation in hot loop
var W uint32
var H uint32

//only used if simulation is active
var lines []string

const numSimUpdater = 1          //0=disable
var targetFps time.Duration = 15 //0=disable
const performTrace = false

var pixelXXCnt int64
var totalPixelCnt int64

var pixels *[]uint32

var xrunning = true

//statisitcs counters
var frames uint64
var errorCnt int64
var outOfRangeErrorCnt int64

var serverQuit = make(chan int)

//status line related state
var statsMsg = "ಠ_ಠ Please stand by."


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

	}
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err.Error())
		os.Exit(1)
	}
}

//no bounds check on x and y!
//color is whatever the underyling texture uses (BGR)
func setPixel(x uint32, y uint32, color uint32) /* chan? */ {

	offset := y*W + x
	offset2 := (W*H - offset)-1

	(*pixels)[offset2] = color

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
			time.Sleep(time.Duration(rand.Int63n(100)) * time.Nanosecond) // some random delay

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

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		//runtime.SetCPUProfileRate(200)
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}
	if *memprofile != "" {
		go memProfileWriter()
	}

	runtime.LockOSThread()  //OpenGL does not like being called from multiple threads
	initGl()

	buf :=  make([]uint32, W*H)
	pixels = (*[]uint32)(unsafe.Pointer(&buf)) //wrangle into array of uint32

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

	//main event loop
	if targetFps != 0 {
		ticker := time.NewTicker(1000 / targetFps * time.Millisecond) //target 30fps
		func() {
			for range ticker.C { //main event loop
				if !ofGlShouldClose() {
					drawScene()
					ofGlSwapBuffer()
					ofGlPollEvents()
					frames++
				} else {
					stopRunning()
					log.Print("Exit Window update ticker")
					return //the boom! //todo needs proper cleanup
				}
			}
		}()
	}
	//cleanup
}
