package ucp

import (
	"bytes"
	"encoding/hex"
	"fmt"
)

// Session is a Session Management Operation(60).
type Session struct {
	pdu  *PDU
	OAdC []byte
	OTON []byte
	ONPI []byte
	STYP []byte
	PWD  []byte
	NPWD []byte
	VERS []byte
	LAdC []byte
	LTON []byte
	LNPI []byte
	OPID []byte
	RES1 []byte
}

// NewSession creates a new Session Management Operation PDU.
func NewSession(pdu *PDU) *Session {
	b := bytes.Split(pdu.Data, []byte("/"))
	return &Session{
		pdu:  pdu,
		OAdC: b[0],
		OTON: b[1],
		ONPI: b[2],
		STYP: b[3],
		PWD:  b[4],
		NPWD: b[5],
		VERS: b[6],
		LAdC: b[7],
		LTON: b[8],
		LNPI: b[9],
		OPID: b[10],
		RES1: b[11],
	}
}

func (s *Session) GetPassword() string {
	pw := make([]byte, hex.DecodedLen(len(s.PWD)))
	hex.Decode(pw, s.PWD)
	return string(pw[:])
}

func (s *Session) GetStyp() string {
	switch string(s.STYP[:]) {
	case "1":
		return "open session"
	case "2", "5":
		return "reserved"
	case "3":
		return "change password"
	case "4":
		return "open provisioning session"
	case "6":
		return "change provisioning password"
	default:
		return ""
	}
}

func (s *Session) GetOAdc() string {
	return string(s.OAdC[:])
}

func (s *Session) GetOTON() string {
	switch string(s.OTON[:]) {
	case "1":
		return "International number"
	case "2":
		return "National number"
	case "6":
		return "Abbreviated number"
	default:
		return ""
	}
}

func (s *Session) GetONPI() string {
	switch string(s.ONPI[:]) {
	case "1":
		return "E.164 address"
	case "3":
		return "X121 address"
	case "5":
		return "SMSC specific: Private (TCP/IP address/abbreviated number)"
	default:
		return ""
	}
}

// Result returns a Session Management Operation Result.
func (s *Session) Result() []byte {
	b := make([]byte, 0)
	b = append(b, STX)
	systemMsg := "BIND AUTHENTICATED"
	Len := 19 + len(systemMsg)
	partial := [][]byte{
		s.pdu.TransRefNum,
		[]byte(fmt.Sprintf("%05d", Len)),
		[]byte("R"), s.pdu.Operation,
		[]byte("A"),
		[]byte(systemMsg),
	}
	p := append(bytes.Join(partial, []byte("/")), []byte("/")...)
	chksum := checkSum(p)
	result := append(p, chksum...)
	b = append(b, result...)
	b = append(b, ETX)
	return b
}

// Error returns a Negative Acknowledgement Result.
func (s *Session) Error() []byte {
	b := make([]byte, 0)
	b = append(b, STX)
	systemMsg := "AUTHENTICATION FAILURE"
	Len := 22 + len(systemMsg)
	partial := [][]byte{
		s.pdu.TransRefNum,
		[]byte(fmt.Sprintf("%05d", Len)),
		[]byte("R"), s.pdu.Operation,
		[]byte("N"),
		[]byte("07"),
		[]byte(systemMsg),
	}
	p := append(bytes.Join(partial, []byte("/")), []byte("/")...)
	chksum := checkSum(p)
	result := append(p, chksum...)
	b = append(b, result...)
	b = append(b, ETX)
	return b
}
