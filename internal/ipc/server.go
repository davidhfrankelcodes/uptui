package ipc

import (
	"context"
	"encoding/json"
	"log"
	"net"

	"uptui/internal/models"
)

// Handler is implemented by the daemon to handle IPC requests.
type Handler interface {
	GetAllStatus() []*models.MonitorStatus
	AddMonitor(m models.Monitor) (*models.MonitorStatus, error)
	DeleteMonitor(name string) error
	PauseMonitor(name string) error
	ResumeMonitor(name string) error
	EditMonitor(oldName string, m models.Monitor) (*models.MonitorStatus, error)
	Reload() error
}

type Server struct {
	addr    string
	handler Handler
}

func NewServer(addr string, h Handler) *Server {
	return &Server{addr: addr, handler: h}
}

func (s *Server) Listen(ctx context.Context) error {
	l, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		l.Close()
	}()

	for {
		conn, err := l.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				log.Printf("ipc accept: %v", err)
				continue
			}
		}
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	dec := json.NewDecoder(conn)
	enc := json.NewEncoder(conn)

	for {
		var req Request
		if err := dec.Decode(&req); err != nil {
			return
		}
		resp := s.dispatch(req)
		if err := enc.Encode(resp); err != nil {
			return
		}
	}
}

func (s *Server) dispatch(req Request) Response {
	switch req.Action {
	case ActionList:
		return Response{OK: true, Monitors: s.handler.GetAllStatus()}

	case ActionAdd:
		if req.Monitor == nil {
			return Response{OK: false, Error: "monitor required"}
		}
		ms, err := s.handler.AddMonitor(*req.Monitor)
		if err != nil {
			return Response{OK: false, Error: err.Error()}
		}
		return Response{OK: true, Monitor: ms}

	case ActionDelete:
		if err := s.handler.DeleteMonitor(req.Name); err != nil {
			return Response{OK: false, Error: err.Error()}
		}
		return Response{OK: true}

	case ActionPause:
		if err := s.handler.PauseMonitor(req.Name); err != nil {
			return Response{OK: false, Error: err.Error()}
		}
		return Response{OK: true}

	case ActionResume:
		if err := s.handler.ResumeMonitor(req.Name); err != nil {
			return Response{OK: false, Error: err.Error()}
		}
		return Response{OK: true}

	case ActionEdit:
		if req.Monitor == nil {
			return Response{OK: false, Error: "monitor required"}
		}
		ms, err := s.handler.EditMonitor(req.OldName, *req.Monitor)
		if err != nil {
			return Response{OK: false, Error: err.Error()}
		}
		return Response{OK: true, Monitor: ms}

	case ActionReload:
		if err := s.handler.Reload(); err != nil {
			return Response{OK: false, Error: err.Error()}
		}
		return Response{OK: true}

	default:
		return Response{OK: false, Error: "unknown action"}
	}
}
