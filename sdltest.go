package main

import (
	"github.com/veandco/go-sdl2/sdl"
	"unsafe"
	"time"
	"math/rand"
)

type opix struct {
	R byte
	G byte
	B byte
}

func printFps(frames *uint32) {
	for {
		time.Sleep(time.Second * 1)
		println(*frames)
		*frames = 0
	}
}

func main() {
	/*const W = 1920
	const H = 1080 */
	const W=600
	const H=200
	var frames uint32

		if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}
	defer sdl.Quit()

	go printFps(&frames)

	window, err := sdl.CreateWindow("test", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		W, H, sdl.WINDOW_SHOWN)
	if err != nil {
		panic(err)
	}
	defer window.Destroy()

	pixels := [W][H]opix{}
	allRect := sdl.Rect{0, 0, W, H}
	srd, err := sdl.CreateRGBSurfaceFrom(unsafe.Pointer(&pixels), W, H, 24, W*3, 0, 0, 0, 0)
	for x := 0; x < W; x++ {
		for y := 0; y < H; y++ {
			rnd := rand.Uint32()
			pixels[x][y].R = byte(rnd & 0xff)
			pixels[x][y].G = byte((rnd >> 8) & 0xff)
			pixels[x][y].B = byte((rnd >> 16) & 0xff)
		}
	}
	for {


		surface, err := window.GetSurface()

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
				//running = false
				break
			}
		}
	}
}
