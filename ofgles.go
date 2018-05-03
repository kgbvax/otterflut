//
// +build arm arm64
// +build linux
//

package main



import (
	"log"
	"github.com/go-gl/glfw/v3.2/glfw"
)


var window *glfw.Window

func initGl() {
	var err error

	if err := glfw.Init(); err != nil {
		log.Fatalln("failed to initialize glfw:", err)
	}

	glfw.WindowHint(glfw.ClientAPI, glfw.OpenGLESAPI)
	glfw.WindowHint(glfw.ContextVersionMajor, 2)
	glfw.WindowHint(glfw.ContextVersionMinor, 0)

	monitor := glfw.GetPrimaryMonitor()

	W = uint32(monitor.GetVideoMode().Width) / 2
	H = uint32(monitor.GetVideoMode().Height) / 2

	log.Printf("monitor '%v' %v x %v", monitor.GetName(), W, H)
	window, err = glfw.CreateWindow(int(W), int(H), "Otterflut", nil, nil)
	if err != nil {
		panic(err)
	}
	window.MakeContextCurrent()

	if err := gl.Init(); err != nil {
		panic(err)
	}

}

func ofGlShouldClose() bool {
	return window.ShouldClose()
}

func  ofGlSwapBuffer() {
	window.SwapBuffers()
}

func ofGlPollEvents() {
	glfw.PollEvents()
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

func tearDown() {
	glfw.Terminate()

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

