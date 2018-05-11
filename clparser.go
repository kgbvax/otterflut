package main

//
// #cgo CFLAGS: -I CL
// #cgo arm64  LDFLAGS: -lOpenCL -L/usr/lib/aarch64-linux-gnu
// #cgo darwin LDFLAGS: -framework OpenCL
//

import (
	"gocl/cl"
	"unsafe"
	"gocl/cl_demo/utils"
	"log"
	"time"
	"github.com/dustin/go-humanize"
)

const pxMsgLen = 32
const batchSize = cl.CL_size_t(500000)
const numIter = 1000

var testMsg [32]cl.CL_char

func initTestMsg() {
	//xxxxxxxxxxx0123456789012345678
	const msg = "PX 1 2 ABCDEF"
	for k, v := range msg {
		testMsg[k] = cl.CL_char(v)
	}
}

func clinit() {
	initTestMsg()

	var A [pxMsgLen * batchSize]cl.CL_char //input array
	var X [batchSize]uint16                //output ushort[]
	var Y [batchSize]uint16                //output ushort[]
	var C [batchSize]uint32                //output uint

	datasizeA := cl.CL_size_t(unsafe.Sizeof(A))
	datasizeXY := cl.CL_size_t(unsafe.Sizeof(X))
	datasizeC := cl.CL_size_t(unsafe.Sizeof(C))

	// Allocate space for input/output data
	//A = make([][pxMsgLen]cl.CL_char, batchSize)
	//X = make([]cl.CL_int, batchSize)
	//Y = make([]cl.CL_int, batchSize)
	//C = make([]cl.CL_long, batchSize)

	var idx cl.CL_size_t

	for idx = 0; idx < batchSize; idx++ {
		var in cl.CL_size_t
		for in = 0; in < pxMsgLen; in++ {
			A[idx*pxMsgLen+in] = testMsg[in]
		}
	}

	// Use this to check the output of each API call
	var status cl.CL_int

	// STEP 1: Discover and initialize the platforms

	var numPlatforms cl.CL_uint
	var platforms []cl.CL_platform_id

	// Use clGetPlatformIDs() to retrieve the number of
	// platforms
	status = cl.CLGetPlatformIDs(0, nil, &numPlatforms)

	log.Printf("Number of platforms: \t%d\n", numPlatforms)

	// Allocate enough space for each platform
	platforms = make([]cl.CL_platform_id, numPlatforms)

	// Fill in platforms with clGetPlatformIDs()
	status = cl.CLGetPlatformIDs(numPlatforms, platforms, nil)
	if status != cl.CL_SUCCESS {
		log.Fatalf("CLGetPlatformIDs failed %v", status)
	}

	// Iterate through the list of platforms displaying associated information
	for i := cl.CL_uint(0); i < numPlatforms; i++ {
		// First we display information associated with the platform
		displayPlatformInfo(platforms[i], cl.CL_PLATFORM_PROFILE, "CL_PLATFORM_PROFILE")
		displayPlatformInfo(platforms[i], cl.CL_PLATFORM_VERSION, "CL_PLATFORM_VERSION")
		displayPlatformInfo(platforms[i], cl.CL_PLATFORM_VENDOR, "CL_PLATFORM_VENDOR")
		displayPlatformInfo(platforms[i], cl.CL_PLATFORM_EXTENSIONS, "CL_PLATFORM_EXTENSIONS")
	}

	// STEP 2: Discover and initialize the devices

	var numDevices cl.CL_uint
	var devices []cl.CL_device_id

	status = cl.CLGetDeviceIDs(platforms[0], cl.CL_DEVICE_TYPE_ALL, 0, nil, &numDevices)
	if status != cl.CL_SUCCESS {
		log.Fatalf("CLGetDeviceIDs failed %v", status)
	}

	// Allocate enough space for each device
	devices = make([]cl.CL_device_id, numDevices)

	// Fill in devices with clGetDeviceIDs()
	status = cl.CLGetDeviceIDs(platforms[0], cl.CL_DEVICE_TYPE_ALL, numDevices, devices, nil)
	if status != cl.CL_SUCCESS {
		log.Fatalf("CLGetDeviceIDs failed %v", status)
	}

	log.Printf("Number of devices: \t%d\n", numDevices)

	// Iterate through each device, displaying associated information
	for j := cl.CL_uint(0); j < numDevices; j++ {
		log.Printf("Device #%v", j)
		displayDeviceInfo(devices[j], cl.CL_DEVICE_TYPE, "CL_DEVICE_TYPE")
		displayDeviceInfo(devices[j], cl.CL_DEVICE_NAME, "CL_DEVICE_NAME")
		displayDeviceInfo(devices[j], cl.CL_DEVICE_VENDOR, "CL_DEVICE_VENDOR")
		displayDeviceInfo(devices[j], cl.CL_DEVICE_PROFILE, "CL_DEVICE_PROFILE")
		displayDeviceInfo(devices[j], cl.CL_DEVICE_MAX_COMPUTE_UNITS, "DEVICE_MAX_COMPUTE_UNITS")
		displayDeviceInfo(devices[j], cl.CL_DEVICE_LOCAL_MEM_SIZE, "DEVICE_LOCAL_MEM_SIZE")
		//displayDeviceInfo(devices[j], cl.CL_DEVICE_EXTENSIONS, "DEVICE_EXTENSIONS")
		log.Print("\n")
	}

	// STEP 3: Create a context

	selectedDevice := devices[0:1]
	log.Print("selected device:")
	displayDeviceInfo(selectedDevice[0], cl.CL_DEVICE_NAME, "CL_DEVICE_NAME")

	var context cl.CL_context
	context = cl.CLCreateContext(nil, 1, selectedDevice, nil, nil, &status)
	if status != cl.CL_SUCCESS {
		log.Fatalf("CLCreateContext failed %v", status)
	}

	// STEP 4: Create a command queue
	var cmdQueue cl.CL_command_queue

	cmdQueue = cl.CLCreateCommandQueue(context, selectedDevice[0], 0, &status)
	if status != cl.CL_SUCCESS {
		log.Fatalf("CLCreateCommandQueue %v", status)
	}



	bufferA := cl.CLCreateBuffer(context, cl.CL_MEM_READ_ONLY | cl.CL_MEM_USE_HOST_PTR , datasizeA, unsafe.Pointer(&A), &status)
	if status != cl.CL_SUCCESS {
		log.Fatalf("CLCreateBuffer failed %v", status)
	}

	bufferX := cl.CLCreateBuffer(context, cl.CL_MEM_WRITE_ONLY /*| cl.CL_MEM_ALLOC_HOST_PTR*/, datasizeXY, nil, &status)
	if status != cl.CL_SUCCESS {
		log.Fatalf("CLCreateBuffer failed %v", status)
	}

	bufferY := cl.CLCreateBuffer(context, cl.CL_MEM_WRITE_ONLY /*| cl.CL_MEM_ALLOC_HOST_PTR*/, datasizeXY, nil, &status)
	if status != cl.CL_SUCCESS {
		log.Fatalf("CLCreateBuffer failed %v", status)
	}

	bufferC := cl.CLCreateBuffer(context, cl.CL_MEM_WRITE_ONLY /* | cl.CL_MEM_ALLOC_HOST_PTR*/, datasizeC, nil, &status)
	if status != cl.CL_SUCCESS {
		log.Fatalf("CLCreateBuffer failed %v", status)
	}

	// STEP 7: Create and compile the program

	program := utils.Build_program(context, selectedDevice, "clparser.cl", []byte("-Werror"))
	if program == nil {
		log.Fatalf("Build program failed")
	}

	// STEP 8: Create the kernel
	var kernel cl.CL_kernel

	kernel = cl.CLCreateKernel(*program, []byte("clparser"), &status)
	if status != cl.CL_SUCCESS {
		log.Fatal("CLCreateKernel  failed %v", status)
	}

	// STEP 9: Set the kernel arguments
	status = cl.CLSetKernelArg(kernel, 0, cl.CL_size_t(unsafe.Sizeof(bufferA)), unsafe.Pointer(&bufferA))
	status |= cl.CLSetKernelArg(kernel, 1, cl.CL_size_t(unsafe.Sizeof(bufferX)), unsafe.Pointer(&bufferX))
	status |= cl.CLSetKernelArg(kernel, 2, cl.CL_size_t(unsafe.Sizeof(bufferY)), unsafe.Pointer(&bufferY))
	status |= cl.CLSetKernelArg(kernel, 3, cl.CL_size_t(unsafe.Sizeof(bufferC)), unsafe.Pointer(&bufferC))
	if status != cl.CL_SUCCESS {
		log.Fatalf("CLSetKernelArg failed %v", status)
	}

	// STEP 6: Write host data to device buffers
	log.Printf("go (batch size/iter %v/%v) total=%v", batchSize,numIter,humanize.Comma(int64(batchSize)*numIter))
	start := time.Now()

	for r := 0; r < numIter; r++ {
		/*
		not required since we submitted this via buffer allocation with CL_MEM_USE_HOST_PTR
		status = cl.CLEnqueueWriteBuffer(cmdQueue, bufferA, cl.CL_FALSE, 0, datasizeA, unsafe.Pointer(&A[0]), 0, nil, nil)
		if status != cl.CL_SUCCESS {
			log.Fatalf("CLEnqueueWriteBuffer failed %v", status)
		}*/

		// STEP 10: Configure the work-item structure

		var globalWorkSize [1]cl.CL_size_t
		// There are 'batchSize' work-items
		globalWorkSize[0] = batchSize

		// STEP 11: Enqueue the kernel for execution

		status = cl.CLEnqueueNDRangeKernel(cmdQueue, kernel, 1, nil, globalWorkSize[:], nil, 0, nil, nil)
		if status != cl.CL_SUCCESS {
			log.Fatalf("CLEnqueueNDRangeKernel failed %v", status)
		}

		// STEP 12: Read the output buffer back to the host

		cl.CLEnqueueReadBuffer(cmdQueue, bufferX, cl.CL_FALSE, 0, datasizeXY, unsafe.Pointer(&X[0]), 0, nil, nil)
		if status != cl.CL_SUCCESS {
			log.Fatalf("CLEnqueueReadBuffer failed %v", status)
		}

		cl.CLEnqueueReadBuffer(cmdQueue, bufferY, cl.CL_FALSE, 0, datasizeXY, unsafe.Pointer(&Y[0]), 0, nil, nil)
		if status != cl.CL_SUCCESS {
			log.Fatalf("CLEnqueueReadBuffer failed %v", status)
		}

		cl.CLEnqueueReadBuffer(cmdQueue, bufferC, cl.CL_FALSE, 0, datasizeC, unsafe.Pointer(&C[0]), 0, nil, nil)
		if status != cl.CL_SUCCESS {
			log.Fatal("CLEnqueueReadBuffer status!=cl.CL_SUCCESS")
		}
		cl.CLFinish(cmdQueue)
	}

	elapsed := time.Since(start)

	// STEP 13: Release OpenCL resources

	log.Printf("elapsed: %s", elapsed)
	log.Printf("x(batchSize/2)=%v", X[1])
	log.Printf("color(batchSize/2)=%X", C[1])
	log.Printf("datasizeC=%v",humanize.Comma(int64(datasizeC)))
	log.Printf("datasizeA=%v",humanize.Comma(int64(datasizeA)))
	log.Printf("throughtput %v PX/sec",humanize.Comma(int64((float64(batchSize*numIter)/elapsed.Seconds()))))

	// Free OpenCL resources
	cl.CLReleaseKernel(kernel)
	cl.CLReleaseProgram(*program)
	cl.CLReleaseCommandQueue(cmdQueue)
	cl.CLReleaseMemObject(bufferA)
	cl.CLReleaseMemObject(bufferX)
	cl.CLReleaseMemObject(bufferY)
	cl.CLReleaseMemObject(bufferC)
	cl.CLReleaseContext(context)
}

func main() {
	clinit()
}

func displayPlatformInfo(id cl.CL_platform_id, name cl.CL_platform_info, str string) {
	var errNum cl.CL_int
	var paramValueSize cl.CL_size_t

	errNum = cl.CLGetPlatformInfo(id, name, 0, nil, &paramValueSize)
	if errNum != cl.CL_SUCCESS {
		log.Fatalf("Failed to find OpenCL platform %s.\n", str)
	}

	var info interface{}
	errNum = cl.CLGetPlatformInfo(id, name, paramValueSize, &info, nil)
	if errNum != cl.CL_SUCCESS {
		log.Fatalf("Failed to find OpenCL platform %s.\n", str)
	}

	log.Printf("\t%-24s: %v\n", str, info)
}

func displayDeviceInfo(id cl.CL_device_id,
	name cl.CL_device_info,
	str string) {

	var errNum cl.CL_int
	var paramValueSize cl.CL_size_t

	errNum = cl.CLGetDeviceInfo(id, name, 0, nil, &paramValueSize)
	if errNum != cl.CL_SUCCESS {
		log.Fatalf("Failed to find OpenCL device info %s.\n", str)
	}

	var info interface{}
	errNum = cl.CLGetDeviceInfo(id, name, paramValueSize, &info, nil)
	if errNum != cl.CL_SUCCESS {
		log.Fatalf("Failed to find OpenCL device info %s.\n", str)
	}

	// Handle a few special cases
	switch name {
	case cl.CL_DEVICE_TYPE:
		var deviceTypeStr string

		appendBitfield(cl.CL_bitfield(info.(cl.CL_device_type)), cl.CL_bitfield(cl.CL_DEVICE_TYPE_CPU), "CL_DEVICE_TYPE_CPU", &deviceTypeStr)
		appendBitfield(cl.CL_bitfield(info.(cl.CL_device_type)), cl.CL_bitfield(cl.CL_DEVICE_TYPE_GPU), "CL_DEVICE_TYPE_GPU", &deviceTypeStr)
		appendBitfield(cl.CL_bitfield(info.(cl.CL_device_type)), cl.CL_bitfield(cl.CL_DEVICE_TYPE_ACCELERATOR), "CL_DEVICE_TYPE_ACCELERATOR", &deviceTypeStr)
		appendBitfield(cl.CL_bitfield(info.(cl.CL_device_type)), cl.CL_bitfield(cl.CL_DEVICE_TYPE_DEFAULT), "CL_DEVICE_TYPE_DEFAULT", &deviceTypeStr)
		info = deviceTypeStr
	default:
	}
	log.Printf("\t\t%-20s: %v\n", str, info)
}

func appendBitfield(info, value cl.CL_bitfield, name string, str *string) {
	if (info & value) != 0 {
		if len(*str) > 0 {
			*str += " | "
		}
		*str += name
	}
}
