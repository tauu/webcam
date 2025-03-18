package webcam

// An input of the webcam which can be used for capturing video.
type Input struct {
	input v4l2_input
}

// The index of the input, which is required to select it.
func (i Input) Index() uint32 {
	return i.input.Index
}

func (i Input) Name() string {
	return CToGoString(i.input.Name[:])
}

func (i Input) IsCamera() bool {
	return i.input.Type&V4L2_INPUT_TYPE_CAMERA != 0
}

func (i Input) NoPower() bool {
	return i.input.Status&V4L2_IN_ST_NO_POWER != 0
}

func (i Input) NoSignal() bool {
	return i.input.Status&V4L2_IN_ST_NO_SIGNAL != 0
}

// If true the input support digital video timings, which likely also means that
// it does not support setting framesizes and capture rates.
func (i Input) SupportsDigitalVideoTinings() bool {
	return i.input.Capabilities&V4L2_IN_CAP_DV_TIMINGS != 0
}
