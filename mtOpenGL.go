package mtOpenGL

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"io"
	"os"
	"strings"
	"unsafe"

	"github.com/go-gl/gl/v4.5-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

type MeshBuffer struct {
	// Important geometry attributes
	ArrayBuffer  uint32
	VertexBuffer uint32
	VertexCount  int32
	IndexBuffer  uint32
	IndexCount   int32
}

type ImageTexture struct {
	TextureHandle uint32
	TextureSize   mgl32.Vec2
}

func compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)

	csources, free := gl.Strs(source)
	var csourceslength int32 = int32(len(source))
	gl.ShaderSource(shader, 1, csources, &csourceslength)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to compile %v: %v", source, log)
	}

	return shader, nil
}

func readFile(name string) (string, error) {

	buf := bytes.NewBuffer(nil)
	f, err := os.Open(name)
	if err != nil {
		return "", err
	}
	io.Copy(buf, f)
	f.Close()

	return string(buf.Bytes()), nil
}

// Mostly taken from the Demo. But compiling and linking shaders
// just should be done like this anyways.
func NewProgram(vertexShaderName, geometryShaderName, tessControlShaderName, tessEvalShaderName, fragmentShaderName string) (uint32, error) {
	useTessellationShader := tessControlShaderName != "" && tessEvalShaderName != ""
	useGeometryShader := geometryShaderName != ""

	vertexShaderSource, err := readFile(vertexShaderName)
	if err != nil {
		return 0, err
	}
	vertexShader, err := compileShader(vertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return 0, err
	}

	// Compile Tessellation shader
	var tessControlShader, tessEvalShader uint32
	if useTessellationShader {
		tessControlSource, err := readFile(tessControlShaderName)
		if err != nil {
			return 0, err
		}
		tessControlShader, err = compileShader(tessControlSource, gl.TESS_CONTROL_SHADER)
		if err != nil {
			return 0, err
		}
		tessEvalSource, err := readFile(tessEvalShaderName)
		if err != nil {
			return 0, err
		}
		tessEvalShader, err = compileShader(tessEvalSource, gl.TESS_EVALUATION_SHADER)
		if err != nil {
			return 0, err
		}
	}

	fragmentShaderSource, err := readFile(fragmentShaderName)
	if err != nil {
		return 0, err
	}
	fragmentShader, err := compileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		return 0, err
	}

	var geometryShader uint32
	if useGeometryShader {
		geometryShaderSource, err := readFile(geometryShaderName)
		if err != nil {
			return 0, err
		}
		geometryShader, err = compileShader(geometryShaderSource, gl.GEOMETRY_SHADER)
		if err != nil {
			return 0, err
		}
	}

	program := gl.CreateProgram()

	gl.AttachShader(program, vertexShader)
	if useTessellationShader {
		gl.AttachShader(program, tessControlShader)
		gl.AttachShader(program, tessEvalShader)
	}
	gl.AttachShader(program, fragmentShader)
	if useGeometryShader {
		gl.AttachShader(program, geometryShader)
	}
	gl.LinkProgram(program)

	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(program, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to link program: %v", log)
	}

	gl.DeleteShader(vertexShader)
	if useTessellationShader {
		gl.DeleteShader(tessControlShader)
		gl.DeleteShader(tessEvalShader)
	}
	gl.DeleteShader(fragmentShader)
	if useGeometryShader {
		gl.DeleteShader(geometryShader)
	}

	return program, nil
}

func NewComputeProgram(computeShaderName string) (uint32, error) {

	computeShaderSource, err := readFile(computeShaderName)
	if err != nil {
		return 0, err
	}
	computeShader, err := compileShader(computeShaderSource, gl.COMPUTE_SHADER)
	if err != nil {
		return 0, err
	}
	program := gl.CreateProgram()

	gl.AttachShader(program, computeShader)
	gl.LinkProgram(program)

	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(program, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to link program: %v", log)
	}

	gl.DeleteShader(computeShader)

	return program, nil
}

func CreateTexture(width, height int32, internalFormat, format, internalType uint32, multisampling bool, samples, mipmapLevels int32) uint32 {

	var texType uint32 = gl.TEXTURE_2D
	if multisampling {
		texType = gl.TEXTURE_2D_MULTISAMPLE
	}

	var tex uint32
	gl.GenTextures(1, &tex)
	gl.BindTexture(texType, tex)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)

	if multisampling {
		gl.TexStorage2DMultisample(gl.TEXTURE_2D_MULTISAMPLE, samples, internalFormat, width, height, false)
	} else {
		gl.TexStorage2D(gl.TEXTURE_2D, mipmapLevels, internalFormat, width, height)
	}

	return tex
}

func CreateImageTexture(imageName string, isRepeating bool) ImageTexture {

	var imageTexture ImageTexture

	img, err := LoadImage(imageName)
	if err != nil {
		fmt.Printf("Image load failed: %v.\n", err)
	}

	var textureWrap int32 = gl.CLAMP_TO_EDGE
	if isRepeating {
		textureWrap = gl.REPEAT
	}

	rgbaImg := image.NewRGBA(img.Img.Bounds())
	draw.Draw(rgbaImg, rgbaImg.Bounds(), img.Img, image.Pt(0, 0), draw.Src)

	gl.GenTextures(1, &imageTexture.TextureHandle)
	gl.BindTexture(gl.TEXTURE_2D, imageTexture.TextureHandle)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, textureWrap)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, textureWrap)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, int32(img.Img.Bounds().Max.X), int32(img.Img.Bounds().Max.Y), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(rgbaImg.Pix))

	imageTexture.TextureSize = mgl32.Vec2{float32(img.Img.Bounds().Max.X), float32(img.Img.Bounds().Max.Y)}

	return imageTexture

}

func CreateFboWithExistingTextures(colorTex, depthTex *uint32, texType uint32) uint32 {

	var fbo uint32
	gl.GenFramebuffers(1, &fbo)
	gl.BindFramebuffer(gl.FRAMEBUFFER, fbo)

	if colorTex != nil {
		gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, texType, *colorTex, 0)
	}
	if depthTex != nil {
		gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, texType, *depthTex, 0)
	}

	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)

	return fbo
}

// Some internal format changes, like only having the RG channels but with higher 32F precision.
func CreateLightFbo(colorTex, depthTex *uint32, width, height int32, multisampling bool, samples int32) uint32 {

	if colorTex != nil {
		*colorTex = CreateTexture(width, height, gl.RG32F, gl.RG, gl.FLOAT, multisampling, samples, 1)
	}
	if depthTex != nil {
		*depthTex = CreateTexture(width, height, gl.DEPTH_COMPONENT32, gl.DEPTH_COMPONENT, gl.FLOAT, multisampling, samples, 1)
	}

	var texType uint32 = gl.TEXTURE_2D
	if multisampling {
		texType = gl.TEXTURE_2D_MULTISAMPLE
	}

	return CreateFboWithExistingTextures(colorTex, depthTex, texType)
}

func CreateFbo(colorTex, depthTex *uint32, width, height int32, multisampling bool, samples int32, isFloatingPoint bool, mipmapLevels int32) uint32 {

	var intFormat uint32 = uint32(gl.RGBA8)
	var format uint32 = uint32(gl.RGBA)
	var ttype uint32 = uint32(gl.UNSIGNED_BYTE)

	if isFloatingPoint {
		intFormat = gl.RGBA32F
		ttype = gl.FLOAT
	}

	if colorTex != nil {
		*colorTex = CreateTexture(width, height, intFormat, format, ttype, multisampling, samples, mipmapLevels)
	}
	if depthTex != nil {
		*depthTex = CreateTexture(width, height, gl.DEPTH_COMPONENT32, gl.DEPTH_COMPONENT, gl.FLOAT, multisampling, samples, 1)
	}

	var texType uint32 = gl.TEXTURE_2D
	if multisampling {
		texType = gl.TEXTURE_2D_MULTISAMPLE
	}

	return CreateFboWithExistingTextures(colorTex, depthTex, texType)
}

func GenerateBufferFromTriangles2D(bufferObject *MeshBuffer, points []mgl32.Vec2) {

	if len(points) < 3 {
		return
	}

	var tmpM mgl32.Vec2
	stride := int32(unsafe.Sizeof(tmpM))

	gl.GenBuffers(1, &bufferObject.ArrayBuffer)
	gl.BindBuffer(gl.ARRAY_BUFFER, bufferObject.ArrayBuffer)
	gl.BufferData(gl.ARRAY_BUFFER, int(stride)*len(points), gl.Ptr(points), gl.STATIC_DRAW)

	gl.GenVertexArrays(1, &bufferObject.VertexBuffer)
	gl.BindVertexArray(bufferObject.VertexBuffer)

	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, stride, gl.PtrOffset(0))

	bufferObject.VertexCount = int32(len(points))

	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
}

func GenerateBufferFromLines2D(bufferObject *MeshBuffer, points []mgl32.Vec2, indices []uint32) {

	if len(points) < 2 || len(indices) < 2 {
		return
	}

	var tmpM mgl32.Vec2
	stride := int32(unsafe.Sizeof(tmpM))
	var ui uint32
	uiStride := int(unsafe.Sizeof(ui))

	gl.GenBuffers(1, &bufferObject.ArrayBuffer)
	gl.BindBuffer(gl.ARRAY_BUFFER, bufferObject.ArrayBuffer)
	gl.BufferData(gl.ARRAY_BUFFER, int(stride)*len(points), gl.Ptr(points), gl.STATIC_DRAW)

	gl.GenVertexArrays(1, &bufferObject.VertexBuffer)
	gl.BindVertexArray(bufferObject.VertexBuffer)

	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, stride, gl.PtrOffset(0))

	bufferObject.VertexCount = int32(len(points))

	gl.BindBuffer(gl.ARRAY_BUFFER, 0)

	gl.GenBuffers(1, &bufferObject.IndexBuffer)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, bufferObject.IndexBuffer)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, uiStride*len(indices), gl.Ptr(indices), gl.STATIC_DRAW)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, 0)
	bufferObject.IndexCount = int32(len(indices))

}

func FreeGLBuffer(buffer *MeshBuffer) {
	if buffer.VertexCount > 0 {
		gl.DeleteBuffers(1, &buffer.ArrayBuffer)
		gl.DeleteVertexArrays(1, &buffer.VertexBuffer)
		buffer.VertexCount = 0
		buffer.ArrayBuffer = 0
		buffer.VertexBuffer = 0
	}

	if buffer.IndexCount > 0 {
		// Does it work, if the buffer was never used?
		gl.DeleteBuffers(1, &buffer.IndexBuffer)
		buffer.VertexCount = 0
		buffer.IndexBuffer = 0
	}
}
