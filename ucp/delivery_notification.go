package ucp

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"time"
)

// DeliverNotification is a Deliver Notification Operation(53).
type DeliverNotification struct {
	pdu   *PDU
	AdC   []byte
	OAdC  []byte
	AC    []byte
	NRq   []byte
	NAdC  []byte
	NT    []byte
	NPID  []byte
	LRq   []byte
	LRAd  []byte
	LPID  []byte
	DD    []byte
	DDT   []byte
	VP    []byte
	RPID  []byte
	SCTS  []byte
	Dst   []byte
	Rsn   []byte
	DSCTS []byte
	MT    []byte
	NB    []byte
	Msg   []byte
	MMS   []byte
	PR    []byte
	DCs   []byte
	MCLs  []byte
	RPI   []byte
	CPg   []byte
	RPLy  []byte
	OTOA  []byte
	HPLMN []byte
	Xser  []byte
	RES4  []byte
	RES5  []byte
}

// NewDeliverNotification creates a new Deliver Notification PDU.
func NewDeliverNotification(pdu *PDU, AdC, OAdC, SCTS string) *DeliverNotification {
	tscts, _ := time.Parse("020106150405", SCTS)
	after2s := tscts.Add(2 * time.Second)
	msg := []byte("Message for " + OAdC + " with identification " + OAdC + ":" + SCTS + " has been delivered at " + after2s.String())
	msgIra := make([]byte, hex.EncodedLen(len(msg)))
	hex.Encode(msgIra, msg)
	return &DeliverNotification{
		pdu:   pdu,
		AdC:   []byte(AdC),
		OAdC:  []byte(OAdC),
		SCTS:  []byte(SCTS),
		Dst:   []byte("0"),
		Rsn:   []byte("000"),
		DSCTS: []byte(after2s.Format("020106150405")),
		MT:    []byte("3"),
		Msg:   msgIra,
	}
}

// Result returns a Deliver Notification Result.
func (d *DeliverNotification) Result() []byte {
	b := make([]byte, 0)
	b = append(b, STX)
	data := [][]byte{
		d.AdC,
		d.OAdC,
		d.AC,
		d.NRq,
		d.NAdC,
		d.NT,
		d.NPID,
		d.LRq,
		d.LRAd,
		d.LPID,
		d.DD,
		d.DDT,
		d.VP,
		d.RPID,
		d.SCTS,
		d.Dst,
		d.Rsn,
		d.DSCTS,
		d.MT,
		d.NB,
		d.Msg,
		d.MMS,
		d.PR,
		d.DCs,
		d.MCLs,
		d.RPI,
		d.CPg,
		d.RPLy,
		d.OTOA,
		d.HPLMN,
		d.Xser,
		d.RES4,
		d.RES5,
	}
	bdata := bytes.Join(data, []byte("/"))
	Len := 17 + len(bdata)
	partial := [][]byte{
		[]byte("99"),
		[]byte(fmt.Sprintf("%05d", Len)),
		[]byte("O"),
		[]byte("53"),
		bdata,
	}
	p := append(bytes.Join(partial, []byte("/")), []byte("/")...)
	chksum := checkSum(p)
	result := append(p, chksum...)
	b = append(b, result...)
	b = append(b, ETX)
	return b
}
