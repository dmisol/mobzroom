package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"sync/atomic"
	"time"

	"github.com/dmisol/mobzroom"
	"github.com/google/uuid"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

const (
	url = "wss://gkeh49rfy1.execute-api.us-east-1.amazonaws.com/staging"
)

func cb(data *mobzroom.DataAck) {
	//defer fmt.Println()

	if data.C != 200 {
		mr.Println("wrong ack", data)
	}
	switch data.S {
	case "JOIN":
		mr.Println("JOIN ACK")
		if err := mr.MakePeerConn(data.B.Icfg.Is); err != nil {
			log.Println(err)
		}
		/*
			mr.PeerConn.OnICECandidate(func(i *webrtc.ICECandidate) {
				if i == nil {
					log.Println("empty ICE")
					return
				}
				fmt.Println("ice", i)
				//if err := mr.SendICE("a=" + i.ToJSON().Candidate); err != nil {
				if err := mr.SendICE(i.ToJSON().Candidate); err != nil {
					log.Println(err)
				}
			})
		*/
		populatePeerConn()
		mr.SendOffer()

		return
	case "OFFER":
		//mr.Println("OFFER ACK", data.B.Sdp)
		sdp := webrtc.SessionDescription{
			Type: webrtc.SDPTypeAnswer,
			SDP:  data.B.Sdp,
		}

		if err := mr.PeerConn.SetRemoteDescription(sdp); err != nil {
			mr.Println("set remote desc", err)
			return
		}
		return
	case "PUBLISH_LIST_CHANGED":
		mr.Println(data.S)
		return
	}

	log.Println(data)
}

func populatePeerConn() (err error) {
	mr.PeerConn.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("Connection State has changed %s \n", connectionState.String())
	})

	videoTrack, err = webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: "video/H264"}, "video", "pionVideo")
	if err != nil {
		log.Println(err)
		return
	}

	// Create an audio track
	audioTrack, err = webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: "audio/opus", ClockRate: 48000, Channels: 2, SDPFmtpLine: "minptime=10;useinbandfec=1", RTCPFeedback: nil}, "audio", "pionAudio")
	if err != nil {
		log.Println(err)
		return
	}

	if _, err = mr.PeerConn.AddTransceiverFromTrack(videoTrack,
		webrtc.RTPTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionSendonly,
		},
	); err != nil {
		log.Println(err)
		return
	}

	if _, err = mr.PeerConn.AddTransceiverFromTrack(audioTrack,
		webrtc.RTPTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionSendonly,
		},
	); err != nil {
		log.Println(err)
		return
	}

	return
}

var (
	mr         *mobzroom.RoomClient
	videoTrack *webrtc.TrackLocalStaticRTP
	audioTrack *webrtc.TrackLocalStaticRTP
	apt, vpt   *int
	ctx        context.Context
	cancel     context.CancelFunc

	running int64
)

func runRtp(udp *net.UDPConn) {
	for {
		select {
		case <-ctx.Done():
			udp.Close()
			return
		default:
			p := make([]byte, 4096)
			n, _, err := udp.ReadFrom(p)
			if err != nil {
				log.Println("udp rd", err)
				cancel()
				return
			}
			packet := &rtp.Packet{}
			if err = packet.Unmarshal(p[:n]); err != nil {
				log.Println("rtp", err)
			}
			if atomic.LoadInt64(&running) == 0 {
				log.Println("webrtc is not ready")
				continue
			}
			switch packet.PayloadType {
			case uint8(*apt):
				tx := p
				if *apt != mobzroom.PayloadTypeOpus {
					log.Println("replacing audio pt")
					packet.PayloadType = mobzroom.PayloadTypeOpus
					if tx, err = packet.Marshal(); err != nil {
						log.Println("audio tx marshal", err)
					}
					n = len(tx)
				}
				if _, writeErr := audioTrack.Write(tx[:n]); writeErr != nil {
					log.Println("audio tx", err)
					cancel()
					return
				}
			case uint8(*vpt):
				tx := p
				if *vpt != mobzroom.PayloadTypeH264 {
					log.Println("replacing video pt")
					packet.PayloadType = mobzroom.PayloadTypeH264
					if tx, err = packet.Marshal(); err != nil {
						log.Println("video tx marshal", err)
					}
					n = len(tx)
				}
				if _, writeErr := videoTrack.Write(tx[:n]); writeErr != nil {
					log.Println("video tx", err)
					cancel()
					return
				}
			default:
				log.Println("unexpected rtp pt", packet.PayloadType)

			}
		}
	}
}

func main() {
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Hour)

	apt = flag.Int("apt", mobzroom.PayloadTypeOpus, "audio payload type")
	vpt = flag.Int("vpt", mobzroom.PayloadTypeH264, "video payload type")
	port := flag.Int("rtp", 10000, "rtp port (a+v)")
	flag.Parse()

	udp, err := net.ListenUDP("udp", &net.UDPAddr{Port: *port})
	if err != nil {
		log.Println(err)
		return
	}
	go runRtp(udp)

	si := &mobzroom.SessionInfo{
		UserId:    "user" + uuid.New().String(),
		SessionId: "sess" + uuid.New().String(),
		Category:  "stream",
		Host:      "Web_Studio",
		DeviceId:  "dev" + uuid.New().String(),
	}

	mr = mobzroom.NewClient(ctx, url, "room"+uuid.NewString(), "m2m", si, cb, nil)
	mr.Join(&mobzroom.Op{RoomCreating: true, UserFaking: true}, true, true)
	defer func() {
		log.Println("CANCEL")
		cancel()
		time.Sleep(2 * time.Second)
	}()

	time.Sleep(30 * time.Second)
}
