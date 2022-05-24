package mobzroom

import (
	"github.com/pion/webrtc/v3"
)

const (
	PayloadTypeH264 = 126
	PayloadTypeOpus = 111

	IceStartPort = 20000
	IceEndPort   = 40000
)

// MakePeerConn() creates peerConection and fills it as PeerConn field;
// callback functions are to be configured afterwards;
// transceivers are to be added, direction set etc
func (mr *RoomClient) MakePeerConn(turns []string) (err error) {

	m := webrtc.MediaEngine{}
	if err = m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType: "video/H264", ClockRate: 90000, Channels: 0, SDPFmtpLine: "",
		},
		PayloadType: PayloadTypeH264,
	}, webrtc.RTPCodecTypeVideo); err != nil {
		return
	}

	if err = m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType: "audio/opus", ClockRate: 48000, Channels: 0, SDPFmtpLine: "",
		},
		PayloadType: PayloadTypeOpus,
	}, webrtc.RTPCodecTypeAudio); err != nil {
		return
	}

	se := webrtc.SettingEngine{}
	if err = se.SetEphemeralUDPPortRange(uint16(IceStartPort), uint16(IceEndPort)); err != nil {
		return
	}
	se.SetLite(false)

	api := webrtc.NewAPI(
		webrtc.WithMediaEngine(&m),
		webrtc.WithSettingEngine(se),
	)

	config := webrtc.Configuration{}
	for _, s := range turns {
		is := webrtc.ICEServer{URLs: []string{s}}
		config.ICEServers = append(config.ICEServers, is)
	}

	mr.PeerConn, err = api.NewPeerConnection(config)
	if err != nil {
		return
	}
	return
}

func (mr *RoomClient) SendOffer() (err error) {
	var sdp webrtc.SessionDescription
	sdp, err = mr.PeerConn.CreateOffer(nil)
	if err != nil {
		return
	}

	if err = mr.PeerConn.SetLocalDescription(sdp); err != nil {
		return
	}

	//mr.Webrtc("offer", sdp.SDP, mr.dc.Ss.UserId, &WrtcOp{})
	mr.Webrtc("offer", sdp.SDP, "smth deprecated", nil)
	return
}
