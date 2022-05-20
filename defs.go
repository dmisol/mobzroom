package mobzroom

type ApiPayload struct {
	Action string      `json:"action"`
	Data   interface{} `json:"data"`
}

type SessionInfo struct {
	UserId    string `json:"userId"`
	SessionId string `json:"sessionId"`
	// Session category
	Category string `json:"category"` //"stream" | "screenshare",
	// Device type
	Host     string `json:"host"` // "iOS" | "iOS_EX" | "Android" | "Web" | "Web_Studio",
	DeviceId string `json:"deviceId"`
}

type Op struct {
	RoomCreating bool `json:"roomCreating"`
	UserFaking   bool `json:"userFaking"`
}

type DataCommon struct {
	S string `json:"s"` // "JOIN" etc

	Ch  string      `json:"ch,omitempty"` // "m2m" | "p2p" | "a2m" | "p2m", // room type
	Ss  SessionInfo `json:"ss,omitempty"`
	Rid string      `json:"rid,omitempty"` // room id

}

type DataAck struct {
	S string   `json:"s"` // "JOIN" etc
	C int      `json:"c,omitempty"`
	B RoomInfo `json:"b,omitempty"`
}

type Data struct {
	DataCommon

	Pt string `json:"pt,omitempty"` // "publisher" | "viewer",
	Op `json:"op,omitempty"`

	Ve bool `json:"ve"`
	Ae bool `json:"ae"`

	Eg string `json:"eg,omitempty"`
}

type DataHb struct {
	DataCommon

	Bid string `json:"bid"`
	Ack bool   `json:"send_ack"`
}

type DataWrtc struct {
	DataCommon

	Sdp     string `json:"offer"`
	*WrtcOp `json:"op,omitempty"`
	To      interface{} `json:"to,omitempty"`
}

type WrtcOp struct {
	Restart bool `json:"restarting"`
	Hls     bool `json:"hlsForwarding"`
}

type RoomInfo struct {
	Pps  []RoomParticipant `json:"pps,omitempty"`
	Rid  string            `json:"rid,omitempty"` // room id
	Pt   string            `json:"pt,omitempty"`  //"publisher" | "viewer"
	Eg   string            `json:"eg,omitempty"`  // "native" | "agora"
	Icfg []string          `json:"is,omitempty"`

	Req interface{} `json:"req,omitempty"`
	Msg string      `json:"msg,omitempty"`
}
type RoomParticipant interface{} // no parcing yet
