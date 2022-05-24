package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dmisol/mobzroom"
	"github.com/google/uuid"
)

const (
	url = "wss://gkeh49rfy1.execute-api.us-east-1.amazonaws.com/staging"
)

func cb(data *mobzroom.DataAck) {
	log.Println(data)

	if data.S == "JOIN" && data.C == 200 {
		if err := mr.MakePeerConn(data.B.Icfg.Is); err != nil {
			fmt.Println(err)
		}
		if err := mr.SendOffer(); err != nil {
			fmt.Println(err)
		}
	}
}

var mr *mobzroom.RoomClient

func main() {

	si := &mobzroom.SessionInfo{
		UserId:    "user" + uuid.New().String(),
		SessionId: "sess" + uuid.New().String(),
		Category:  "stream",
		Host:      "Web_Studio",
		DeviceId:  "dev" + uuid.New().String(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	mr = mobzroom.NewClient(ctx, url, "room"+uuid.NewString(), "m2m", si, cb, nil)
	mr.Join(&mobzroom.Op{RoomCreating: true, UserFaking: true}, true, true)
	defer func() {
		log.Println("CANCEL")
		cancel()
		time.Sleep(2 * time.Second)
	}()

	time.Sleep(30 * time.Second)
}
