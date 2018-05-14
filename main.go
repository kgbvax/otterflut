package main

import (
	"fmt"
	"io/ioutil"
	"log"
	_ "net/http/pprof"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
	"github.com/dustin/go-humanize"
	"net/http"
	"github.com/pkg/profile"
)

const numSimUpdater = 10 //0=disable
const enableProfiling = false

var (
	//uint32 to save on sign manipulation in hot loop
	W uint32
	H uint32

	//only used if simulation is active
	blines [][]byte

	targetFps time.Duration = 15 //0=disable

	pixelXXCnt    int64
	totalPixelCnt int64

	pixels *[]uint32

	xrunning = true

	//statisitcs counters
	frames             uint64
	errorCnt           int64
	outOfRangeErrorCnt int64

	serverQuit = make(chan int)

	//status line related state
	statsMsg = "ಠ_ಠ Please stand by."
)

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
	offset2 := (W*H - offset) - 1
	(*pixels)[offset2] = color
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

func updateSim(gridx int) {
	for pixels == nil { //wait for bitmap to become available
		runtime.Gosched()
	}

	numLines := len(blines)
	for isRunning() {
		for _, element := range blines {
			//pfparse(element)
			//clparse([]byte(element))
			clparse(element)
			//time.Sleep(time.Duration(rand.Int63n(10)) * time.Nanosecond) // some random delay
		}
		atomic.AddInt64(&pixelXXCnt, int64(numLines))
	}
	log.Printf("Exit updateSim %v", gridx)
}

func main() {
	runtime.GOMAXPROCS(8 + runtime.NumCPU())
	if enableProfiling    {
		//defer profile.Start(profile.TraceProfile).Stop()
		defer profile.Start().Stop()

		go func() {
			log.Println(http.ListenAndServe("localhost:8080", nil))
		}()
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)


	initClParser()

	runtime.LockOSThread() //OpenGL does not like being called from multiple threads

	initGl()

	buf := make([]uint32, W*H)
	pixels = &buf// (*[]uint32)(unsafe.Pointer(&buf)) //wrangle into array of uint32

	setupScene()

	go updateStatsDisplay()


	//simulated messages
	if numSimUpdater > 0 {
		bdata, err := ioutil.ReadFile("test.pxfl")
		checkError(err)
		s := string(bdata)
		lines := strings.Split(s, "\n")
		blines = make ([][]byte,len(lines))
		for k,v := range lines {
			blines[k]=[]byte(v)
		}

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
