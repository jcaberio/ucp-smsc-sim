package ucp

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"time"
)

var DeliverSMCh = make(chan DeliverSM, 10)

type DeliverSM struct {
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

func (d *DeliverSM) Result() []byte {
	msg := d.Msg
	msgIra := make([]byte, hex.EncodedLen(len(msg)))
	hex.Encode(msgIra, msg)
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
		[]byte(time.Now().Format("020106150405")),
		d.Dst,
		d.Rsn,
		d.DSCTS,
		d.MT,
		d.NB,
		msgIra,
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
		[]byte("01"),
		[]byte(fmt.Sprintf("%05d", Len)),
		[]byte("O"),
		[]byte("52"),
		bdata,
	}
	p := append(bytes.Join(partial, []byte("/")), []byte("/")...)
	chksum := checkSum(p)
	result := append(p, chksum...)
	b = append(b, result...)
	b = append(b, ETX)
	return b
}
