package protocol

// opcodes
const (
	zwlrLayerSurfaceV1SetSize                  = 0
	zwlrLayerSurfaceV1SetAnchor                = 1
	zwlrLayerSurfaceV1SetExlusiveZone          = 2
	zwlrLayerSurfaceV1SetMargin                = 3
	zwlrLayerSurfaceV1SetKeyboardInteractivity = 4
	zwlrLayerSurfaceV1GetPopup                 = 5

	zwlrLayerSurfaceV1AckConfigure = 6
	zwlrLayerSurfaceV1Destroy      = 7

	zwlrLayerSurfaceV1SetLayer        = 8
	zwlrLayerSurfaceV1SetExlusiveEdge = 9
)
