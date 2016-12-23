package info

const kHTTPRequest = 0x0
const kHTTPSRequest = 0x1

// TODO: impl tcp、udp、ftp request
const kTCPRequest = 0x2
const KUDPRequest = 0x3

type HTTPRequest struct {
	Addr            string   `json:"addr"`
	Method          string   `json:"method"`
	URL             string   `json:"url"`
	ProtocolVersion string   `json:"version"`
	Header          []string `json:"header"`
	Body            string   `json:"body"`
	ExtraData       []byte
}
