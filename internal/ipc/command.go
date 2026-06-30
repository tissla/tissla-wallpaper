package ipc

type Request struct {
	Verb   string `json:"verb"`
	Path   string `json:"path,omitempty"`
	Output string `json:"output,omitempty"`
	Mode   string `json:"mode,omitempty"`
}
