// +build cl12

package ocl

import (
	"fmt"
	"gocl/cl"
)

func (this *device) CreateSubDevices(properties []cl.CL_device_partition_property) ([]Device, error) {
	var numDevices cl.CL_uint
	var deviceIds []cl.CL_device_id
	var devices []Device
	var errCode cl.CL_int

	/* Determine number of connected devices */
	if errCode = cl.CLCreateSubDevices(this.device_id, properties, 0, nil, &numDevices); errCode != cl.CL_SUCCESS {
		return nil, fmt.Errorf("CreateSubDevices failure with errcode_ret %d: %s", errCode, cl.ERROR_CODES_STRINGS[-errCode])
	}

	/* Access connected devices */
	deviceIds = make([]cl.CL_device_id, numDevices)
	if errCode = cl.CLCreateSubDevices(this.device_id, properties, numDevices, deviceIds, nil); errCode != cl.CL_SUCCESS {
		return nil, fmt.Errorf("CreateSubDevices failure with errcode_ret %d: %s", errCode, cl.ERROR_CODES_STRINGS[-errCode])
	}

	devices = make([]Device, numDevices)
	for i := cl.CL_uint(0); i < numDevices; i++ {
		devices[i] = &device{deviceIds[i]}
	}

	return devices, nil
}

func (this *device) Retain() error {
	if errCode := cl.CLRetainDevice(this.device_id); errCode != cl.CL_SUCCESS {
		return fmt.Errorf("Retain failure with errcode_ret %d: %s", errCode, cl.ERROR_CODES_STRINGS[-errCode])
	}
	return nil
}

func (this *device) Release() error {
	if errCode := cl.CLReleaseDevice(this.device_id); errCode != cl.CL_SUCCESS {
		return fmt.Errorf("Release failure with errcode_ret %d: %s", errCode, cl.ERROR_CODES_STRINGS[-errCode])
	}
	return nil
}
