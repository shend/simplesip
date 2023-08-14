package simplesip

import (
	"context"
	"log/slog"
	"net"

	"github.com/shend/simplesip/message"
	"github.com/shend/simplesip/parser"
	"github.com/shend/simplesip/transport"

	jartsip "github.com/jart/gosip/sip"
	jartutil "github.com/jart/gosip/util"
)

// Server is a SIP server
type Server struct {
	tp *transport.Layer
	// requestHandlers map of all registered request handlers
	requestHandlers map[message.RequestMethod]message.RequestHandler
	noRouteHandler  message.RequestHandler

	requestMiddlewares  []message.RequestMiddleware
	responseMiddlewares []message.ResponseMiddleware
}

func NewServer() (*Server, error) {
	s := &Server{
		tp:                  transport.NewLayer(parser.NewParser()),
		requestHandlers:     make(map[message.RequestMethod]message.RequestHandler),
		requestMiddlewares:  make([]message.RequestMiddleware, 1),
		responseMiddlewares: make([]message.ResponseMiddleware, 1),
	}

	s.requestMiddlewares[0] = s.defaultRequestMiddleware
	s.responseMiddlewares[0] = s.defaultResponseMiddleware
	s.noRouteHandler = s.defaultUnhandledHandler

	return s, nil
}

// ListenAndServe will fire all listeners. Ctx allows canceling
func (srv *Server) ListenAndServe(ctx context.Context, network string, addr string) error {
	srv.tp.AppendHandlers(srv.handleRequest)
	return srv.tp.ListenAndServe(network, addr)
}

// ServeUDP starts serving request on UDP type listener.
func (srv *Server) ServeUDP(l net.PacketConn) error {
	srv.tp.AppendHandlers(srv.handleRequest)
	return srv.tp.ServeUDP(l)
}

// ServeTCP starts serving request on TCP type listener.
func (srv *Server) ServeTCP(l net.Listener) error {
	srv.tp.AppendHandlers(srv.handleRequest)
	return srv.tp.ServeTCP(l)
}

// onRequest gets request from Transaction layer
func (srv *Server) onRequest(req *message.Message) {
	go srv.handleRequest(req)
}

// handleRequest must be run in separate goroutine
func (srv *Server) handleRequest(req *message.Message) *message.Message {
	for _, mid := range srv.requestMiddlewares {
		mid(req)
	}

	var method string
	if req.Msg.IsResponse() {
		method = req.Msg.CSeqMethod
	} else {
		method = req.Msg.Method
	}
	handler := srv.getHandler(message.FromString(method))
	res := handler(req)

	final := false
	for _, mid := range srv.responseMiddlewares {
		if final {
			break
		}
		final = mid(res)
	}

	return nil
}

// WriteResponse will proxy message to transport layer. Use it in stateless mode
func (srv *Server) WriteResponse(r *message.Message) error {
	return srv.tp.WriteMsg(r)
}

// Close gracefully shutdowns SIP server
func (srv *Server) Close() {
	// stop transport layer
	srv.tp.Close()
}

// OnInvite registers Invite request handler
func (srv *Server) OnInvite(handler message.RequestHandler) {
	srv.requestHandlers[message.INVITE] = handler
}

// OnAck registers Ack request handler
func (srv *Server) OnAck(handler message.RequestHandler) {
	srv.requestHandlers[message.ACK] = handler
}

// OnCancel registers Cancel request handler
func (srv *Server) OnCancel(handler message.RequestHandler) {
	srv.requestHandlers[message.CANCEL] = handler
}

// OnBye registers Bye request handler
func (srv *Server) OnBye(handler message.RequestHandler) {
	srv.requestHandlers[message.BYE] = handler
}

// OnRegister registers Register request handler
func (srv *Server) OnRegister(handler message.RequestHandler) {
	srv.requestHandlers[message.REGISTER] = handler
}

// OnOptions registers Options request handler
func (srv *Server) OnOptions(handler message.RequestHandler) {
	srv.requestHandlers[message.OPTIONS] = handler
}

// OnSubscribe registers Subscribe request handler
func (srv *Server) OnSubscribe(handler message.RequestHandler) {
	srv.requestHandlers[message.SUBSCRIBE] = handler
}

// OnNotify registers Notify request handler
func (srv *Server) OnNotify(handler message.RequestHandler) {
	srv.requestHandlers[message.NOTIFY] = handler
}

// OnRefer registers Refer request handler
func (srv *Server) OnRefer(handler message.RequestHandler) {
	srv.requestHandlers[message.REFER] = handler
}

// OnInfo registers Info request handler
func (srv *Server) OnInfo(handler message.RequestHandler) {
	srv.requestHandlers[message.INFO] = handler
}

// OnMessage registers Message request handler
func (srv *Server) OnMessage(handler message.RequestHandler) {
	srv.requestHandlers[message.MESSAGE] = handler
}

// OnPrack registers Prack request handler
func (srv *Server) OnPrack(handler message.RequestHandler) {
	srv.requestHandlers[message.PRACK] = handler
}

// OnUpdate registers Update request handler
func (srv *Server) OnUpdate(handler message.RequestHandler) {
	srv.requestHandlers[message.UPDATE] = handler
}

// OnPublish registers Publish request handler
func (srv *Server) OnPublish(handler message.RequestHandler) {
	srv.requestHandlers[message.PUBLISH] = handler
}

func (srv *Server) getHandler(method message.RequestMethod) (handler message.RequestHandler) {
	handler, ok := srv.requestHandlers[method]
	if !ok {
		return srv.defaultUnhandledHandler
	}
	return handler
}

func (srv *Server) defaultRequestMiddleware(req *message.Message) {
	hTo := req.Msg.To.Copy()
	hToParam := hTo.Param
	if hToParam.Get("tag") != nil {
		return
	}
	hToTagParam := &jartsip.Param{
		Name:  "tag",
		Value: jartutil.GenerateTag(),
	}
	if hToParam == nil {
		hToParam = hToTagParam
	} else {
		hToParam.Next = hToTagParam
	}
	hTo.Param = hToParam
	req.Msg.To = hTo
}

func (srv *Server) defaultResponseMiddleware(res *message.Message) bool {
	if res != nil {
		if err := srv.WriteResponse(res); err != nil {
			slog.Error("respond failed", "err", err)
		}
	}
	return true
}

func (srv *Server) defaultUnhandledHandler(req *message.Message) *message.Message {
	slog.Warn("SIP request handler not found")
	res := NewResponseFromRequest(req, 405, "Method Not Allowed")
	res.Msg.Payload = nil
	if err := srv.WriteResponse(res); err != nil {
		slog.Error("respond '405 Method Not Allowed' failed", "err", err)
	}
	return nil
}

// ReplaceDefaultRequestMiddlware adds a middleware to preprocessing message
func (srv *Server) ReplaceDefaultRequestMiddlware(f message.RequestMiddleware) {
	srv.requestMiddlewares[0] = f
}

// AddRequestMiddleware adds a middleware to preprocessing message
func (srv *Server) AddRequestMiddleware(f message.RequestMiddleware) {
	srv.requestMiddlewares = append(srv.requestMiddlewares, f)
}

// TransportLayer is function to get transport layer of server
// Can be used for modifying
func (srv *Server) TransportLayer() *transport.Layer {
	return srv.tp
}
