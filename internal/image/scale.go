package image

import "fmt"

type ScaleMode int

const (
	ScaleFill ScaleMode = iota
	ScaleFit
	ScaleStretch
)

func (s ScaleMode) String() string {
	switch s {
	case ScaleFill:
		return "fill"
	case ScaleFit:
		return "fit"
	case ScaleStretch:
		return "stretch"
	default:
		return "fill"
	}
}

func ParseScaleMode(s string) (ScaleMode, error) {
	switch s {
	case "fill":
		return ScaleFill, nil
	case "fit":
		return ScaleFit, nil
	case "stretch":
		return ScaleStretch, nil
	default:
		return ScaleFill, fmt.Errorf("unknown scale mode %q, valid: fill, fit, stretch", s)
	}
}
