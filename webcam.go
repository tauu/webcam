// Library for working with webcams and other video capturing devices.
// It depends entirely on v4l2 framework, thus will compile and work
// only on Linux machine
package webcam

import (
	"errors"
	"fmt"
	"reflect"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Webcam object
type Webcam struct {
	fd                 uintptr
	bufcount           uint32
	buffers            [][]byte
	multiPlaneBuffers  [][][]byte
	streaming          bool
	pollFds            []unix.PollFd
	singlePlaneCapture bool
	multiPlaneCapture  bool
	useMultiPlane      bool
	numPlanes          uint32
}

type ControlID uint32

type Control struct {
	Name string
	Min  int32
	Max  int32
	Type int32
	Step int32
}

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

type DigitalVideoTimings struct {
	timings v4l2_dv_timings
}

func (dvt DigitalVideoTimings) Width() uint32 {
	return dvt.timings.bt.width
}

func (dvt DigitalVideoTimings) Height() uint32 {
	return dvt.timings.bt.height
}

// Open a webcam with a given path
// Checks if device is a v4l2 device and if it is
// capable to stream video
func Open(path string) (*Webcam, error) {
	handle, err := unix.Open(path, unix.O_RDONLY|unix.O_NONBLOCK, 0666)
	if err != nil {
		return nil, err
	}
	if handle < 0 {
		return nil, fmt.Errorf("failed to open %v", path)
	}
	// At this point the handle is valid and must be return or closed
	success := false // If this is not set true prior to function exit we assume error and close the handle
	defer func() {
		if !success {
			// Since the handle is not returned on error we must close it or leak
			unix.Close(handle)
		}
	}()
	fd := uintptr(handle)

	supportsVideoCaptureSinglePlane, supportsVideoCaptureMultiPlane, supportsVideoStreaming, err := checkCapabilities(fd)

	if err != nil {
		return nil, err
	}

	if !supportsVideoCaptureSinglePlane && !supportsVideoCaptureMultiPlane {
		return nil, errors.New("Not a video capture device")
	}

	if !supportsVideoStreaming {
		return nil, errors.New("Device does not support the streaming I/O method")
	}

	w := new(Webcam)
	w.fd = fd
	w.singlePlaneCapture = supportsVideoCaptureSinglePlane
	w.multiPlaneCapture = supportsVideoCaptureMultiPlane
	w.bufcount = 256
	w.pollFds = []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
	// Choose singe plane api by default if available.
	if supportsVideoCaptureSinglePlane {
		w.useMultiPlane = false
	} else {
		w.useMultiPlane = true
	}
	success = true
	return w, nil
}

// Returns image formats supported by the device alongside with
// their text description
// Note that this function is somewhat experimental. Frames are not ordered in
// any meaning, also duplicates can occur so it's up to developer to clean it up.
// See http://linuxtv.org/downloads/v4l-dvb-apis/vidioc-enum-framesizes.html
// for more information
func (w *Webcam) GetSupportedFormats() map[PixelFormat]string {

	result := make(map[PixelFormat]string)
	var err error
	var code uint32
	var desc string
	var index uint32

	if !w.useMultiPlane {
		for index = 0; err == nil; index++ {
			code, desc, err = getPixelFormat(w.fd, index, false)

			if err != nil {
				break
			}

			result[PixelFormat(code)] = desc
		}
	} else {
		for index = 0; err == nil; index++ {
			code, desc, err = getPixelFormat(w.fd, index, true)

			if err != nil {
				break
			}

			result[PixelFormat(code)] = desc
		}
	}

	return result
}

// GetName returns the human-readable name of the device
func (w *Webcam) GetName() (string, error) {
	return getName(w.fd)
}

// GetBusInfo returns the location of the device in the system
func (w *Webcam) GetBusInfo() (string, error) {
	return getBusInfo(w.fd)
}

// SelectInput selects the current video input.
func (w *Webcam) SelectInput(index uint32) error {
	return selectInput(w.fd, index)
}

// GetInput queries the current video input.
func (w *Webcam) GetInput() (int32, error) {
	return getInput(w.fd)
}

// GetInputs queries the list of video inputs for the current device.
func (w *Webcam) GetInputs() []Input {
	inputs := []Input{}
	for _, input := range enumInputs(w.fd) {
		inputs = append(inputs, Input{input: input})
	}
	return inputs
}

// Returns supported frame sizes for a given image format
func (w *Webcam) GetSupportedFrameSizes(f PixelFormat) []FrameSize {
	result := make([]FrameSize, 0)

	var index uint32
	var err error

	for index = 0; err == nil; index++ {
		s, err := getFrameSize(w.fd, index, uint32(f))

		if err != nil {
			break
		}

		result = append(result, s)
	}

	return result
}

// GetSupportedFramerates returns supported frame rates for a given image format and frame size.
func (w *Webcam) GetSupportedFramerates(fp PixelFormat, width uint32, height uint32) []FrameRate {
	var result []FrameRate
	var index uint32
	var err error

	// keep incrementing the index value until we get an EINVAL error
	index = 0
	for err == nil {
		r, err := getFrameInterval(w.fd, index, uint32(fp), width, height)
		if err != nil {
			break
		}
		result = append(result, r)
		index++
	}

	return result
}

// Sets desired image format and frame size
// Note, that device driver can change that values.
// Resulting values are returned by a function
// alongside with an error if any
func (w *Webcam) SetImageFormat(f PixelFormat, width, height uint32) (PixelFormat, uint32, uint32, error) {

	code := uint32(f)
	cw := width
	ch := height

	err := setImageFormat(w.fd, &code, &width, &height, w.useMultiPlane)

	if err != nil {
		return 0, 0, 0, err
	} else {
		return PixelFormat(code), cw, ch, nil
	}
}

// Get the currently set image format. For example, DV (Digital Video) devices
// may derive the format from the DV timings detected on the input and not
// support setting a specific format.
func (w *Webcam) GetImageFormat() (PixelFormat, uint32, uint32, error) {
	format, err := getImageFormat(w.fd, w.useMultiPlane)
	if err != nil {
		return 0, 0, 0, err
	}
	switch format._type {
	case V4L2_BUF_TYPE_VIDEO_CAPTURE:
		pixelFormat, err := format.pix_format()
		if err != nil {
			return 0, 0, 0, err
		}
		return PixelFormat(pixelFormat.Pixelformat), pixelFormat.Width, pixelFormat.Height, nil
	case V4L2_BUF_TYPE_VIDEO_CAPTURE_MPLANE:
		pixelFormat, err := format.pix_format_mplane()
		if err != nil {
			return 0, 0, 0, err
		}
		return PixelFormat(pixelFormat.Pixelformat), pixelFormat.Width, pixelFormat.Height, nil
	default:
		return 0, 0, 0, err
	}
}

// Retrieve the currently detected DV timings and set them to use for capturing.
// If video is captured from a DV input this has to be done before capturing and
// also every time the input source changes. After updating the timings, the
// image format is automatically updated and the format can be corresponding
// format can be received using GetImageFormat .
func (w *Webcam) UpdateDigitalVideoTimings() error {
	timings, err := queryDigitalVideoTimings(w.fd)
	if err != nil {
		return err
	}
	err = setDigitalVideoTimings(w.fd, timings)
	return err
}

// Set the number of frames to be buffered.
// Not allowed if streaming is already on.
func (w *Webcam) SetBufferCount(count uint32) error {
	if w.streaming {
		return errors.New("Cannot set buffer count when streaming")
	}
	w.bufcount = count
	return nil
}

// Switches between single plane and multi plane frame capturing.
// Not allowed if streaming is already on.
func (w *Webcam) UseMulitPlaneCapturing(use bool) error {
	if w.streaming {
		return errors.New("Cannot switch capture api when streaming")
	}
	if use && !w.multiPlaneCapture {
		return errors.New("Device does not support multi plane capturing")
	}
	if !use && !w.singlePlaneCapture {
		return errors.New("Device does not support single plane capturing")
	}
	w.useMultiPlane = use
	return nil
}

// True if the device supports capturing frames in single plane formats.
func (w *Webcam) CanCaptureSinglePlane() bool {
	return w.singlePlaneCapture
}

// True if the device supports capturing frames in multi plane formats.
func (w *Webcam) CanCaptureMultiPlane() bool {
	return w.multiPlaneCapture
}

// Get a map of available controls.
func (w *Webcam) GetControls() map[ControlID]Control {
	cmap := make(map[ControlID]Control)
	for _, c := range queryControls(w.fd) {
		cmap[ControlID(c.id)] = Control{
			Name: c.name,
			Min:  c.min,
			Max:  c.max,
			Type: int32(c.c_type),
			Step: c.step,
		}
	}
	return cmap
}

// Get the value of a control.
func (w *Webcam) GetControl(id ControlID) (int32, error) {
	return getControl(w.fd, uint32(id))
}

// Set a control.
func (w *Webcam) SetControl(id ControlID, value int32) error {
	return setControl(w.fd, uint32(id), value)
}

// Get the framerate.
func (w *Webcam) GetFramerate() (float32, error) {
	return getFramerate(w.fd)
}

// Set FPS
func (w *Webcam) SetFramerate(fps float32) error {
	return setFramerate(w.fd, 1000, uint32(1000*(fps)))
}

// Start streaming process
func (w *Webcam) StartStreaming() error {
	if w.streaming {
		return errors.New("Already streaming")
	}

	err := mmapRequestBuffers(w.fd, &w.bufcount, w.useMultiPlane)

	if err != nil {
		return errors.New("Failed to map request buffers: " + string(err.Error()))
	}

	if w.useMultiPlane {
		// The number of planes of the current capture format is required
		// for initializing the buffers.
		format, err := getImageFormat(w.fd, true)

		if err != nil {
			return errors.New("Failed to retrieve the current capture format: " + err.Error())
		}

		pixelFormat, err := format.pix_format_mplane()

		if err != nil {
			return errors.New("Failed to extract the pixel fromat form the capture format: " + err.Error())
		}

		w.multiPlaneBuffers = make([][][]byte, w.bufcount)
		w.numPlanes = uint32(pixelFormat.NumPlanes)

		for index, _ := range w.multiPlaneBuffers {
			buffer, err := mmapQueryBufferMultiPlane(w.fd, uint32(index), w.numPlanes)

			if err != nil {
				return errors.New("Failed to map memory: " + string(err.Error()))
			}

			w.multiPlaneBuffers[index] = buffer
		}

		for index, _ := range w.multiPlaneBuffers {
			_, err := mmapEnqueueBufferMultiPlane(w.fd, uint32(index), w.numPlanes)

			if err != nil {
				return errors.New("Failed to enqueue buffer: " + string(err.Error()))
			}
		}
	} else {
		w.buffers = make([][]byte, w.bufcount, w.bufcount)
		for index, _ := range w.buffers {
			var length uint32

			buffer, err := mmapQueryBuffer(w.fd, uint32(index), &length)

			if err != nil {
				return errors.New("Failed to map memory: " + string(err.Error()))
			}

			w.buffers[index] = buffer
		}

		for index := uint32(0); index < w.bufcount; index++ {
			err := mmapEnqueueBuffer(w.fd, index)

			if err != nil {
				return errors.New("Failed to enqueue buffer: " + string(err.Error()))
			}

		}
	}

	err = startStreaming(w.fd, w.useMultiPlane)

	if err != nil {
		return errors.New("Failed to start streaming: " + string(err.Error()))
	}
	w.streaming = true

	return nil
}

// Read a single frame from the webcam
// If frame cannot be read at the moment
// function will return empty slice
func (w *Webcam) ReadFrame() ([]byte, error) {
	result, index, err := w.GetFrame()
	if err == nil {
		w.ReleaseFrame(index)
	}
	return result, err
}

// Get a single frame from the webcam and return the frame and
// the buffer index. To return the buffer, ReleaseFrame must be called.
// If frame cannot be read at the moment
// function will return empty slice
func (w *Webcam) GetFrame() ([]byte, uint32, error) {
	var index uint32
	var length uint32

	err := mmapDequeueBuffer(w.fd, &index, &length)

	if err != nil {
		return nil, 0, err
	}

	return w.buffers[int(index)][:length], index, nil

}

// Get a single multi pane frame from the webcam and return the frame and
// the buffer index. To return the buffer, ReleaseFrame must be called.
// If frame cannot be read at the moment
// function will return empty slice
func (w *Webcam) GetMultiPlaneFrame() ([][]byte, uint32, error) {
	var index uint32

	lengths, err := mmapDequeueBufferMultiPlane(w.fd, &index, w.numPlanes)

	if err != nil {
		return nil, 0, err
	}

	buffer := make([][]byte, w.numPlanes)
	for i, length := range lengths {
		buffer[i] = w.multiPlaneBuffers[int(index)][i][:int(length)]
	}

	return buffer, index, nil

}

// Release the frame buffer that was obtained via GetFrame
func (w *Webcam) ReleaseFrame(index uint32) error {
	if w.useMultiPlane {
		_, err := mmapEnqueueBufferMultiPlane(w.fd, index, w.numPlanes)
		return err
	} else {
		return mmapEnqueueBuffer(w.fd, index)
	}
}

// Wait until frame could be read
func (w *Webcam) WaitForFrame(timeout uint32) error {

	count, err := waitForFrame(w.pollFds, timeout)

	if count < 0 || err != nil {
		return err
	} else if count == 0 {
		return new(Timeout)
	} else {
		return nil
	}
}

func (w *Webcam) StopStreaming() error {
	if !w.streaming {
		return errors.New("Request to stop streaming when not streaming")
	}
	w.streaming = false
	for _, buffer := range w.buffers {
		err := mmapReleaseBuffer(buffer)
		if err != nil {
			return err
		}
	}
	return stopStreaming(w.fd, w.useMultiPlane)
}

// Close the device
func (w *Webcam) Close() error {
	if w.streaming {
		w.StopStreaming()
	}

	err := unix.Close(int(w.fd))

	return err
}

// Sets automatic white balance correction
func (w *Webcam) SetAutoWhiteBalance(val bool) error {
	v := int32(0)
	if val {
		v = 1
	}
	return setControl(w.fd, V4L2_CID_AUTO_WHITE_BALANCE, v)
}

func gobytes(p unsafe.Pointer, n int) []byte {

	h := reflect.SliceHeader{uintptr(p), n, n}
	s := *(*[]byte)(unsafe.Pointer(&h))

	return s
}
