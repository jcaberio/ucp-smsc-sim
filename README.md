ucp-smsc-sim
============

UCP SMSC simulator for testing UCP clients.

![sample](https://i.imgur.com/ydSTLSC.png)

Supported operations
--------------------
- Alert
- Session Management
- Submit Short Message
- Delivery Notification
- Delivery Short Message

Dependencies
------------
* [Go](https://golang.org) 1.11
* [Redis](https://redis.io/) on localhost:6379

Run
---
```
$ go run main.go
```

Open http://localhost:16003 on your browser
