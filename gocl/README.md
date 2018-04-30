gocl
====

Go OpenCL (GOCL) Binding (http://www.gocl.org)


Library documentation: 

http://www.khronos.org/registry/cl/sdk/1.1/docs/man/xhtml/

http://www.khronos.org/registry/cl/sdk/1.2/docs/man/xhtml/

http://www.khronos.org/registry/cl/sdk/2.0/docs/man/xhtml/

In order to build this, make sure you have the required drivers and SDK installed for your graphics card. You will need at least opencl.lib from Intel/AMD/NVIDIA:

http://software.intel.com/en-us/intel-opencl

http://developer.amd.com/tools-and-sdks/opencl-zone/

https://developer.nvidia.com/opencl


The locations of the library and include file can be supplied by way of environment variables, for example: 

export CGO_LDFLAGS=-L$OPENCLSDKROOT/lib/x86     			(or null for NVIDIA and Mac OSX)

export CGO_CFLAGS=-I$GOPATH/src/gocl/android/include     	(gocl/android/include/CL have the latest OpenCL 2.0 include files from https://www.khronos.org/registry/cl/)

===============================================

To build OpenCL 1.1/1.2/2.0 compliance C-style binding (replacing "clxx" with "cl11"/"cl12"/"cl20"):

go build -tags="clxx" gocl/cl

go test -v -tags="clxx" gocl/cl_test

go install -tags="cl11" gocl/cl_demo/opencl11/ch(x)   		(Examples in "OpenCL in Action")

go install -tags="cl12" gocl/cl_demo/opencl12/chapter(x)	(Examples in "Heterogeneous Computing with OpenCL, 2nd Edition")

go install -tags="cl20" gocl/cl_demo/opencl20/x				(Examples in "AMDAPPSDK and INTELOCLSDK")

===============================================

To build OpenCL 1.1/1.2/2.0 compliance OO-style binding (replacing "clxx" with "cl11"/"cl12"/"cl20"):

go build -tags="clxx" gocl/ocl

go test -v -tags="clxx" gocl/ocl_test