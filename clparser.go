package main

import (
	 
	"math/rand"
    "otterflut/cl"
	"log"
)

var kernelSource = `
__kernel void square(
   __global float* input,
   __global float* output,
   const unsigned int count)
{
   int i = get_global_id(0);
   if(i < count)
       output[i] = input[i] * input[i];
}
`

func main() {
	var data [1024]float32
	for i := 0; i < len(data); i++ {
		data[i] = rand.Float32()
	}

	platforms, err := cl.GetPlatforms()
	if err != nil {
		log.Fatalf("Failed to get platforms: %+v", err)
	}
	for i, p := range platforms {
		log.Printf("Platform %d:", i)
		log.Printf("  Name: %s", p.Name())
		log.Printf("  Vendor: %s", p.Vendor())
		log.Printf("  Profile: %s", p.Profile())
		log.Printf("  Version: %s", p.Version())
		log.Printf("  Extensions: %s", p.Extensions())
	}
	platform := platforms[0]

	devices, err := platform.GetDevices(cl.DeviceTypeAll)
	if err != nil {
		log.Fatalf("Failed to get devices: %+v", err)
	}
	if len(devices) == 0 {
		log.Fatalf("GetDevices returned no devices")
	}
	deviceIndex := -1
	for i, d := range devices {
		if deviceIndex < 0 && d.Type() == cl.DeviceTypeGPU {
			deviceIndex = i
		}
		log.Printf("Device %d (%s): %s", i, d.Type(), d.Name())
		log.Printf("  Address Bits: %d", d.AddressBits())
		log.Printf("  Available: %+v", d.Available())
		// log.Printf("  Built-In Kernels: %s", d.BuiltInKernels())
		log.Printf("  Compiler Available: %+v", d.CompilerAvailable())
		log.Printf("  Double FP Config: %s", d.DoubleFPConfig())
		log.Printf("  Driver Version: %s", d.DriverVersion())
		log.Printf("  Error Correction Supported: %+v", d.ErrorCorrectionSupport())
		log.Printf("  Execution Capabilities: %s", d.ExecutionCapabilities())
		log.Printf("  Extensions: %s", d.Extensions())
		log.Printf("  Global Memory Cache Type: %s", d.GlobalMemCacheType())
		log.Printf("  Global Memory Cacheline Size: %d KB", d.GlobalMemCachelineSize()/1024)
		log.Printf("  Global Memory Size: %d MB", d.GlobalMemSize()/(1024*1024))
		//log.Printf("  Half FP Config: %s", d.HalfFPConfig())
		log.Printf("  Host Unified Memory: %+v", d.HostUnifiedMemory())
		log.Printf("  Image Support: %+v", d.ImageSupport())
		log.Printf("  Image2D Max Dimensions: %d x %d", d.Image2DMaxWidth(), d.Image2DMaxHeight())
		log.Printf("  Image3D Max Dimenionns: %d x %d x %d", d.Image3DMaxWidth(), d.Image3DMaxHeight(), d.Image3DMaxDepth())
		// log.Printf("  Image Max Buffer Size: %d", d.ImageMaxBufferSize())
		// log.Printf("  Image Max Array Size: %d", d.ImageMaxArraySize())
		// log.Printf("  Linker Available: %+v", d.LinkerAvailable())
		log.Printf("  Little Endian: %+v", d.EndianLittle())
		log.Printf("  Local Mem Size Size: %d KB", d.LocalMemSize()/1024)
		log.Printf("  Local Mem Type: %s", d.LocalMemType())
		log.Printf("  Max Clock Frequency: %d", d.MaxClockFrequency())
		log.Printf("  Max Compute Units: %d", d.MaxComputeUnits())
		log.Printf("  Max Constant Args: %d", d.MaxConstantArgs())
		log.Printf("  Max Constant Buffer Size: %d KB", d.MaxConstantBufferSize()/1024)
		log.Printf("  Max Mem Alloc Size: %d KB", d.MaxMemAllocSize()/1024)
		log.Printf("  Max Parameter Size: %d", d.MaxParameterSize())
		log.Printf("  Max Read-Image Args: %d", d.MaxReadImageArgs())
		log.Printf("  Max Samplers: %d", d.MaxSamplers())
		log.Printf("  Max Work Group Size: %d", d.MaxWorkGroupSize())
		log.Printf("  Max Work Item Dimensions: %d", d.MaxWorkItemDimensions())
		log.Printf("  Max Work Item Sizes: %d", d.MaxWorkItemSizes())
		log.Printf("  Max Write-Image Args: %d", d.MaxWriteImageArgs())
		log.Printf("  Memory Base Address Alignment: %d", d.MemBaseAddrAlign())
		log.Printf("  Native Vector Width Char: %d", d.NativeVectorWidthChar())
		log.Printf("  Native Vector Width Short: %d", d.NativeVectorWidthShort())
		log.Printf("  Native Vector Width Int: %d", d.NativeVectorWidthInt())
		log.Printf("  Native Vector Width Long: %d", d.NativeVectorWidthLong())
		log.Printf("  Native Vector Width Float: %d", d.NativeVectorWidthFloat())
		log.Printf("  Native Vector Width Double: %d", d.NativeVectorWidthDouble())
		log.Printf("  Native Vector Width Half: %d", d.NativeVectorWidthHalf())
		log.Printf("  OpenCL C Version: %s", d.OpenCLCVersion())
		// log.Printf("  Parent Device: %+v", d.ParentDevice())
		log.Printf("  Profile: %s", d.Profile())
		log.Printf("  Profiling Timer Resolution: %d", d.ProfilingTimerResolution())
		log.Printf("  Vendor: %s", d.Vendor())
		log.Printf("  Version: %s", d.Version())
	}
	if deviceIndex < 0 {
		deviceIndex = 0
	}
	device := devices[deviceIndex]
	log.Printf("Using device %d", deviceIndex)
	context, err := cl.CreateContext([]*cl.Device{device})
	if err != nil {
		log.Fatalf("CreateContext failed: %+v", err)
	}
	// imageFormats, err := context.GetSupportedImageFormats(0, MemObjectTypeImage2D)
	// if err != nil {
	// 	t.Fatalf("GetSupportedImageFormats failed: %+v", err)
	// }
	// log.Printf("Supported image formats: %+v", imageFormats)
	queue, err := context.CreateCommandQueue(device, 0)
	if err != nil {
		log.Fatalf("CreateCommandQueue failed: %+v", err)
	}
	program, err := context.CreateProgramWithSource([]string{kernelSource})
	if err != nil {
		log.Fatalf("CreateProgramWithSource failed: %+v", err)
	}
	if err := program.BuildProgram(nil, ""); err != nil {
		log.Fatalf("BuildProgram failed: %+v", err)
	}
	kernel, err := program.CreateKernel("square")
	if err != nil {
		log.Fatalf("CreateKernel failed: %+v", err)
	}
	for i := 0; i < 3; i++ {
		name, err := kernel.ArgName(i)
		if err == cl.ErrUnsupported {
			break
		} else if err != nil {
			log.Printf("GetKernelArgInfo for name failed: %+v", err)
			break
		} else {
			log.Printf("Kernel arg %d: %s", i, name)
		}
	}
	input, err := context.CreateEmptyBuffer(cl.MemReadOnly, 4*len(data))
	if err != nil {
		log.Fatalf("CreateBuffer failed for input: %+v", err)
	}
	output, err := context.CreateEmptyBuffer(cl.MemReadOnly, 4*len(data))
	if err != nil {
		log.Fatalf("CreateBuffer failed for output: %+v", err)
	}
	if _, err := queue.EnqueueWriteBufferFloat32(input, true, 0, data[:], nil); err != nil {
		log.Fatalf("EnqueueWriteBufferFloat32 failed: %+v", err)
	}
	if err := kernel.SetArgs(input, output, uint32(len(data))); err != nil {
		log.Fatalf("SetKernelArgs failed: %+v", err)
	}

	local, err := kernel.WorkGroupSize(device)
	if err != nil {
		log.Fatalf("WorkGroupSize failed: %+v", err)
	}
	log.Printf("Work group size: %d", local)
	size, _ := kernel.PreferredWorkGroupSizeMultiple(nil)
	log.Printf("Preferred Work Group Size Multiple: %d", size)

	global := len(data)
	d := len(data) % local
	if d != 0 {
		global += local - d
	}
	if _, err := queue.EnqueueNDRangeKernel(kernel, nil, []int{global}, []int{local}, nil); err != nil {
		log.Fatalf("EnqueueNDRangeKernel failed: %+v", err)
	}

	if err := queue.Finish(); err != nil {
		log.Fatalf("Finish failed: %+v", err)
	}

	results := make([]float32, len(data))
	if _, err := queue.EnqueueReadBufferFloat32(output, true, 0, results, nil); err != nil {
		log.Fatalf("EnqueueReadBufferFloat32 failed: %+v", err)
	}

	correct := 0
	for i, v := range data {
		if results[i] == v*v {
			correct++
		}
	}

	if correct != len(data) {
		log.Fatalf("%d/%d correct values", correct, len(data))
	}
}
