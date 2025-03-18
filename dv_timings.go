package webcam

import "strings"

type DigitalVideoTimings struct {
	timings   v4l2_dv_timings
	btTimings v4l2_bt_timings
}

type DVStandards struct {
	CEA861 bool
	DMT    bool
	CVT    bool
	GTF    bool
	SDI    bool
}

func (s DVStandards) String() string {
	names := []string{}
	if s.CEA861 {
		names = append(names, "CEA681")
	}
	if s.DMT {
		names = append(names, "DMT")
	}
	if s.CVT {
		names = append(names, "CVT")
	}
	if s.GTF {
		names = append(names, "GTF")
	}
	if s.SDI {
		names = append(names, "SDI")
	}
	return strings.Join(names, " ")
}

func (dvt DigitalVideoTimings) Width() uint32 {
	return dvt.btTimings.Width
}

func (dvt DigitalVideoTimings) Height() uint32 {
	return dvt.btTimings.Height
}

func (dvt DigitalVideoTimings) RefreshRate() float64 {
	width := dvt.btTimings.Width +
		dvt.btTimings.Hfrontporch +
		dvt.btTimings.Hbackporch +
		dvt.btTimings.Hsync
	height := dvt.btTimings.Height +
		dvt.btTimings.Vfrontporch +
		dvt.btTimings.Vbackporch +
		dvt.btTimings.Vsync
	return float64(dvt.btTimings.Pixelclock) /
		float64(width) / float64(height)
}

func (dvt DigitalVideoTimings) Interlaced() bool {
	return dvt.btTimings.Interlaced&V4L2_DV_INTERLACED != 0
}

func (dvt DigitalVideoTimings) Standards() DVStandards {
	return DVStandards{
		CEA861: dvt.btTimings.Standards&V4L2_DV_BT_STD_CEA861 != 0,
		DMT:    dvt.btTimings.Standards&V4L2_DV_BT_STD_DMT != 0,
		CVT:    dvt.btTimings.Standards&V4L2_DV_BT_STD_CVT != 0,
		SDI:    dvt.btTimings.Standards&V4L2_DV_BT_STD_SDI != 0,
		GTF:    dvt.btTimings.Standards&V4L2_DV_BT_STD_GTF != 0,
	}
}
