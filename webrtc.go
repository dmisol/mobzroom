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
			MimeType: "audio/opus", ClockRate: 48000, Channels: 2, SDPFmtpLine: "",
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

func (mr *RoomClient) SendOffer() {
	go func() {
		gc := webrtc.GatheringCompletePromise(mr.PeerConn)

		off, err := mr.PeerConn.CreateOffer(nil)
		if err != nil {
			mr.Println("create offer", err)
			return
		}

		if err = mr.PeerConn.SetLocalDescription(off); err != nil {
			mr.Println("set local desc", err)
			return
		}

		<-gc
		sdp := mr.PeerConn.LocalDescription()
		//mr.Println("offer", sdp.SDP)
		mr.Webrtc("offer", sdp.SDP, "smth deprecated", &WrtcOp{})
	}()

	return
}

func (mr *RoomClient) SendICE(s string) (err error) {
	//mr.Webrtc("iceCandidate", s, &IceTo{UserId: mr.dc.Ss.UserId, SessionId: mr.dc.Ss.SessionId}, nil)
	mr.Webrtc("iceCandidate", s, mr.dc.Ss.UserId, nil)
	//mr.Webrtc("iceCandidate", s, mr.dc.Ss.SessionId, nil)
	return
}
