// +build cl11 cl12 cl20

package ocl

import (
	"fmt"
	"gocl/cl"
	"unsafe"
)

type event struct {
	event_id cl.CL_event
}

func (this *event) GetID() cl.CL_event {
	return this.event_id
}

func (this *event) GetInfo(param_name cl.CL_event_info) (interface{}, error) {
	/* param data */
	var param_value interface{}
	var param_size cl.CL_size_t
	var errCode cl.CL_int

	/* Find size of param data */
	if errCode = cl.CLGetEventInfo(this.event_id, param_name, 0, nil, &param_size); errCode != cl.CL_SUCCESS {
		return nil, fmt.Errorf("GetInfo failure with errcode_ret %d: %s", errCode, cl.ERROR_CODES_STRINGS[-errCode])
	}

	/* Access param data */
	if errCode = cl.CLGetEventInfo(this.event_id, param_name, param_size, &param_value, nil); errCode != cl.CL_SUCCESS {
		return nil, fmt.Errorf("GetInfo failure with errcode_ret %d: %s", errCode, cl.ERROR_CODES_STRINGS[-errCode])
	}

	return param_value, nil
}

func (this *event) Retain() error {
	if errCode := cl.CLRetainEvent(this.event_id); errCode != cl.CL_SUCCESS {
		return fmt.Errorf("Retain failure with errcode_ret %d: %s", errCode, cl.ERROR_CODES_STRINGS[-errCode])
	}
	return nil
}

func (this *event) Release() error {
	if errCode := cl.CLReleaseEvent(this.event_id); errCode != cl.CL_SUCCESS {
		return fmt.Errorf("Release failure with errcode_ret %d: %s", errCode, cl.ERROR_CODES_STRINGS[-errCode])
	}
	return nil
}

func (this *event) SetCallback(command_exec_callback_type cl.CL_int,
	pfn_notify cl.CL_evt_notify,
	user_data unsafe.Pointer) error {
	if errCode := cl.CLSetEventCallback(this.event_id, command_exec_callback_type, pfn_notify, user_data); errCode != cl.CL_SUCCESS {
		return fmt.Errorf("SetCallback failure with errcode_ret %d: %s", errCode, cl.ERROR_CODES_STRINGS[-errCode])
	} else {
		return nil
	}
}

func (this *event) SetStatus(execution_status cl.CL_int) error {
	if errCode := cl.CLSetUserEventStatus(this.event_id, execution_status); errCode != cl.CL_SUCCESS {
		return fmt.Errorf("SetStatus failure with errcode_ret %d: %s", errCode, cl.ERROR_CODES_STRINGS[-errCode])
	} else {
		return nil
	}
}

func (this *event) GetProfilingInfo(param_name cl.CL_profiling_info) (interface{}, error) {
	/* param data */
	var param_value interface{}
	var param_size cl.CL_size_t
	var errCode cl.CL_int

	/* Find size of param data */
	if errCode = cl.CLGetEventProfilingInfo(this.event_id, param_name, 0, nil, &param_size); errCode != cl.CL_SUCCESS {
		return nil, fmt.Errorf("GetProfilingInfo failure with errcode_ret %d: %s", errCode, cl.ERROR_CODES_STRINGS[-errCode])
	}

	/* Access param data */
	if errCode = cl.CLGetEventProfilingInfo(this.event_id, param_name, param_size, &param_value, nil); errCode != cl.CL_SUCCESS {
		return nil, fmt.Errorf("GetProfilingInfo failure with errcode_ret %d: %s", errCode, cl.ERROR_CODES_STRINGS[-errCode])
	}

	return param_value, nil
}

func WaitForEvents(event_list []Event) error {
	numEvents := cl.CL_uint(len(event_list))
	events := make([]cl.CL_event, numEvents)
	for i := cl.CL_uint(0); i < numEvents; i++ {
		events[i] = event_list[i].GetID()
	}

	if errCode := cl.CLWaitForEvents(numEvents, events); errCode != cl.CL_SUCCESS {
		return fmt.Errorf("WaitForEvents failure with errcode_ret %d: %s", errCode, cl.ERROR_CODES_STRINGS[-errCode])
	} else {
		return nil
	}
}
