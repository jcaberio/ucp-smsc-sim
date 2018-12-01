// Package ui provides a web interface for displaying statistics.
package ui

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-gsm/charset"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/jcaberio/ucp-smsc-sim/ucp"
	"github.com/jcaberio/ucp-smsc-sim/util"
	"github.com/shirou/gopsutil/process"
	"gopkg.in/redis.v5"
)

const (
	writeWait = 60 * time.Second
	// Time allowed to read the next pong message from the client.
	pongWait = 60 * time.Second
	// Send pings to client with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

var (
	httpAddress string
	homeTempl   = template.Must(template.New("").Parse(htmlTemplate))
	upgrader    = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	client = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	sigChan = make(chan os.Signal, 3)
)

func reader(ws *websocket.Conn) {
	defer ws.Close()
	ws.SetReadLimit(1024)
	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error { ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			break
		}
	}
}

func writer(ws *websocket.Conn) {
	ticker := time.NewTicker(time.Millisecond * 200)
	pingTicker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		pingTicker.Stop()
		ws.Close()
	}()
	for {
		select {
		case <-ticker.C:
			proc, _ := process.NewProcess(int32(os.Getpid()))
			memoryPercent, _ := proc.MemoryPercent()
			cpuPercent, _ := proc.Percent(1 * time.Second)
			drCount, _ := client.HGet(ucp.CountersKey, ucp.DrField).Int64()
			smCount, _ := client.HGet(ucp.CountersKey, ucp.SmField).Int64()
			tps, _ := client.Get(ucp.TpsKey).Int64()
			cost, _ := client.Get(ucp.Cost).Float64()
			client.ZRemRangeByScore(ucp.ActiveConnection, "0", string(time.Now().Unix()))
			nowStr := strconv.FormatInt(time.Now().Unix(), 10)
			reqPacket := client.Get(ucp.ReqPacket).Val()
			resPacket := client.Get(ucp.ResPacket).Val()
			activeConns := client.ZRangeByScore(ucp.ActiveConnection,
				redis.ZRangeBy{Min: nowStr, Max: "+inf"}).Val()
			msgListStr := client.LRange(ucp.MsgList, 0, -1).Val()
			msgList := make([]util.Message, 0)
			var msgObj util.Message
			for _, msgStr := range msgListStr {
				json.Unmarshal([]byte(msgStr), &msgObj)
				msgList = append(msgList, msgObj)
			}
			ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ws.WriteJSON(struct {
				IncomingPackets string
				OutgoingPackets string
				MemoryPercent   float32
				CpuPercent      float64
				Cost            float64
			}{
				IncomingPackets: reqPacket,
				OutgoingPackets: resPacket,
				MemoryPercent:   memoryPercent,
				CpuPercent:      cpuPercent,
				Cost:            cost,
			}); err != nil {
				log.Println("Error in writing packets to web socket")
				return
			}
			if err := ws.WriteJSON(struct {
				Messages []util.Message
			}{Messages: msgList}); err != nil {
				log.Println("Error in writing latest messages to web socket")
				return
			}
			if err := ws.WriteJSON(struct {
				ClientConns []string
			}{ClientConns: activeConns}); err != nil {
				log.Println("Error in writing active client connections to web socket")
				return
			}
			if err := ws.WriteJSON(struct {
				DRCount int64
				SMCount int64
				Tps     int64
			}{
				DRCount: drCount,
				SMCount: smCount,
				Tps:     tps,
			}); err != nil {
				log.Println("Error in writing atomic counters to web socket")
				return
			}

		case <-pingTicker.C:
			ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Println(err)
		}
		return
	}
	go writer(ws)
	reader(ws)
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", 404)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	homeTempl.Execute(w, struct{ WsPort string }{WsPort: httpAddress})
}

func messagesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	msgListStr := client.LRange(ucp.MsgList, 0, -1).Val()
	msgList := make([]util.Message, 0)
	var msgObj util.Message
	for _, msgStr := range msgListStr {
		json.Unmarshal([]byte(msgStr), &msgObj)
		msgList = append([]util.Message{msgObj}, msgList...)
	}
	json.NewEncoder(w).Encode(msgList)
}

func drHandler(w http.ResponseWriter, r *http.Request) {
	drCount, _ := client.HGet(ucp.CountersKey, ucp.DrField).Int64()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(struct {
		DrCount int64 `json:"deliver_sm_resp_count"`
	}{
		drCount,
	})
}

func smHandler(w http.ResponseWriter, r *http.Request) {
	smCount, _ := client.HGet(ucp.CountersKey, ucp.SmField).Int64()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(struct {
		SmCount int64 `json:"submit_sm_count"`
	}{
		smCount,
	})
}

func tpsHandler(w http.ResponseWriter, r *http.Request) {
	tps, _ := client.Get(ucp.TpsKey).Int64()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(struct {
		Tps int64 `json:"tps"`
	}{
		Tps: tps,
	})
}

func resetHandler(w http.ResponseWriter, r *http.Request) {
	client.HMSet(ucp.CountersKey, map[string]string{ucp.SmField: "0"})
}

func timeoutHandler(w http.ResponseWriter, r *http.Request) {
	ucp.SetKeepAliveTimeout(60)
}

func deliverSmHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	type Mo struct {
		Sender   string `json:"sender"`
		Receiver string `json:"receiver"`
		Message  string `json:"message"`
	}

	decoder := json.NewDecoder(r.Body)
	var mo Mo
	err := decoder.Decode(&mo)
	if err != nil {
		log.Println(err)
	}
	defer r.Body.Close()
	deliverSm1 := ucp.DeliverSM{
		AdC:  []byte(mo.Receiver),
		OAdC: []byte(mo.Sender),
		Msg:  charset.EncodeUcs2(mo.Message),
		Xser: []byte("020108"),
	}

	select {
	case ucp.DeliverSMCh <- deliverSm1:
		log.Println("sent deliver_sm")
	default:
		log.Println("no deliver_sm sent")
	}
}

// Render displays the web UI.
func Render(conf util.Config) {
	httpAddress = conf.HttpAddr
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGUSR2)
	go func() {
		for {
			sigc := <-sigChan
			switch sigc {
			case os.Interrupt, syscall.SIGTERM:
				log.Println("Deleting redis keys")
				client.Del(ucp.CountersKey, ucp.ActiveConnection, ucp.ReqPacket,
					ucp.ResPacket, ucp.MsgList, ucp.TpsKey, ucp.RefNum, ucp.Cost, ucp.IpSrcDstMsg)
				log.Println("Exit")
				os.Exit(0)
			case syscall.SIGUSR1:
				log.Println("sleeping for 30 seconds on keepalive")
				ucp.SetKeepAliveTimeout(60)
			case syscall.SIGUSR2:
				log.Println("Reset keepalive sleep to 0")
				ucp.SetKeepAliveTimeout(0)
			}
		}

	}()
	r := mux.NewRouter()
	attachProfiler(r)
	r.HandleFunc("/", serveHome)
	r.HandleFunc("/ws", serveWs)
	r.HandleFunc("/deliver_sm_resp_count", drHandler)
	r.HandleFunc("/submit_sm_count", smHandler)
	r.HandleFunc("/messages", messagesHandler)
	r.HandleFunc("/tps", tpsHandler)
	r.HandleFunc("/mo", deliverSmHandler)
	r.HandleFunc("/resetHandler", resetHandler)
	r.HandleFunc("/timeoutHandler", timeoutHandler)
	srv := &http.Server{
		Addr:    httpAddress,
		Handler: r,
	}
	go http.ListenAndServe(":80", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := "https://" + r.Host + r.URL.Path
		http.Redirect(w, r, target, http.StatusTemporaryRedirect)
	}))
	log.Fatal(srv.ListenAndServe())
}

func attachProfiler(router *mux.Router) {
	router.HandleFunc("/debug/pprof/", pprof.Index)
	router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	router.HandleFunc("/debug/pprof/profile", pprof.Profile)
	router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
    <head>
        <title>UCP Server</title>
		<link rel="shortcut icon" href="https://www.voyagerinnovation.com/hubfs/Voyager-July2016/favicon.ico?t=1500366615456">
        <meta charset="utf-8">
        <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css">
        <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap-theme.min.css">
        <script src="https://ajax.googleapis.com/ajax/libs/jquery/3.2.1/jquery.min.js"></script>
        <script src="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/js/bootstrap.min.js"></script>
    </head>
    <body><div class="container">
	<h4>Stats</h4>
	<ul class="list-group">
        <li class="list-group-item">
            <span id="sm_count" class="badge"></span>
            Submitted Messages
        </li>
        <li class="list-group-item">
            <span id="dr_count" class="badge"></span>
            Delivered Messages
		</li>
		<li class="list-group-item">
            <span id="tps" class="badge"></span>
            Request-per-second
        </li>
        <li class="list-group-item">
            <span id="cost" class="badge"></span>
            Cost
        </li>
	</ul>
    <button class="btn btn-info" data-toggle="collapse" data-target="#advanced_options">Advanced Options</button>
    <div id="advanced_options"  class="collapse">

		  <div  class="form-group">
      <br>
	    <form method="post" action="/resetHandler" id="resetHandler">
                <button type="submit" class="btn btn-warning">Reset</button>
      </form>
      </div>
 
      <div  class="form-group">
      <br>
	    <form method="post" action="/timeoutHandler" id="timeoutHandler">
                <button type="submit" class="btn btn-warning">TImeout</button>
      </form>
      </div>
    </div>

    <h4>Active Clients</h4>
	<div>
		<ul id="client-conns" class="list-group">
    	</ul>
	</div>
	<h4>Latest Messages</h4>	
	<table id="arrmsg" class="table table-hover">
		<thead>
		<tr>
		<th>source</th>
		<th>destination</th>
		<th>message</th>
    <th>timestamp</th>
		</tr>
		</thead>

		<tbody></tbody>
	</table>

	<button class="btn btn-info" data-toggle="collapse" data-target="#packets">View packets</button>
	<div id="packets" class="collapse">Request<pre id="req-packet"></pre>Response<pre id="res-packet"></pre></div>
    <hr>
	<button class="btn btn-info" data-toggle="collapse" data-target="#utilization">View utilization</button>
	<div id="utilization" class="collapse">
	    <pre>Memory: <span id="util-stats-memory"></span>%
CPU: <span id="util-stats-cpu"></span>%</pre>
	</div>
	
    </div></body>

	<script type="text/javascript">
	(function(){
		var conn = new WebSocket("ws://" + window.location.hostname + "{{.WsPort}}" +"/ws");
		conn.onclose = function(evt) {
            console.log("Connection closed");
        }
		conn.onmessage = function(evt) {
			var dr  = $("#dr_count");
			var sm  = $("#sm_count");
			var tps = $("#tps");
			var cost = $("#cost");
			var reqpackets = $("#req-packet");
			var respackets = $("#res-packet");
			var memory = $("#util-stats-memory");
			var cpu = $("#util-stats-cpu");
			dr.text(JSON.parse(evt.data).DRCount);
			sm.text(JSON.parse(evt.data).SMCount);
			tps.text(JSON.parse(evt.data).Tps);
			cost.text(JSON.parse(evt.data).Cost);
			reqpackets.text(JSON.parse(evt.data).IncomingPackets);
			respackets.text(JSON.parse(evt.data).OutgoingPackets);
			memory.text(JSON.parse(evt.data).MemoryPercent);
			cpu.text(JSON.parse(evt.data).CpuPercent);
			var elems = JSON.parse(evt.data).Messages;		
			if(elems !== undefined && elems.length > 0) {
				var tbody = $('#arrmsg tbody');
				tbody.empty();
    			var props = ["sender", "recipient", "message", "timestamp"];
    			$.each(elems, function(i, elem) {
      				var tr = $('<tr>');
      				$.each(props, function(i, prop) {
        				$('<td>').html(elem[prop]).appendTo(tr);
      				});
      				tbody.append(tr);
    			});
			}
			var clientConns = JSON.parse(evt.data).ClientConns;
			if(clientConns !== undefined && clientConns.length > 0) {
				var clientConnList = $("#client-conns");
				var parent = clientConnList.parent();
				clientConnList.detach().empty().each(function(i){
    				for (var x = 0; x < clientConns.length; x++){
        				$(this).append('<li class="list-group-item">' + clientConns[x] + '</li>');
						if (x == clientConns.length - 1){
            				$(this).appendTo(parent);
        				}
    				}
				});
			}
		}

    $('#resetHandler').submit(function(e){
			e.preventDefault();
			$.ajax({
				url:"/resetHandler",
				type:"post",
				success:function(){
					alert("Reset Submitted Messages");
				}
			});
		});


     $('#timeoutHandler').submit(function(e){
			e.preventDefault();
			$.ajax({
				url:"/timeoutHandler",
				type:"post",
				success:function(){
					alert("Simulate keepalive timeout");
				}
			});
		});

	})();
	</script>
</html>`
