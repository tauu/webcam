package webcam

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"unsafe"

	"github.com/blackjack/webcam/ioctl"
	"golang.org/x/sys/unix"
)

/*
	V4L2 constants, enums, and structs are derived from the header file
	'videodev2.h'. See the following reference for more information:
	https://www.kernel.org/doc/html/latest/userspace-api/media/v4l/videodev.html#videodev2-h
*/

type controlType int

const (
	c_int controlType = iota
	c_bool
	c_menu
)

type control struct {
	id     uint32
	name   string
	c_type controlType
	step   int32
	min    int32
	max    int32
}

const (
	V4L2_CAP_VIDEO_CAPTURE             uint32 = 0x00000001
	V4L2_CAP_VIDEO_CAPTURE_MPLANE      uint32 = 0x00001000
	V4L2_CAP_STREAMING                 uint32 = 0x04000000
	V4L2_BUF_TYPE_VIDEO_CAPTURE        uint32 = 1
	V4L2_BUF_TYPE_VIDEO_CAPTURE_MPLANE uint32 = 9
	V4L2_MEMORY_MMAP                   uint32 = 1
	V4L2_FIELD_ANY                     uint32 = 0
)

const (
	V4L2_FRMSIZE_TYPE_DISCRETE   uint32 = 1
	V4L2_FRMSIZE_TYPE_CONTINUOUS uint32 = 2
	V4L2_FRMSIZE_TYPE_STEPWISE   uint32 = 3
)

const (
	V4L2_FRMIVAL_TYPE_DISCRETE   uint32 = 1
	V4L2_FRMIVAL_TYPE_CONTINUOUS uint32 = 2
	V4L2_FRMIVAL_TYPE_STEPWISE   uint32 = 3
)

const (
	V4L2_CID_BASE               uint32 = 0x00980900
	V4L2_CID_AUTO_WHITE_BALANCE uint32 = V4L2_CID_BASE + 12
	V4L2_CID_PRIVATE_BASE       uint32 = 0x08000000
)

const (
	VIDEO_MAX_PLANES uint32 = 8
)

const (
	V4L2_CTRL_TYPE_INTEGER      uint32 = 1
	V4L2_CTRL_TYPE_BOOLEAN      uint32 = 2
	V4L2_CTRL_TYPE_MENU         uint32 = 3
	V4L2_CTRL_TYPE_BUTTON       uint32 = 4
	V4L2_CTRL_TYPE_INTEGER64    uint32 = 5
	V4L2_CTRL_TYPE_CTRL_CLASS   uint32 = 6
	V4L2_CTRL_TYPE_STRING       uint32 = 7
	V4L2_CTRL_TYPE_BITMASK      uint32 = 8
	V4L2_CTRL_TYPE_INTEGER_MENU uint32 = 9

	V4L2_CTRL_COMPOUND_TYPES uint32 = 0x0100
	V4L2_CTRL_TYPE_U8        uint32 = 0x0100
	V4L2_CTRL_TYPE_U16       uint32 = 0x0101
	V4L2_CTRL_TYPE_U32       uint32 = 0x0102
)

const (
	V4L2_CTRL_FLAG_DISABLED  uint32 = 0x00000001
	V4L2_CTRL_FLAG_NEXT_CTRL uint32 = 0x80000000
)

// Input Type
const (
	V4L2_INPUT_TYPE_TUNER  uint32 = 1
	V4L2_INPUT_TYPE_CAMERA uint32 = 2
	V4L2_INPUT_TYPE_TOUCH  uint32 = 3
)

// Input Status
const (
	V4L2_IN_ST_NO_POWER  uint32 = 0x00000001
	V4L2_IN_ST_NO_SIGNAL uint32 = 0x00000002
	V4L2_IN_ST_NO_COLOR  uint32 = 0x00000004
)

// Input Capabilities
const (
	V4L2_IN_CAP_DV_TIMINGS  uint32 = 0x00000002
	V4L2_IN_CAP_STD         uint32 = 0x00000004
	V4L2_IN_CAP_NATIVE_SIZE uint32 = 0x00000008
)

// Interlaced
const (
	V4L2_DV_PROGRESSIVE uint32 = 0
	V4L2_DV_INTERLACED  uint32 = 1
)

// Polarities
const (
	V4L2_DV_VSYNC_POS_POL uint32 = 1
	V4L2_DV_HSYNC_POS_POL uint32 = 2
)

// DV Flags
const (
	V4L2_DV_FL_REDUCED_BLANKING       uint32 = 1 << 0
	V4L2_DV_FL_CAN_REDUCE_FPS         uint32 = 1 << 1
	V4L2_DV_FL_REDUCED_FPS            uint32 = 1 << 2
	V4L2_DV_FL_HALF_LINE              uint32 = 1 << 3
	V4L2_DV_FL_IS_CE_VIDEO            uint32 = 1 << 4
	V4L2_DV_FL_FIRST_FIELD_EXTRA_LINE uint32 = 1 << 5
	V4L2_DV_FL_HAS_PICTURE_ASPECT     uint32 = 1 << 6
	V4L2_DV_FL_HAS_CEA861_VIC         uint32 = 1 << 7
	V4L2_DV_FL_HAS_HDMI_VIC           uint32 = 1 << 8
	V4L2_DV_FL_CAN_DETECT_REDUCED_FPS uint32 = 1 << 9
)

// DV Standards
const (
	V4L2_DV_BT_STD_CEA861 uint32 = 1 << 0
	V4L2_DV_BT_STD_DMT    uint32 = 1 << 1
	V4L2_DV_BT_STD_CVT    uint32 = 1 << 2
	V4L2_DV_BT_STD_GTF    uint32 = 1 << 3
	V4L2_DV_BT_STD_SDI    uint32 = 1 << 4
)

// DV Timing Type
const (
	V4L2_DV_BT_656_1120 uint32 = 0
)

// EDID sizes
const (
	EDID_NUM_BLOCKS_MAX uint32 = 256
	EDID_BLOCK_SIZE     uint32 = 128
)

var (
	VIDIOC_QUERYCAP  = ioctl.IoR(uintptr('V'), 0, unsafe.Sizeof(v4l2_capability{}))
	VIDIOC_ENUM_FMT  = ioctl.IoRW(uintptr('V'), 2, unsafe.Sizeof(v4l2_fmtdesc{}))
	VIDIOC_G_FMT     = ioctl.IoRW(uintptr('V'), 4, unsafe.Sizeof(v4l2_format{}))
	VIDIOC_S_FMT     = ioctl.IoRW(uintptr('V'), 5, unsafe.Sizeof(v4l2_format{}))
	VIDIOC_REQBUFS   = ioctl.IoRW(uintptr('V'), 8, unsafe.Sizeof(v4l2_requestbuffers{}))
	VIDIOC_QUERYBUF  = ioctl.IoRW(uintptr('V'), 9, unsafe.Sizeof(v4l2_buffer{}))
	VIDIOC_QBUF      = ioctl.IoRW(uintptr('V'), 15, unsafe.Sizeof(v4l2_buffer{}))
	VIDIOC_DQBUF     = ioctl.IoRW(uintptr('V'), 17, unsafe.Sizeof(v4l2_buffer{}))
	VIDIOC_G_PARM    = ioctl.IoRW(uintptr('V'), 21, unsafe.Sizeof(v4l2_streamparm{}))
	VIDIOC_S_PARM    = ioctl.IoRW(uintptr('V'), 22, unsafe.Sizeof(v4l2_streamparm{}))
	VIDIOC_ENUMINPUT = ioctl.IoRW(uintptr('V'), 26, unsafe.Sizeof(v4l2_input{}))
	VIDIOC_G_CTRL    = ioctl.IoRW(uintptr('V'), 27, unsafe.Sizeof(v4l2_control{}))
	VIDIOC_S_CTRL    = ioctl.IoRW(uintptr('V'), 28, unsafe.Sizeof(v4l2_control{}))
	VIDIOC_QUERYCTRL = ioctl.IoRW(uintptr('V'), 36, unsafe.Sizeof(v4l2_queryctrl{}))
	VIDIOC_G_EDID    = ioctl.IoRW(uintptr('V'), 40, unsafe.Sizeof(v4l2_edid{}))
	VIDIOC_S_EDID    = ioctl.IoRW(uintptr('V'), 41, unsafe.Sizeof(v4l2_edid{}))
	//sizeof int32
	VIDIOC_STREAMON            = ioctl.IoW(uintptr('V'), 18, 4)
	VIDIOC_STREAMOFF           = ioctl.IoW(uintptr('V'), 19, 4)
	VIDIOC_G_INPUT             = ioctl.IoR(uintptr('V'), 38, 4)
	VIDIOC_S_INPUT             = ioctl.IoRW(uintptr('V'), 39, 4)
	VIDIOC_ENUM_FRAMESIZES     = ioctl.IoRW(uintptr('V'), 74, unsafe.Sizeof(v4l2_frmsizeenum{}))
	VIDIOC_ENUM_FRAMEINTERVALS = ioctl.IoRW(uintptr('V'), 75, unsafe.Sizeof(v4l2_frmivalenum{}))
	VIDIOC_S_DV_TIMINGS        = ioctl.IoRW(uintptr('V'), 87, unsafe.Sizeof(v4l2_dv_timings{}))
	VIDIOC_G_DV_TIMINGS        = ioctl.IoRW(uintptr('V'), 88, unsafe.Sizeof(v4l2_dv_timings{}))
	VIDIOC_ENUM_DV_TIMINGS     = ioctl.IoRW(uintptr('V'), 98, unsafe.Sizeof(v4l2_enum_dv_timings{}))
	VIDIOC_QUERY_DV_TIMINGS    = ioctl.IoR(uintptr('V'), 99, unsafe.Sizeof(v4l2_dv_timings{}))
	__p                        = unsafe.Pointer(uintptr(0))
	NativeByteOrder            = getNativeByteOrder()
)

type v4l2_capability struct {
	driver       [16]uint8
	card         [32]uint8
	bus_info     [32]uint8
	version      uint32
	capabilities uint32
	device_caps  uint32
	reserved     [3]uint32
}

type v4l2_fmtdesc struct {
	index       uint32
	_type       uint32
	flags       uint32
	description [32]uint8
	pixelformat uint32
	reserved    [4]uint32
}

type v4l2_frmsizeenum struct {
	index        uint32
	pixel_format uint32
	_type        uint32
	union        [24]uint8
	reserved     [2]uint32
}

type v4l2_frmsize_discrete struct {
	Width  uint32
	Height uint32
}

type v4l2_frmsize_stepwise struct {
	Min_width   uint32
	Max_width   uint32
	Step_width  uint32
	Min_height  uint32
	Max_height  uint32
	Step_height uint32
}

// v4l2_frmivalenum that contains a pixel format and size and receives a
// frame interval (time duration between frames of a video stream).
type v4l2_frmivalenum struct {
	index        uint32
	pixel_format uint32
	width        uint32
	height       uint32
	_type        uint32
	union        [24]uint8
	reserved     [2]uint32
}

// v4l2_frmival_stepwise represents the frame interval range
// as minimum, maximum, and step size intervals.
type v4l2_frmival_stepwise struct {
	min  v4l2_fract // minimum frame interval [s]
	max  v4l2_fract // maximum frame interval [s]
	step v4l2_fract // frame interval step size [s]
}

// Hack to make go compiler properly align union
type v4l2_format_aligned_union struct {
	data [200 - unsafe.Sizeof(__p)]byte
	_    unsafe.Pointer
}

type v4l2_format struct {
	_type uint32
	union v4l2_format_aligned_union
}

type v4l2_pix_format struct {
	Width        uint32
	Height       uint32
	Pixelformat  uint32
	Field        uint32
	Bytesperline uint32
	Sizeimage    uint32
	Colorspace   uint32
	Priv         uint32
	Flags        uint32
	Ycbcr_enc    uint32
	Quantization uint32
	Xfer_func    uint32
}

type v4l2_pix_format_mplane struct {
	Width        uint32
	Height       uint32
	Pixelformat  uint32
	Field        uint32
	Colorspace   uint32
	PlaneFmt     [VIDEO_MAX_PLANES]v4l2_plane_pix_format
	NumPlanes    uint8
	Flags        uint8
	Ycbcr_enc    uint8
	Quantization uint8
	Xfer_func    uint8
	Reserved     [7]uint8
}

type v4l2_plane_pix_format struct {
	Sizeimage    uint32
	Bytesperline uint32
	Reserved     [6]uint16
}

func (format v4l2_format) pix_format() (v4l2_pix_format, error) {
	pixFormat := v4l2_pix_format{}
	if format._type != V4L2_BUF_TYPE_VIDEO_CAPTURE {
		return pixFormat, errors.New("the type of the format is not V4L2_BUF_TYPE_VIDEO_CAPTURE and the payload of the format is not a v4l2_pix_format structure")
	}
	buf := bytes.NewBuffer(format.union.data[:])
	err := binary.Read(buf, NativeByteOrder, &pixFormat)
	return pixFormat, err
}

func (format v4l2_format) pix_format_mplane() (v4l2_pix_format_mplane, error) {
	pixFormat := v4l2_pix_format_mplane{}
	if format._type != V4L2_BUF_TYPE_VIDEO_CAPTURE_MPLANE {
		return pixFormat, errors.New("the type of the format is not V4L2_BUF_TYPE_VIDEO_CAPTURE_MPLANE and the payload of the format is not a v4l2_pix_format_mplane structure")
	}
	buf := bytes.NewBuffer(format.union.data[:])
	err := binary.Read(buf, NativeByteOrder, &pixFormat)
	return pixFormat, err
}

type v4l2_requestbuffers struct {
	count    uint32
	_type    uint32
	memory   uint32
	reserved [2]uint32
}

type v4l2_buffer struct {
	index     uint32
	_type     uint32
	bytesused uint32
	flags     uint32
	field     uint32
	timestamp unix.Timeval
	timecode  v4l2_timecode
	sequence  uint32
	memory    uint32
	union     [unsafe.Sizeof(__p)]uint8
	length    uint32
	reserved2 uint32
	reserved  uint32
}

func (b *v4l2_buffer) initPlanes(numPlanes uint32) []v4l2_plane {
	// Initialize a planes array and write a pointer to it into the union.
	b.length = numPlanes
	planes := make([]v4l2_plane, numPlanes)
	planesPointer := uintptr(unsafe.Pointer(&planes[0]))
	pointerBytes := *(*[unsafe.Sizeof(planesPointer)]byte)(unsafe.Pointer(&planesPointer))
	copy(b.union[:], pointerBytes[:])

	return planes
}

type v4l2_plane struct {
	bytesused  uint32
	length     uint32
	union      [unsafe.Sizeof(__p)]uint8
	dataOffset uint32
	reserved   [11]uint32
}

type v4l2_timecode struct {
	_type    uint32
	flags    uint32
	frames   uint8
	seconds  uint8
	minutes  uint8
	hours    uint8
	userbits [4]uint8
}

type v4l2_queryctrl struct {
	id            uint32
	_type         uint32
	name          [32]uint8
	minimum       int32
	maximum       int32
	step          int32
	default_value int32
	flags         uint32
	reserved      [2]uint32
}

type v4l2_control struct {
	id    uint32
	value int32
}

type v4l2_fract struct {
	Numerator   uint32
	Denominator uint32
}

type v4l2_streamparm_union struct {
	capability     uint32
	output_mode    uint32
	time_per_frame v4l2_fract
	extended_mode  uint32
	buffers        uint32
	reserved       [4]uint32
	data           [200 - (10 * unsafe.Sizeof(uint32(0)))]byte
}

type v4l2_streamparm struct {
	_type uint32
	union v4l2_streamparm_union
}

type v4l2_std_id uint64

type v4l2_input struct {
	Index        uint32
	Name         [32]uint8
	Type         uint32
	Audioset     uint32
	Tuner        uint32
	Std          v4l2_std_id
	Status       uint32
	Capabilities uint32
	reserved     [3]uint32
}

type v4l2_enum_dv_timings struct {
	index    uint32
	pad      uint32
	reserved [2]uint32
	timings  v4l2_dv_timings
}

type v4l2_dv_timings struct {
	_type uint32
	union [32 * 4]uint8
	// Currently (2025-03-17) the union can only containg the v4l2_bt_timings
	// structure, but this structure is one uint32 smaller than the union.
	// Therefore embedding the structure intead of the union would break the
	// ioctl call as the full size of v4l2_dv_timings does not match the
	// expected size. To handle this in a future proof way, the union is
	// represented as the byte array and afterwards written to the
	// v4l2_bt_timings structure.
	//bt    v4l2_bt_timings
}

func (timings v4l2_dv_timings) v4l2_bt_timings() (v4l2_bt_timings, error) {
	btTimings := v4l2_bt_timings{}
	if timings._type != V4L2_DV_BT_656_1120 {
		return btTimings, fmt.Errorf("the type of the timings %d not supported, currently only not V4L2_DV_BT_656_1120 is supported", timings._type)
	}
	buf := bytes.NewBuffer(timings.union[:])
	err := binary.Read(buf, NativeByteOrder, &btTimings)
	return btTimings, err
}

type v4l2_bt_timings struct {
	Width          uint32
	Height         uint32
	Interlaced     uint32
	Polarities     uint32
	Pixelclock     uint64
	Hfrontporch    uint32
	Hsync          uint32
	Hbackporch     uint32
	Vfrontporch    uint32
	Vsync          uint32
	Vbackporch     uint32
	Il_vfrontporch uint32
	Il_vsync       uint32
	Il_vbackporch  uint32
	Standards      uint32
	Flags          uint32
	Picture_aspect v4l2_fract
	Cea861_vic     uint8
	Hdmi_vic       uint8
	Reserved       [46]uint8
}

type v4l2_edid struct {
	pad         uint32
	start_block uint32
	blocks      uint32
	reserved    [5]uint32
	edid        [unsafe.Sizeof(__p)]byte
}

func getCapability(fd uintptr) (caps v4l2_capability, err error) {
	err = ioctl.Ioctl(fd, VIDIOC_QUERYCAP, uintptr(unsafe.Pointer(&caps)))
	return
}

func getPixelFormat(fd uintptr, index uint32, mplane bool) (code uint32, description string, err error) {

	fmtdesc := &v4l2_fmtdesc{}

	fmtdesc.index = index
	if mplane {
		fmtdesc._type = V4L2_BUF_TYPE_VIDEO_CAPTURE_MPLANE
	} else {
		fmtdesc._type = V4L2_BUF_TYPE_VIDEO_CAPTURE
	}

	err = ioctl.Ioctl(fd, VIDIOC_ENUM_FMT, uintptr(unsafe.Pointer(fmtdesc)))

	if err != nil {
		return
	}

	code = fmtdesc.pixelformat
	description = CToGoString(fmtdesc.description[:])

	return
}

func getFrameSize(fd uintptr, index uint32, code uint32) (frameSize FrameSize, err error) {

	frmsizeenum := &v4l2_frmsizeenum{}
	frmsizeenum.index = index
	frmsizeenum.pixel_format = code

	err = ioctl.Ioctl(fd, VIDIOC_ENUM_FRAMESIZES, uintptr(unsafe.Pointer(frmsizeenum)))

	if err != nil {
		return
	}

	switch frmsizeenum._type {

	case V4L2_FRMSIZE_TYPE_DISCRETE:
		discrete := &v4l2_frmsize_discrete{}
		err = binary.Read(bytes.NewBuffer(frmsizeenum.union[:]), NativeByteOrder, discrete)

		if err != nil {
			return
		}

		frameSize.MinWidth = discrete.Width
		frameSize.MaxWidth = discrete.Width
		frameSize.StepWidth = 0
		frameSize.MinHeight = discrete.Height
		frameSize.MaxHeight = discrete.Height
		frameSize.StepHeight = 0

	case V4L2_FRMSIZE_TYPE_CONTINUOUS:

	case V4L2_FRMSIZE_TYPE_STEPWISE:
		stepwise := &v4l2_frmsize_stepwise{}
		err = binary.Read(bytes.NewBuffer(frmsizeenum.union[:]), NativeByteOrder, stepwise)

		if err != nil {
			return
		}

		frameSize.MinWidth = stepwise.Min_width
		frameSize.MaxWidth = stepwise.Max_width
		frameSize.StepWidth = stepwise.Step_width
		frameSize.MinHeight = stepwise.Min_height
		frameSize.MaxHeight = stepwise.Max_height
		frameSize.StepHeight = stepwise.Step_height
	}

	return
}

func getName(fd uintptr) (string, error) {
	var caps v4l2_capability
	if err := ioctl.Ioctl(fd, VIDIOC_QUERYCAP, uintptr(unsafe.Pointer(&caps))); err != nil {
		return "", err
	}

	return CToGoString(caps.card[:]), nil
}

func getFrameInterval(fd uintptr, index uint32, code uint32, width uint32, height uint32) (FrameRate, error) {
	frmivalEnum := &v4l2_frmivalenum{
		index:        index,
		pixel_format: code,
		width:        width,
		height:       height,
	}

	if err := ioctl.Ioctl(fd, VIDIOC_ENUM_FRAMEINTERVALS, uintptr(unsafe.Pointer(frmivalEnum))); err != nil {
		return FrameRate{}, err
	}

	switch frmivalEnum._type {

	case V4L2_FRMIVAL_TYPE_DISCRETE:
		discrete := &v4l2_fract{}
		if err := binary.Read(bytes.NewBuffer(frmivalEnum.union[:]), NativeByteOrder, discrete); err != nil {
			return FrameRate{}, err
		}
		return FrameRate{
			MinDenominator:  discrete.Denominator,
			MaxDenominator:  discrete.Denominator,
			StepDenominator: 0,
			MinNumerator:    discrete.Numerator,
			MaxNumerator:    discrete.Numerator,
			StepNumerator:   0,
		}, nil

	case V4L2_FRMIVAL_TYPE_CONTINUOUS:
		return FrameRate{}, fmt.Errorf("V4L2_FRMIVAL_TYPE_CONTINUOUS not implemented")

	case V4L2_FRMIVAL_TYPE_STEPWISE:
		stepwise := &v4l2_frmival_stepwise{}
		if err := binary.Read(bytes.NewBuffer(frmivalEnum.union[:]), NativeByteOrder, stepwise); err != nil {
			return FrameRate{}, err
		}
		return FrameRate{
			MinDenominator:  stepwise.min.Denominator,
			MaxDenominator:  stepwise.max.Denominator,
			StepDenominator: stepwise.step.Denominator,
			MinNumerator:    stepwise.min.Numerator,
			MaxNumerator:    stepwise.max.Numerator,
			StepNumerator:   stepwise.step.Numerator,
		}, nil
	}

	return FrameRate{}, fmt.Errorf("unknown frame interval type")
}

func getBusInfo(fd uintptr) (string, error) {
	var caps v4l2_capability
	if err := ioctl.Ioctl(fd, VIDIOC_QUERYCAP, uintptr(unsafe.Pointer(&caps))); err != nil {
		return "", err
	}

	return CToGoString(caps.bus_info[:]), nil
}

func getImageFormat(fd uintptr, multiPlane bool) (format v4l2_format, err error) {
	if multiPlane {
		format._type = V4L2_BUF_TYPE_VIDEO_CAPTURE_MPLANE
	} else {
		format._type = V4L2_BUF_TYPE_VIDEO_CAPTURE
	}
	err = ioctl.Ioctl(fd, VIDIOC_G_FMT, uintptr(unsafe.Pointer(&format)))
	return
}

func setImageFormat(fd uintptr, formatcode *uint32, width *uint32, height *uint32, multiPlane bool) (err error) {

	format := &v4l2_format{}
	pixbytes := &bytes.Buffer{}

	if multiPlane {
		format._type = V4L2_BUF_TYPE_VIDEO_CAPTURE_MPLANE
		pix := v4l2_pix_format_mplane{
			Width:       *width,
			Height:      *height,
			Pixelformat: *formatcode,
			Field:       V4L2_FIELD_ANY,
		}

		err = binary.Write(pixbytes, NativeByteOrder, pix)

		if err != nil {
			return
		}
	} else {
		format._type = V4L2_BUF_TYPE_VIDEO_CAPTURE
		pix := v4l2_pix_format{
			Width:       *width,
			Height:      *height,
			Pixelformat: *formatcode,
			Field:       V4L2_FIELD_ANY,
		}

		err = binary.Write(pixbytes, NativeByteOrder, pix)

		if err != nil {
			return
		}
	}

	copy(format.union.data[:], pixbytes.Bytes())

	err = ioctl.Ioctl(fd, VIDIOC_S_FMT, uintptr(unsafe.Pointer(format)))

	if err != nil {
		return
	}

	if multiPlane {
		pixReverse := &v4l2_pix_format_mplane{}
		err = binary.Read(bytes.NewBuffer(format.union.data[:]), NativeByteOrder, pixReverse)

		if err != nil {
			return
		}

		*width = pixReverse.Width
		*height = pixReverse.Height
		*formatcode = pixReverse.Pixelformat
	} else {
		pixReverse := &v4l2_pix_format{}
		err = binary.Read(bytes.NewBuffer(format.union.data[:]), NativeByteOrder, pixReverse)

		if err != nil {
			return
		}

		*width = pixReverse.Width
		*height = pixReverse.Height
		*formatcode = pixReverse.Pixelformat
	}

	return

}

func mmapRequestBuffers(fd uintptr, buf_count *uint32, multiPlane bool) (err error) {

	req := &v4l2_requestbuffers{}
	req.count = *buf_count
	if multiPlane {
		req._type = V4L2_BUF_TYPE_VIDEO_CAPTURE_MPLANE
	} else {
		req._type = V4L2_BUF_TYPE_VIDEO_CAPTURE
	}
	req.memory = V4L2_MEMORY_MMAP

	err = ioctl.Ioctl(fd, VIDIOC_REQBUFS, uintptr(unsafe.Pointer(req)))

	if err != nil {
		return
	}

	*buf_count = req.count

	return

}

func mmapQueryBuffer(fd uintptr, index uint32, length *uint32) (buffer []byte, err error) {

	req := &v4l2_buffer{}

	req._type = V4L2_BUF_TYPE_VIDEO_CAPTURE
	req.memory = V4L2_MEMORY_MMAP
	req.index = index

	err = ioctl.Ioctl(fd, VIDIOC_QUERYBUF, uintptr(unsafe.Pointer(req)))

	if err != nil {
		return
	}

	var offset uint32
	err = binary.Read(bytes.NewBuffer(req.union[:]), NativeByteOrder, &offset)

	if err != nil {
		return
	}

	*length = req.length

	buffer, err = unix.Mmap(int(fd), int64(offset), int(req.length), unix.PROT_READ, unix.MAP_SHARED)
	return
}

func mmapQueryBufferMultiPlane(fd uintptr, index uint32, numPlanes uint32) (buffers [][]byte, err error) {

	req := &v4l2_buffer{}

	req._type = V4L2_BUF_TYPE_VIDEO_CAPTURE_MPLANE
	req.memory = V4L2_MEMORY_MMAP
	req.index = index
	planes := req.initPlanes(numPlanes)

	err = ioctl.Ioctl(fd, VIDIOC_QUERYBUF, uintptr(unsafe.Pointer(req)))

	if err != nil {
		return
	}

	buffers = make([][]byte, req.length)
	for i, plane := range planes {
		var offset uint32
		err = binary.Read(bytes.NewBuffer(plane.union[:]), NativeByteOrder, &offset)

		if err != nil {
			return
		}

		length := plane.length
		buffers[i], err = unix.Mmap(int(fd), int64(offset), int(length), unix.PROT_READ, unix.MAP_SHARED)

		if err != nil {
			return
		}
	}

	return
}

func mmapDequeueBuffer(fd uintptr, index *uint32, length *uint32) (err error) {

	buffer := &v4l2_buffer{}

	buffer._type = V4L2_BUF_TYPE_VIDEO_CAPTURE
	buffer.memory = V4L2_MEMORY_MMAP

	err = ioctl.Ioctl(fd, VIDIOC_DQBUF, uintptr(unsafe.Pointer(buffer)))

	if err != nil {
		return
	}

	*index = buffer.index
	*length = buffer.bytesused

	return

}

func mmapDequeueBufferMultiPlane(fd uintptr, index *uint32, numPlanes uint32) ([]uint32, error) {

	buffer := &v4l2_buffer{}

	buffer._type = V4L2_BUF_TYPE_VIDEO_CAPTURE_MPLANE
	buffer.memory = V4L2_MEMORY_MMAP
	// Initialize an array that will hold the number of bytes used in each plane.
	lengths := make([]uint32, numPlanes)
	planes := buffer.initPlanes(numPlanes)

	err := ioctl.Ioctl(fd, VIDIOC_DQBUF, uintptr(unsafe.Pointer(buffer)))

	if err != nil {
		return lengths, err
	}

	*index = buffer.index
	for i, plane := range planes {
		lengths[i] = plane.bytesused
	}

	return lengths, nil

}

func mmapEnqueueBuffer(fd uintptr, index uint32) (err error) {

	buffer := &v4l2_buffer{}

	buffer._type = V4L2_BUF_TYPE_VIDEO_CAPTURE
	buffer.memory = V4L2_MEMORY_MMAP
	buffer.index = index

	err = ioctl.Ioctl(fd, VIDIOC_QBUF, uintptr(unsafe.Pointer(buffer)))
	return

}

func mmapEnqueueBufferMultiPlane(fd uintptr, index uint32, numPlanes uint32) ([]uint32, error) {

	buffer := &v4l2_buffer{}

	buffer._type = V4L2_BUF_TYPE_VIDEO_CAPTURE_MPLANE
	buffer.memory = V4L2_MEMORY_MMAP
	buffer.index = index

	// Initialize an array that will hold the number of bytes used in each plane.
	lengths := make([]uint32, numPlanes)
	planes := buffer.initPlanes(numPlanes)

	err := ioctl.Ioctl(fd, VIDIOC_QBUF, uintptr(unsafe.Pointer(buffer)))

	if err != nil {
		return lengths, err
	}

	// The length are actually not actually useful, this is just added, to
	// (hopefully) prevent the planes array from being garbage collected. The
	// pointer to the planes array in the buffer has to be valid in the
	// systemcall, so the planes array may not be garbage collected.
	for i, plane := range planes {
		lengths[i] = plane.length
	}
	return lengths, err

}

func mmapReleaseBuffer(buffer []byte) (err error) {
	err = unix.Munmap(buffer)
	return
}

func startStreaming(fd uintptr, multiPlane bool) (err error) {

	var uintPointer uint32 = V4L2_BUF_TYPE_VIDEO_CAPTURE
	if multiPlane {
		uintPointer = V4L2_BUF_TYPE_VIDEO_CAPTURE_MPLANE
	}
	err = ioctl.Ioctl(fd, VIDIOC_STREAMON, uintptr(unsafe.Pointer(&uintPointer)))
	return

}

func stopStreaming(fd uintptr, multiPlane bool) (err error) {

	var uintPointer uint32 = V4L2_BUF_TYPE_VIDEO_CAPTURE
	if multiPlane {
		uintPointer = V4L2_BUF_TYPE_VIDEO_CAPTURE_MPLANE
	}
	err = ioctl.Ioctl(fd, VIDIOC_STREAMOFF, uintptr(unsafe.Pointer(&uintPointer)))
	return

}

func waitForFrame(pollFds []unix.PollFd, timeout uint32) (count int, err error) {
	for {
		count, err = unix.Poll(pollFds, int(timeout*1000))
		if count < 0 && err == unix.EINTR {
			continue
		}
		return
	}

}

func getControl(fd uintptr, id uint32) (int32, error) {
	ctrl := &v4l2_control{}
	ctrl.id = id
	err := ioctl.Ioctl(fd, VIDIOC_G_CTRL, uintptr(unsafe.Pointer(ctrl)))
	return ctrl.value, err
}

func setControl(fd uintptr, id uint32, val int32) error {
	ctrl := &v4l2_control{}
	ctrl.id = id
	ctrl.value = val
	return ioctl.Ioctl(fd, VIDIOC_S_CTRL, uintptr(unsafe.Pointer(ctrl)))
}

func getInput(fd uintptr) (index int32, err error) {
	err = ioctl.Ioctl(fd, VIDIOC_G_INPUT, uintptr(unsafe.Pointer(&index)))
	return
}

func selectInput(fd uintptr, index uint32) (err error) {
	err = ioctl.Ioctl(fd, VIDIOC_S_INPUT, uintptr(unsafe.Pointer(&index)))
	return
}

func enumInputs(fd uintptr) (inputs []v4l2_input) {
	var err error
	for index := uint32(0); err == nil; index++ {
		input := v4l2_input{Index: index}
		err = ioctl.Ioctl(fd, VIDIOC_ENUMINPUT, uintptr(unsafe.Pointer(&input)))
		if err == nil {
			inputs = append(inputs, input)
		}
	}
	return
}

func getDigitalVideoTimings(fd uintptr) (v4l2_dv_timings, error) {
	timings := v4l2_dv_timings{}
	err := ioctl.Ioctl(fd, VIDIOC_G_DV_TIMINGS, uintptr(unsafe.Pointer(&timings)))
	return timings, err
}

func setDigitalVideoTimings(fd uintptr, timings v4l2_dv_timings) error {
	err := ioctl.Ioctl(fd, VIDIOC_S_DV_TIMINGS, uintptr(unsafe.Pointer(&timings)))
	return err
}

func enumDigitalVideoTimings(fd uintptr) (all []v4l2_enum_dv_timings) {
	var err error
	for index := uint32(0); err == nil; index++ {
		timings := v4l2_enum_dv_timings{index: index}
		err = ioctl.Ioctl(fd, VIDIOC_ENUM_DV_TIMINGS, uintptr(unsafe.Pointer(&timings)))
		if err == nil {
			all = append(all, timings)
		}
	}
	return
}

func queryDigitalVideoTimings(fd uintptr) (v4l2_dv_timings, error) {
	timings := v4l2_dv_timings{}
	err := ioctl.Ioctl(fd, VIDIOC_QUERY_DV_TIMINGS, uintptr(unsafe.Pointer(&timings)))
	return timings, err
}

func getFramerate(fd uintptr) (float32, error) {
	param := &v4l2_streamparm{}
	param._type = V4L2_BUF_TYPE_VIDEO_CAPTURE

	err := ioctl.Ioctl(fd, VIDIOC_G_PARM, uintptr(unsafe.Pointer(param)))
	if err != nil {
		return 0, err
	}
	tf := param.union.time_per_frame
	if tf.Denominator == 0 || tf.Numerator == 0 {
		return 0, fmt.Errorf("Invalid framerate (%d/%d)", tf.Denominator, tf.Numerator)
	}
	return float32(tf.Denominator) / float32(tf.Numerator), nil
}

func setFramerate(fd uintptr, num, denom uint32) error {
	param := &v4l2_streamparm{}
	param._type = V4L2_BUF_TYPE_VIDEO_CAPTURE
	param.union.time_per_frame.Numerator = num
	param.union.time_per_frame.Denominator = denom
	return ioctl.Ioctl(fd, VIDIOC_S_PARM, uintptr(unsafe.Pointer(param)))
}

func queryControls(fd uintptr) []control {
	controls := []control{}
	var err error
	// Don't use V42L_CID_BASE since it is the same as brightness.
	var id uint32
	for err == nil {
		id |= V4L2_CTRL_FLAG_NEXT_CTRL
		query := &v4l2_queryctrl{}
		query.id = id
		err = ioctl.Ioctl(fd, VIDIOC_QUERYCTRL, uintptr(unsafe.Pointer(query)))
		id = query.id
		if err == nil {
			if (query.flags & V4L2_CTRL_FLAG_DISABLED) != 0 {
				continue
			}
			var c control
			switch query._type {
			default:
				continue
			case V4L2_CTRL_TYPE_INTEGER, V4L2_CTRL_TYPE_INTEGER64:
				c.c_type = c_int
			case V4L2_CTRL_TYPE_BOOLEAN:
				c.c_type = c_bool
			case V4L2_CTRL_TYPE_MENU:
				c.c_type = c_menu
			}
			c.id = id
			c.name = CToGoString(query.name[:])
			c.min = query.minimum
			c.max = query.maximum
			c.step = query.step
			controls = append(controls, c)
		}
	}
	return controls
}

// getEdid retrieves the edid data of the input with the given index. The index
// of a device is reported by enumInputs.
func getEdid(fd uintptr, index uint32) ([]byte, error) {
	edid := v4l2_edid{
		pad:         index,
		start_block: 0,
		blocks:      EDID_NUM_BLOCKS_MAX,
	}

	// Create and set array to hold the edid data. The maximum possible size of
	// and edid ist used to ensure the whole edid is returned.
	var edidData [EDID_NUM_BLOCKS_MAX * EDID_BLOCK_SIZE]byte
	edidPointer := uintptr(unsafe.Pointer(&edidData[0]))
	pointerBytes := *(*[unsafe.Sizeof(edidPointer)]byte)(unsafe.Pointer(&edidPointer))
	copy(edid.edid[:], pointerBytes[:])

	err := ioctl.Ioctl(fd, VIDIOC_G_EDID, uintptr(unsafe.Pointer(&edid)))
	if err != nil {
		return edidData[:0], nil
	}

	// The drive should set the number of block actually used by the edid.
	return edidData[:edid.blocks*EDID_BLOCK_SIZE], nil
}

// setEdid updates the edid data of the input with the given index. The index
// of a device is reported by enumInputs.
func setEdid(fd uintptr, index uint32, data []byte) error {
	edid := v4l2_edid{
		pad:         index,
		start_block: 0,
		blocks:      uint32(len(data) / int(EDID_BLOCK_SIZE)),
	}

	// Create and set array to hold the edid data. The maximum possible size of
	// and edid ist used to ensure the whole edid is returned.
	edidPointer := uintptr(unsafe.Pointer(&data[0]))
	pointerBytes := *(*[unsafe.Sizeof(edidPointer)]byte)(unsafe.Pointer(&edidPointer))
	copy(edid.edid[:], pointerBytes[:])

	err := ioctl.Ioctl(fd, VIDIOC_G_EDID, uintptr(unsafe.Pointer(&edid)))
	return err
}

func getNativeByteOrder() binary.ByteOrder {
	var i int32 = 0x01020304
	u := unsafe.Pointer(&i)
	pb := (*byte)(u)
	b := *pb
	if b == 0x04 {
		return binary.LittleEndian
	} else {
		return binary.BigEndian
	}
}

func CToGoString(c []byte) string {
	n := -1
	for i, b := range c {
		if b == 0 {
			break
		}
		n = i
	}
	return string(c[:n+1])
}
