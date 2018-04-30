// +build cl11 cl12 cl20

package ocl

import (
	"fmt"
	"gocl/cl"
)

type sampler struct {
	sampler_id cl.CL_sampler
}

func (this *sampler) GetID() cl.CL_sampler {
	return this.sampler_id
}

func (this *sampler) GetInfo(param_name cl.CL_sampler_info) (interface{}, error) {
	/* param data */
	var param_value interface{}
	var param_size cl.CL_size_t
	var errCode cl.CL_int

	/* Find size of param data */
	if errCode = cl.CLGetSamplerInfo(this.sampler_id, param_name, 0, nil, &param_size); errCode != cl.CL_SUCCESS {
		return nil, fmt.Errorf("GetInfo failure with errcode_ret %d: %s", errCode, cl.ERROR_CODES_STRINGS[-errCode])
	}

	/* Access param data */
	if errCode = cl.CLGetSamplerInfo(this.sampler_id, param_name, param_size, &param_value, nil); errCode != cl.CL_SUCCESS {
		return nil, fmt.Errorf("GetInfo failure with errcode_ret %d: %s", errCode, cl.ERROR_CODES_STRINGS[-errCode])
	}

	return param_value, nil
}

func (this *sampler) Retain() error {
	if errCode := cl.CLRetainSampler(this.sampler_id); errCode != cl.CL_SUCCESS {
		return fmt.Errorf("Retain failure with errcode_ret %d: %s", errCode, cl.ERROR_CODES_STRINGS[-errCode])
	}
	return nil
}

func (this *sampler) Release() error {
	if errCode := cl.CLReleaseSampler(this.sampler_id); errCode != cl.CL_SUCCESS {
		return fmt.Errorf("Release failure with errcode_ret %d: %s", errCode, cl.ERROR_CODES_STRINGS[-errCode])
	}
	return nil
}
