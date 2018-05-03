//
// +build arm arm64
// +build linux
//

package main



import (
	"log"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/go-gl/gl/v3.1/gles2"
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

	if err := gles2.Init(); err != nil {
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
	gles2.Enable(gles2.TEXTURE_2D)
	gles2.GenTextures(1, &texture)
	gles2.BindTexture(gles2.TEXTURE_2D, texture)
	gles2.TexParameteri(gles2.TEXTURE_2D, gles2.TEXTURE_MIN_FILTER, gles2.LINEAR)
	gles2.TexParameteri(gles2.TEXTURE_2D, gles2.TEXTURE_MAG_FILTER, gles2.LINEAR)
	gles2.TexParameteri(gles2.TEXTURE_2D, gles2.TEXTURE_WRAP_S, gles2.CLAMP_TO_EDGE)
	gles2.TexParameteri(gles2.TEXTURE_2D, gles2.TEXTURE_WRAP_T, gles2.CLAMP_TO_EDGE)
	gles2.TexImage2D(
		gles2.TEXTURE_2D,
		0,
		gles2.RGBA,
		int32(W),
		int32(H),
		0,
		gles2.RGBA,
		gles2.UNSIGNED_BYTE,
		gles2.Ptr(*pixels))


	return texture
}

func setupScene() {
	//gles2.Enable(gles2.DEPTH_TEST)

	gles2.ClearColor(0.0, 0.0, 0.0, 0.0)
	gles2.ClearDepth(1)
	gles2.DepthFunc(gles2.LEQUAL)

	gles2.MatrixMode(gles2.PROJECTION)
	gles2.LoadIdentity()
	gles2.Frustum(-1, 1, -1, 1, 1.0, 10.0)
	gles2.MatrixMode(gles2.MODELVIEW)
	gles2.LoadIdentity()
}

func destroyScene() {
}

func tearDown() {
	glfw.Terminate()

}

func drawScene() {
	gles2.Clear(gles2.COLOR_BUFFER_BIT)

	gles2.MatrixMode(gles2.MODELVIEW)
	gles2.LoadIdentity()
	gles2.Translatef(0, 0, -0.0000001) //I have no idea what I am doing ;-)
	gles2.Rotatef(0, 0, 0, 0)

	var texture = makeTexture()

	gles2.BindTexture(gles2.TEXTURE_2D, texture)
	gles2.Color4f(1, 1, 1, 1)

	gles2.Begin(gles2.QUADS)
	gles2.TexCoord2f(0, 0)
	gles2.Vertex3f(-1, -1, -1)
	gles2.TexCoord2f(1, 0)
	gles2.Vertex3f(1, -1, -1)
	gles2.TexCoord2f(1, 1)
	gles2.Vertex3f(1, 1, -1)
	gles2.TexCoord2f(0, 1)
	gles2.Vertex3f(-1, 1, -1)
	gles2.End()
}

