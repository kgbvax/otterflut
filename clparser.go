package main

//
// #cgo CFLAGS: -I CL
// #cgo arm64  LDFLAGS: -lOpenCL -L/usr/lib/aarch64-linux-gnu
// #cgo darwin LDFLAGS: -framework OpenCL
//

/*
 * WARNING: This will fail if endianness of host and GPU do not match.
 * Mea culpa
 */

import (
	"github.com/rainliu/gocl/cl"
	"unsafe"
 	"log"
	"sync"
	"sync/atomic"
	"os"
)

var testMsg [22]byte

const (
	pxMsgLen = 22  //the maximum size of "PX" message we are willing to handle
	//batchSize = cl.CL_size_t(524288) // number of PX messages we submit to the kernel in one go
	batchSize = cl.CL_size_t(524288)
	lenUint16 = 2 //OpenCL ushort
	lenUint32 = 4 //OpenCL uint
	numIter   = 500 // only for testing

	datasizeA  = cl.CL_size_t(pxMsgLen * batchSize) //number of bytes in the A input array (PX msg)
	datasizeXY = cl.CL_size_t(batchSize * lenUint16) //number of bytes in the X/Y output
	datasizeC  = cl.CL_size_t(batchSize * lenUint32) //number of bytes in the color output
)

func initTestMsg() {
	//xxxxxxxxxxx0123456789012345678901
	const msg = "PX 255 1337 ABCDEF"
	//const msg2 ="PX 1111 1111 AABBCCDD"
	for k, v := range msg {
		testMsg[k] = byte(v)
	}
}

type pxBufT *[batchSize * pxMsgLen]byte
type coordBufT *[batchSize]uint16
type colBufT *[batchSize]uint32
type pixelBufT* [1920*1080]uint32

type oclParam struct {
	currentOffset	int32
	A        pxBufT    //input array
	Amutex	 sync.Mutex
	X        coordBufT //output ushort[]
	Y        coordBufT //output ushort[]
	C        colBufT   //output uint
	PXM      pixelBufT //output pixmap
	bufferA  cl.CL_mem
	bufferX  cl.CL_mem
	bufferY  cl.CL_mem
	bufferC  cl.CL_mem
	bufferPXM cl.CL_mem
	context  cl.CL_context
	kernel   cl.CL_kernel
	cmdQueue cl.CL_command_queue
	program  *cl.CL_program

}


var (
	left  *oclParam
	right *oclParam
	currentProc *oclParam
	oclBankSelect = false
)


func bankSelect() *oclParam {
	switch oclBankSelect {
	case true:
		return left
	case false:
		return right
	}
	return nil //never reached
}

func dumpA(px *pxBufT){
	var str=""
	pxbuf:=*px
 	for i:=0;i<3*pxMsgLen;i++ {
		str+=string(pxbuf[i])
	}
	log.Printf("A=%v/%v....",px,str)
}

func  goIt(lbf *oclParam) {
//	defer timeTrack(time.Now(), "OCLProc")


	var status cl.CL_int

	var globalWorkSize [1]cl.CL_size_t
	// There are 'batchSize' work-items
	globalWorkSize[0] = batchSize

	// STEP 11: Enqueue the kernel for execution

	status = cl.CLEnqueueNDRangeKernel(lbf.cmdQueue, lbf.kernel, 1, nil, globalWorkSize[:], nil, 0, nil, nil)
	if status != cl.CL_SUCCESS {
		log.Fatalf("CLEnqueueNDRangeKernel failed %v", status)
	}

	cl.CLFinish(lbf.cmdQueue)
	lbf.currentOffset=0 //set this buffer to 0
	lbf.Amutex.Unlock() // from here on updates of input params is accetable


	for i:=0;i<int(batchSize);i++ {
		x:=lbf.X[i]
		y:=lbf.Y[i]

	//	log.Printf("setting px %v %v %v",x,y,c)
		if uint32(x) <W&& uint32(y) < H{
			setPixel(uint32(x),uint32(y),lbf.C[i])
		}
	}
}

// to my dismay, we currently need to copy the "PX" line here into the target buffer
// todo next iteration, shove TCP buffers straight into OCL buffer
func copyArr(startOffset int32,bank *oclParam, m []byte) {
	//mlen:=int32(len(m))

	slicer:=(*bank.A)[startOffset:]
	//log.Printf("len/cap %v %v",len(m),cap(m))
	//log.Printf("len2/cap2 %v %v ",len(slicer[startOffset:]),cap(slicer[startOffset:]))
	//if (len(m)< cap(slicer)) {
	copy(slicer,m)
}

func clparse(m []byte) {
	bank:=bankSelect()

	//the next 5 lines are really fishy in terms of race conditions...

	endOffset:=atomic.AddInt32(&bank.currentOffset,pxMsgLen) //for the next to come...
	startOffset:=endOffset-pxMsgLen

	if endOffset >= int32(datasizeA) { //we're full, off to the races!
		oclBankSelect=!oclBankSelect  //select other bank so other callers
		bank.Amutex.Lock() //lock this bank until it is processed.

		//dumpA(&bank.A)
		go goIt(bank) //start processing this bank

		//log.Printf("flip bank to %v",&bankSelect().A)
  		clparse(m)  //try again
	} else {
		copyArr(startOffset,bank,m)
	}
}

func initClParser() {
	left = &oclParam{}
	right = &oclParam{}
	clinit(left)
	clinit(right)
	oclBankSelect=false
	currentProc=bankSelect()
}



func clinit(lbf *oclParam) {
	var idx cl.CL_size_t
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
		displayDeviceInfo(devices[j], cl.CL_DEVICE_ENDIAN_LITTLE, "DEVICE_ENDIAN_LITTLE")

		log.Print("\n")
	}

	// STEP 3: Create a context

	selectedDevice := devices[1:2]
	log.Print("selected device:")
	displayDeviceInfo(selectedDevice[0], cl.CL_DEVICE_NAME, "CL_DEVICE_NAME")

	lbf.context = cl.CLCreateContext(nil, 1, selectedDevice, nil, nil, &status)
	if status != cl.CL_SUCCESS {
		log.Fatalf("CLCreateContext failed %v", status)
	}

	// STEP 4: Create a command queue

	lbf.cmdQueue = cl.CLCreateCommandQueue(lbf.context, selectedDevice[0], 0, &status)
	if status != cl.CL_SUCCESS {
		log.Fatalf("CLCreateCommandQueue %v", status)
	}

	//Let OpenCL create the buffer (to deal with alignment)
	lbf.bufferA = cl.CLCreateBuffer(lbf.context, cl.CL_MEM_READ_ONLY|cl.CL_MEM_HOST_WRITE_ONLY|cl.CL_MEM_ALLOC_HOST_PTR, datasizeA, nil, &status)
	if status != cl.CL_SUCCESS {
		log.Fatalf("CLCreateBuffer failed %v", status)
	}
	//and map it back into host space, wait for sync
	lbf.A = pxBufT(cl.CLEnqueueMapBuffer(lbf.cmdQueue, lbf.bufferA, cl.CL_TRUE, cl.CL_MAP_WRITE, 0, datasizeA, 0, nil, nil, &status))
	if status != cl.CL_SUCCESS {
		log.Fatalf("MapBuffer failed %v", status)
	}

	lbf.bufferX = cl.CLCreateBuffer(lbf.context, cl.CL_MEM_WRITE_ONLY|cl.CL_MEM_HOST_READ_ONLY|cl.CL_MEM_ALLOC_HOST_PTR, datasizeXY, nil, &status)
	if status != cl.CL_SUCCESS {
		log.Fatalf("CLCreateBuffer failed %v", status)
	}
	lbf.X = coordBufT(cl.CLEnqueueMapBuffer(lbf.cmdQueue, lbf.bufferX, cl.CL_FALSE, cl.CL_MAP_READ, 0, datasizeXY, 0, nil, nil, &status))
	if status != cl.CL_SUCCESS {
		log.Fatalf("MapBuffer failed %v", status)
	}

	lbf.bufferY = cl.CLCreateBuffer(lbf.context, cl.CL_MEM_WRITE_ONLY|cl.CL_MEM_HOST_READ_ONLY|cl.CL_MEM_ALLOC_HOST_PTR, datasizeXY, nil, &status)
	if status != cl.CL_SUCCESS {
		log.Fatalf("CLCreateBuffer failed %v", status)
	}

	//map output buffers
	lbf.Y = coordBufT(cl.CLEnqueueMapBuffer(lbf.cmdQueue, lbf.bufferY, cl.CL_FALSE, cl.CL_MAP_READ, 0, datasizeXY, 0, nil, nil, &status))
	if status != cl.CL_SUCCESS {
		log.Fatalf("MapBuffer failed %v", status)
	}

	lbf.bufferC = cl.CLCreateBuffer(lbf.context, cl.CL_MEM_WRITE_ONLY|cl.CL_MEM_HOST_READ_ONLY|cl.CL_MEM_ALLOC_HOST_PTR, datasizeC, nil, &status)
	if status != cl.CL_SUCCESS {
		log.Fatalf("CLCreateBuffer failed %v", status)
	}

	lbf.C = colBufT(cl.CLEnqueueMapBuffer(lbf.cmdQueue, lbf.bufferC, cl.CL_FALSE, cl.CL_MAP_READ, 0, datasizeC, 0, nil, nil, &status))
	if status != cl.CL_SUCCESS {
		log.Fatalf("MapBuffer failed %v", status)
	}


	// init test data

	for idx = 0; idx < batchSize; idx++ {
		var in cl.CL_size_t
		for in = 0; in < pxMsgLen; in++ {
			lbf.A[idx*pxMsgLen+in] = testMsg[in]
		}
	}

	// Create and compile the program

	lbf.program = build_program(lbf.context, selectedDevice, "clparser.cl", []byte("-Werror"))
	if lbf.program == nil {
		log.Fatalf("Build program failed")
	}

	lbf.kernel = cl.CLCreateKernel(*lbf.program, []byte("clparser"), &status)
	if status != cl.CL_SUCCESS {
		log.Fatalf("CLCreateKernel failed %v", status)
	}

	//  Set the kernel arguments
	bufA := lbf.bufferA
	bufX := lbf.bufferX
	bufY := lbf.bufferY
	bufC := lbf.bufferC
	//bufPixel :=lbf.bufferPXM
	status = cl.CLSetKernelArg(lbf.kernel, 0, cl.CL_size_t(unsafe.Sizeof(bufA)), unsafe.Pointer(&bufA))
	status |= cl.CLSetKernelArg(lbf.kernel, 1, cl.CL_size_t(unsafe.Sizeof(bufX)), unsafe.Pointer(&bufX))
	status |= cl.CLSetKernelArg(lbf.kernel, 2, cl.CL_size_t(unsafe.Sizeof(bufY)), unsafe.Pointer(&bufY))
	status |= cl.CLSetKernelArg(lbf.kernel, 3, cl.CL_size_t(unsafe.Sizeof(bufC)), unsafe.Pointer(&bufC))
	//status |= cl.CLSetKernelArg(lbf.kernel, 4, cl.CL_size_t(unsafe.Sizeof(bufPixel)), unsafe.Pointer(&bufPixel))

	//status |= cl.CLSetKernelArg(lbf.kernel, 4, cl.CL_size_t(unsafe.Sizeof(maxX)), unsafe.Pointer(&maxX))
	//status |= cl.CLSetKernelArg(lbf.kernel, 5, cl.CL_size_t(unsafe.Sizeof(maxY)), unsafe.Pointer(&maxY))


	if status != cl.CL_SUCCESS {
		log.Fatalf("CLSetKernelArg failed %v", status)
	}

}


func oclFree(lbf *oclParam) {
	// Free OpenCL resources
	cl.CLReleaseKernel(lbf.kernel)
	cl.CLReleaseProgram(*lbf.program)
	cl.CLReleaseCommandQueue(lbf.cmdQueue)
	cl.CLReleaseMemObject(lbf.bufferA)
	cl.CLReleaseMemObject(lbf.bufferX)
	cl.CLReleaseMemObject(lbf.bufferY)
	cl.CLReleaseMemObject(lbf.bufferC)
	cl.CLReleaseContext(lbf.context)
}


/*
func main() {
	initTestMsg()

	//right :=&oclParam{}
	clinit(left)

	// STEP 6: Write host data to device buffers
	log.Printf("go (batch size/iter %v/%v)", batchSize, numIter)
	start := time.Now()

	for r := 0; r < numIter; r++ {
		goIt(left)
	}

	log.Print("finished");
	elapsed := time.Since(start)

	// STEP 13: Release OpenCL resources

	log.Printf("elapsed: %s", elapsed)
	log.Printf("x(1)=%v", (*left.X)[0])
	log.Printf("y(1)=%v", left.Y[0])
	log.Printf("color(b1)=%X", left.C[1])
	log.Printf("datasizeC=%v", datasizeC)
	log.Printf("throughput %v", humanize.Comma(int64(float64(batchSize)*float64(numIter)/float64(elapsed.Seconds()))))
	oclFree(left)

}
*/

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

/* Create program from a file and compile it */
func build_program(context cl.CL_context, device []cl.CL_device_id,
	filename string, options []byte) *cl.CL_program {
	var program cl.CL_program
	//var program_handle;
	var program_buffer [1][]byte
	var program_log interface{}
	var program_size [1]cl.CL_size_t
	var log_size cl.CL_size_t
	var err cl.CL_int

	/* Read each program file and place content into buffer array */
	program_handle, err1 := os.Open(filename)
	if err1 != nil {
		log.Printf("Couldn't find the program file %s\n", filename)
		return nil
	}
	defer program_handle.Close()

	fi, err2 := program_handle.Stat()
	if err2 != nil {
		log.Printf("Couldn't find the program stat\n")
		return nil
	}
	program_size[0] = cl.CL_size_t(fi.Size())
	program_buffer[0] = make([]byte, program_size[0])
	read_size, err3 := program_handle.Read(program_buffer[0])
	if err3 != nil || cl.CL_size_t(read_size) != program_size[0] {
		log.Printf("read file error or file size wrong\n")
		return nil
	}

	/* Create a program containing all program content */
	program = cl.CLCreateProgramWithSource(context, 1,
		program_buffer[:], program_size[:], &err)
	if err < 0 {
		log.Printf("Couldn't create the program\n")
	}

	/* Build program */
	err = cl.CLBuildProgram(program, 1, device[:], options, nil, nil)
	if err < 0 {
		/* Find size of log and print to std output */
		cl.CLGetProgramBuildInfo(program, device[0], cl.CL_PROGRAM_BUILD_LOG,
			0, nil, &log_size)
		cl.CLGetProgramBuildInfo(program, device[0], cl.CL_PROGRAM_BUILD_LOG,
			log_size, &program_log, nil)
		log.Printf("%s\n", program_log)
		return nil
	}

	return &program
}
