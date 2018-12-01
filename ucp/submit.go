package ucp

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/go-gsm/charset"
)

// ExtraService field allows the specification of one or more additional services,
// all in the format TTLLDD,
// where TT field specifies the type of service,
// LL indicates the length of data and
// DD indicates zero or more data elements
type ExtraService string

const (
	// User Data Header
	UDH ExtraService = "01"
	//BillingIdentifier enables the client to send additional billing information to the server
	BillingIdentifier ExtraService = "0C"
)

// Submit is a Submit Short Message Operation(51).
type Submit struct {
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

// NewSubmit creates a new Submit Short Message Operation PDU.
func NewSubmit(pdu *PDU) *Submit {
	b := bytes.Split(pdu.Data, []byte("/"))
	return &Submit{
		pdu:   pdu,
		AdC:   b[0],
		OAdC:  b[1],
		AC:    b[2],
		NRq:   b[3],
		NAdC:  b[4],
		NT:    b[5],
		NPID:  b[6],
		LRq:   b[7],
		LRAd:  b[8],
		LPID:  b[9],
		DD:    b[10],
		DDT:   b[11],
		VP:    b[12],
		RPID:  b[13],
		SCTS:  []byte(time.Now().Format("020106150405")),
		Dst:   b[15],
		Rsn:   b[16],
		DSCTS: b[17],
		MT:    b[18],
		NB:    b[19],
		Msg:   b[20],
		MMS:   b[21],
		PR:    b[22],
		DCs:   b[23],
		MCLs:  b[24],
		RPI:   b[25],
		CPg:   b[26],
		RPLy:  b[27],
		OTOA:  b[28],
		HPLMN: b[29],
		Xser:  b[30],
		RES4:  b[31],
		RES5:  b[32],
	}
}

// GetMessage returns the decoded message
func (submit *Submit) GetMessage() string {
	const (
		AMsg = "3"
		TD   = "4"
	)
	var msg string

	if string(submit.MT[:]) == AMsg {
		msg = decodeIRA(submit.Msg)
	} else if string(submit.MT[:]) == TD {
		decoded, err := hex.DecodeString(string(submit.Msg))
		if err != nil {
			log.Println(err)
		}
		utf16, err := charset.DecodeUcs2(decoded)
		if err != nil {
			log.Println(err)
		}
		msg = utf16
	}

	return msg
}

// GetRecipient returns the recipient of the messsage
func (submit *Submit) GetRecipient() string {
	return string(submit.AdC[:])
}

// IsNotifRequested returns true if a delivery notification is requested
func (submit *Submit) IsNotifRequested() bool {
	return string(submit.NT) == "1"
}

// GetSCTS returns the Service Center Timestamp
func (submit *Submit) GetSCTS() string {
	return string(submit.SCTS[:])
}

// Result returns a Submit Short Message Result
func (a *Submit) Result() []byte {
	b := make([]byte, 0)
	b = append(b, STX)
	message := string(a.AdC[:]) + ":" + time.Now().Format("020106150405")
	Len := 20 + len(message)
	partial := [][]byte{
		a.pdu.TransRefNum,
		[]byte(fmt.Sprintf("%05d", Len)),
		[]byte("R"), a.pdu.Operation,
		[]byte("A"),
		[]byte(""),
		[]byte(message),
	}
	p := append(bytes.Join(partial, []byte("/")), []byte("/")...)
	chksum := checkSum(p)
	result := append(p, chksum...)
	b = append(b, result...)
	b = append(b, ETX)
	return b
}

// ParseXser returns a map of Extra Services
func (s *Submit) ParseXser() map[ExtraService]string {
	m := make(map[ExtraService]string)
	if len(s.Xser) == 0 {
		return m
	}
	buf := bytes.NewBuffer(s.Xser)
	for buf.Len() > 0 {
		xserType := buf.Next(2)
		xserLen := buf.Next(2)
		convXserlen, _ := strconv.ParseInt(string(xserLen), 16, 0)
		xserData := buf.Next(int(convXserlen) * 2)
		m[ExtraService(xserType)] = string(xserData)
	}
	return m
}
