package main

import (
	"log"

	"github.com/gvalkov/golang-evdev"
)

var keyMap = map[string]string{
	"KEY_W": "",
	"KEY_A": "bg1.raw",
	"KEY_S": "bg2.raw",
	"KEY_D": "bg3.raw",
	"KEY_F": "bg4.raw",
	"KEY_G": "bg5.raw",
}

func ctlKeys(kbd string, control chan CTL) {
	dev, err := evdev.Open(kbd)
	if err != nil {
		log.Printf("unable to open kbd: %v", err)
		return
	}
	for {
		ev, err := dev.ReadOne()
		if err != nil {
			log.Printf("failed to ReadOne: %v", err)
			return
		}
		if ev.Type != evdev.EV_KEY {
			continue
		}
		if ev.Value != int32(evdev.KeyDown) {
			continue
		}
		kname := evdev.KEY[int(ev.Code)]
		name, found := keyMap[kname]
		if !found {
			continue
		}
		control <- CTL(name)
	}
}
