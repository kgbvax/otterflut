package main

import (
	"log"
	"github.com/go-gl/glfw/v3.2/glfw")

const ox float32 = 1.0
const oy float32 = 1.0

var quadVertices  = [...]float32{
	-1.0 + ox, 1.0 - oy,
	1.0 - ox, 1.0 - oy,
	-1.0 + ox, -1.0 + oy,
	1.0 - ox, -1.0 + oy,
	-1.0 + ox, -1.0 + oy,
	1.0 - ox, 1.0 - oy}

var quadTexcoord = [...]float32{
	0.0, 0.0,
	1.0, 0.0,
	0.0, 1.0,
	1.0, 1.0,
	0.0, 1.0,
	1.0, 0.0}

var  fragment_shader = `precision mediump float;
varying vec2 uv;
uniform sampler2D tex;
void main()
{
gl_FragColor = vec4(texture2D(tex, uv).rgb, 1.0); 
}`

var vertex_shader = `attribute vec2 vPosition;
	attribute vec2 vTexcoord;\
	varying vec2 uv;
	void main()
	{
	    gl_Position = vec4(vPosition, 0, 1);
	    uv = vTexcoord;
	};`

func initGlfw() {
	var err error

	if err := glfw.Init(); err != nil {
		log.Fatalln("failed to initialize glfw:", err)
	}


	monitor := glfw.GetPrimaryMonitor()

	W = uint32(monitor.GetVideoMode().Width)
	H = uint32(monitor.GetVideoMode().Height)

	log.Printf("monitor '%v' %v x %v", monitor.GetName(), W, H)
	window, err = glfw.CreateWindow(int(W), int(H), "Otterflut", nil, nil)
	if err != nil {
		panic(err)
	}
	window.MakeContextCurrent()
}