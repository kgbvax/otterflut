//
// +build darwin linux
// +build amd64 arm64
//
// ^^  tested platforms

package main

import (
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/go-gl/gl/v3.2-compatibility/gl"

)

var (
	texture uint32
	window *glfw.Window
	)

/*
 const  quad_texcoord []gl
{
0.0f, 0.0f,
1.0f, 0.0f,
0.0f, 1.0f,
1.0f, 1.0f,
0.0f, 1.0f,
1.0f, 0.0f
}  */



func ofGlShouldClose() bool {
	return window.ShouldClose()
}

func  ofGlSwapBuffer() {
	window.SwapBuffers()
}

func ofGlPollEvents() {
	glfw.PollEvents()
}

func makeTexture()  {

	gl.DeleteTextures(1,&texture)


	gl.GenTextures(1, &texture)
	gl.BindTexture(gl.TEXTURE_2D, texture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(W), int32(H), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(*pixels))

}

func setupScene() {
	//gl.Enable(gl.DEPTH_TEST)

	if err := gl.Init(); err != nil {
		panic(err)
	}
	gl.Enable(gl.TEXTURE_2D)
	gl.Disable(gl.DEPTH_TEST)


	gl.ClearColor(0.0, 0.0, 0.0, 0.0)


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

	makeTexture()

	gl.BindTexture(gl.TEXTURE_2D, texture)
	gl.Color4f(1, 1, 1, 1)

	/* gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 0, unsafe.Pointer(&quadVertices))
	gl.EnableVertexAttribArray(0)

	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, 0, unsafe.Pointer(&quadTexcoord))
	gl.EnableVertexAttribArray(1) */

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

