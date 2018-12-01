package ucp

import (
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"

	"github.com/go-gsm/charset"
	"github.com/jcaberio/ucp-smsc-sim/util"
	"github.com/paulbellamy/ratecounter"
	"github.com/satori/go.uuid"
	"gopkg.in/redis.v5"
)

var (
	// Suffix is a unique identifier for the current running server
	Suffix = uuid.NewV1().String()
	// CountersKey is a redis key for atomic counters
	CountersKey = "counters_" + Suffix
	// SmField is a redis field for submit_sm
	SmField = "submit_sm_" + Suffix
	// DrField is a redis field for deliver_sm
	DrField = "deliver_sm_" + Suffix
	// TpsKey is a redis key for tps
	TpsKey = "tps_" + Suffix
	// ActiveConnection is a redis key for sorted set of active connections
	ActiveConnection = "active_conn_" + Suffix
	// ReqPacket is a redis key for incoming tcp packet
	ReqPacket = "req_packet_" + Suffix
	// ResPacket is a redis key for outgoing tcp packet
	ResPacket = "res_packet_" + Suffix
	// MsgList is a redis key for message list
	MsgList = "msg_list_" + Suffix
	// Cost is a redis key for the total message cost
	Cost = "cost_" + Suffix
	// RefNum is a redis key for a long message with a reference number
	RefNum string
	// IpSrcDstMsg is a redis key for storing messages sent by an IP addr
	IpSrcDstMsg string = "ip_src_dst_msg_" + Suffix
	client             = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	tpsCounter       = ratecounter.NewRateCounter(1 * time.Second)
	mu               sync.Mutex
	KeepAliveTimeout = 0
)

func SetKeepAliveTimeout(n int) {
	mu.Lock()
	KeepAliveTimeout = n
	mu.Unlock()
}

func GetKeepAliveTimeout() int {
	mu.Lock()
	defer mu.Unlock()
	return KeepAliveTimeout

}

// Stats updates the Stats display in the web UI.
func (pdu *PDU) Stats() {
	client.Set(ReqPacket, pdu.String(), 30*time.Second)
	switch string(pdu.Operation) {
	case SUBMIT_SHORT_MESSAGE_OP:
		tpsCounter.Incr(1)
		client.Set(TpsKey, tpsCounter.Rate(), 1*time.Second)
		client.HIncrBy(CountersKey, SmField, 1)
		submitPdu := NewSubmit(pdu)
		shortMessage := submitPdu.GetMessage()
		source := string(submitPdu.OAdC)
		destination := string(submitPdu.AdC)
		decodedSrc, _ := hex.DecodeString(source)
		unpacked := charset.Unpack7Bit(decodedSrc[1:])
		src, _ := charset.Decode7Bit(unpacked)
		if val, ok := submitPdu.ParseXser()[UDH]; ok {
			RefNum = val[len(val)-6 : len(val)-4]
			msgPartsLen := val[len(val)-4 : len(val)-2]
			msgPart := val[len(val)-2:]
			lastMsg := client.HGet(RefNum, "message").Val()
			client.HMSet(RefNum, map[string]string{
				"total_parts":      msgPartsLen,
				"current_part_num": msgPart,
				"message":          lastMsg + shortMessage,
			})
			total_parts := client.HGet(RefNum, "total_parts").Val()
			current_part_num := client.HGet(RefNum, "current_part_num").Val()
			if total_parts == current_part_num {
				shortMessage = client.HGet(RefNum, "message").Val()
				wsMsg := util.Message{Message: shortMessage, Sender: src, Recipient: destination, Timestamp: time.Now().String()}
				save(wsMsg)
				client.Del(RefNum)
			}
		} else {
			wsMsg := util.Message{Message: shortMessage, Sender: src, Recipient: destination, Timestamp: time.Now().String()}
			save(wsMsg)
		}
		client.HMSet(IpSrcDstMsg, map[string]string{
			pdu.conn.RemoteAddr().String(): src + "_" + destination + "_" + shortMessage,
		})
	case ALERT_OP:
		nextMinute := time.Now().Add(1 * time.Minute).Unix()
		client.ZAdd(ActiveConnection,
			redis.Z{
				Score:  float64(nextMinute),
				Member: pdu.conn.RemoteAddr().String(),
			},
		)
	}
}

func save(wsMsg util.Message) {
	msgJSON, _ := json.Marshal(&wsMsg)
	client.RPush(MsgList, msgJSON)
	client.LTrim(MsgList, -10, -1)
}
