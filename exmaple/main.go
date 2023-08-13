package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jart/gosip/sdp"
	"github.com/jart/gosip/sip"
	"github.com/jart/gosip/util"

	"github.com/shend/simplesip"
	"github.com/shend/simplesip/message"
	"github.com/shend/simplesip/transport"
)

func init() {
	InitLog()
}

func main() {
	go ListenAndServe()

	go ListenAndServeHttp()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	<-c
}

func InitLog() {
	options := slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	}

	var logWriters []io.Writer
	logWriters = append(logWriters, os.Stdout)

	handler := slog.NewTextHandler(io.MultiWriter(logWriters...), &options)
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

func ListenAndServe() {
	transport.SIPDebug = true

	srv, _ := simplesip.NewServer()

	srv.OnRegister(handleRegister)
	srv.OnInvite(handleInvite)
	srv.AddRequestMiddleware(handleServeRequest)

	addr := fmt.Sprintf("%s:%d", "127.0.0.1", 5060)
	srv.ListenAndServe(context.TODO(), "udp", addr)
}

func handleServeRequest(req *message.Message) {
	slog.Debug("This is a request middleware")
}

func handleRegister(req *message.Message) *message.Message {
	slog.Debug("Received REGISTER request")
	return nil
}

func handleInvite(req *message.Message) *message.Message {
	slog.Debug("Received INVITE request")
	res := simplesip.NewResponseFromRequest(req, 200, "OK")
	res.Msg.Payload = nil
	return res
}

func SendInviteMsg() string {
	invite := &sip.Msg{
		Method:  sip.MethodInvite,
		Request: &sip.URI{User: "echo", Host: "127.0.0.1", Port: 8888},
		Via: &sip.Via{
			Host: "sip.example.org",
			Port: 8888,
			Param: &sip.Param{
				Name:  "branch",
				Value: util.GenerateBranch(),
				Next:  &sip.Param{Name: "rport"},
			},
		},
		From: &sip.Addr{
			Display: "Alice",
			Uri: &sip.URI{
				Scheme: "sip",
				User:   "alic",
				Host:   "sip.example.org",
			},
			Param: &sip.Param{Name: "tag", Value: util.GenerateTag()},
		},
		To: &sip.Addr{
			Uri: &sip.URI{
				Scheme: "sip",
				User:   "bob",
				Host:   "sip.example.org",
			},
		},
		CallID:     util.GenerateCallID(),
		CSeq:       1,
		CSeqMethod: sip.MethodInvite,
		Date:       time.Now().Format(time.RFC1123),
		Contact: &sip.Addr{
			Uri: &sip.URI{
				Scheme: "sip",
				User:   "alice",
				Host:   "sip.example.org",
				Port:   8888,
			},
		},
		Payload: &sdp.SDP{
			Origin: sdp.Origin{ID: util.GenerateOriginID()},
			Audio: &sdp.Media{
				Port:   51002,
				Codecs: []sdp.Codec{sdp.ULAWCodec, sdp.DTMFCodec},
			},
		},
	}

	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:8888")
	if err != nil {
		log.Fatal(err)
	}

	dialer := net.Dialer{
		LocalAddr: addr,
	}
	conn, err := dialer.Dial("udp", "127.0.0.1:5060")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	raddr := conn.RemoteAddr().(*net.UDPAddr)

	var buf1 bytes.Buffer
	invite.Append(&buf1)

	log.Printf(fmt.Sprintf("Send INVITE request: %s:\n%s", raddr.IP.String(), buf1.String()))
	if amt, err := conn.Write(buf1.Bytes()); err != nil || amt != buf1.Len() {
		slog.Error("Send request failed", "err", err)
		return "error"
	}

	buf2 := make([]byte, 65535)
	n, err := conn.Read(buf2)

	return fmt.Sprintf("Send:\n%s\n\n\nRecv:\n%s", buf1.String(), string(buf2[:n]))
}

func sendMsg(w http.ResponseWriter, req *http.Request) {
	res := SendInviteMsg()
	io.WriteString(w, res)
}

func ListenAndServeHttp() {
	http.HandleFunc("/send", sendMsg)
	err := http.ListenAndServe(":3333", nil)
	if err != nil {
		log.Fatal(err)
	}
}
