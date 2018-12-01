package ucp

import (
	"bytes"
	"fmt"
)

// Alert is an Alert Operation(31).
type Alert struct {
	pdu *PDU
	AdC []byte
	PID []byte
}

// NewAlert creates a new Alert PDU.
func NewAlert(pdu *PDU) *Alert {
	b := bytes.Split(pdu.Data, []byte("/"))
	return &Alert{
		pdu: pdu,
		AdC: b[0],
		PID: b[1],
	}
}

// Result returns an Alert Operation Result.
func (a *Alert) Result() []byte {
	b := make([]byte, 0)
	b = append(b, STX)
	Len := 19 + len("0000")
	partial := [][]byte{
		a.pdu.TransRefNum,
		[]byte(fmt.Sprintf("%05d", Len)),
		[]byte("R"), a.pdu.Operation,
		[]byte("A"),
		[]byte("0000"),
	}
	p := append(bytes.Join(partial, []byte("/")), []byte("/")...)
	chksum := checkSum(p)
	result := append(p, chksum...)
	b = append(b, result...)
	b = append(b, ETX)
	return b
}
