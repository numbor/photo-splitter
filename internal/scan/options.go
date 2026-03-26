package scan

type Options struct {
	DPI        int
	Brightness int
	Contrast   int
}

func (o Options) normalized() Options {
	if o.DPI <= 0 {
		o.DPI = 300
	}
	if o.DPI < 75 {
		o.DPI = 75
	}
	if o.DPI > 1200 {
		o.DPI = 1200
	}
	if o.Brightness < -1000 {
		o.Brightness = -1000
	}
	if o.Brightness > 1000 {
		o.Brightness = 1000
	}
	if o.Contrast < -1000 {
		o.Contrast = -1000
	}
	if o.Contrast > 1000 {
		o.Contrast = 1000
	}
	return o
}
