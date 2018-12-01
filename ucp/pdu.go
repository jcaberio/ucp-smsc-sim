// Package ucp provides UCP primitives.
//
// References:
//	http://documents.swisscom.com/product/1000174-Internet/Documents/Landingpage-Mobile-Mehrwertdienste/UCP_R4.7.pdf
// 	https://wiki.wireshark.org/UCP
//	https://www.wireshark.org/docs/dfref/u/ucp.html
package ucp

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jcaberio/ucp-smsc-sim/util"
	"github.com/pkg/errors"
	"gopkg.in/redis.v5"
)

const (
	STX                      = 2
	ETX                      = 3
	ALERT_OP                 = "31"
	SUBMIT_SHORT_MESSAGE_OP  = "51"
	DELIVER_SHORT_MESSAGE_OP = "52"
	DELIVER_NOTIFICATION_OP  = "53"
	SESSION_MANAGEMENT_OP    = "60"
)

// PDU is a UCP protocol data unit.
type PDU struct {
	sync.Mutex
	conn net.Conn
	// Transaction Reference Number
	TransRefNum []byte
	// PDU length
	Len []byte
	// Operation type or Result type
	Type []byte
	// Operation Identifier
	Operation []byte
	// Operation data
	Data []byte
	// PDU checksum
	Checksum []byte
}

// New creates a new PDU object.
func New(r net.Conn) (pdu *PDU, err error) {
	reader := bufio.NewReader(r)
	raw, _ := reader.ReadSlice(ETX)
	if len(raw) == 0 {
		return pdu, errors.New("Empty packet")
	}
	if raw[0] != STX {
		return pdu, errors.New("Invalid STX")
	}
	if raw[len(raw)-1] != ETX {
		return pdu, errors.New("Invalid ETX")
	}
	TransRefNum := raw[1:3]
	Len := raw[4:9]
	OperationOrResult := []byte{raw[10]}
	OpType := raw[12:14]
	Data := raw[15 : len(raw)-4]
	Checksum := raw[len(raw)-3 : len(raw)-1]
	pdu = &PDU{
		conn:        r,
		TransRefNum: TransRefNum,
		Len:         Len,
		Type:        OperationOrResult,
		Operation:   OpType,
		Data:        Data,
		Checksum:    Checksum,
	}
	return pdu, err
}

// Decode sends a result PDU to the client.
func (pdu *PDU) Decode(conf util.Config) {
	if pdu == nil {
		return
	}

	switch string(pdu.Operation) {
	case ALERT_OP:
		alert := NewAlert(pdu)
		res := alert.Result()
		client.Set(ResPacket, string(res), 30*time.Second)
		pdu.Lock()
		time.Sleep(time.Duration(GetKeepAliveTimeout()) * time.Second)
		_, err := pdu.conn.Write(res)
		SetKeepAliveTimeout(0)
		pdu.Unlock()
		if err != nil {
			log.Println("Writing ALERT failed: ", err)
		}
	case SUBMIT_SHORT_MESSAGE_OP:
		sub := NewSubmit(pdu)
		recipient := sub.GetRecipient()
		scts := sub.GetSCTS()

		if val, ok := sub.ParseXser()[BillingIdentifier]; ok {
			tariff, _ := hex.DecodeString(val)
			cost := conf.Tariff[string(tariff)]
			client.IncrByFloat(Cost, cost)
		}
		if sub.IsNotifRequested() {
			go func(pdu *PDU, recipient, scts string, client *redis.Client) {
				time.Sleep(time.Duration(conf.DNDelay) * time.Millisecond)
				dlvr := NewDeliverNotification(pdu, conf.AccessCode, recipient, scts)
				res := dlvr.Result()
				client.Set(ResPacket, string(res), 30*time.Second)
				pdu.Lock()
				_, err := pdu.conn.Write(dlvr.Result())
				pdu.Unlock()
				if err != nil {
					log.Println("Writing DR failed: ", err)
				}
				client.HIncrBy(CountersKey, DrField, 1)
			}(pdu, recipient, scts, client)
		}
		res := sub.Result()
		client.Set(ResPacket, string(res), 30*time.Second)
		pdu.Lock()
		_, err := pdu.conn.Write(res)
		pdu.Unlock()
		if err != nil {
			log.Println("Writing SM failed: ", err)
		}
	case DELIVER_NOTIFICATION_OP:
	case DELIVER_SHORT_MESSAGE_OP:
	case SESSION_MANAGEMENT_OP:
		sesMngt := NewSession(pdu)
		var res []byte
		if sesMngt.GetOAdc() == conf.User && sesMngt.GetPassword() == conf.Password {
			res = sesMngt.Result()
		} else {
			res = sesMngt.Error()
		}
		client.Set(ResPacket, string(res), 30*time.Second)
		pdu.conn.Write(res)

	default:
		log.Println("UNKNOWN OPERATION")
	}
}

// String returns the string representation of a PDU.
func (pdu *PDU) String() string {
	b := bytes.Join([][]byte{pdu.TransRefNum, pdu.Len, pdu.Type, pdu.Operation, pdu.Data, pdu.Checksum}, []byte("/"))
	return string(b[:])
}

// checkSum computes the checksum of the pdu
func checkSum(b []byte) []byte {
	var sum byte
	for _, i := range b {
		sum += i
	}
	mask := sum & 0xFF
	ck := strings.ToUpper(strconv.FormatInt(int64(mask), 16))
	return []byte(fmt.Sprintf("%02s", ck))
}
