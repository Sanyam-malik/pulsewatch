package websocket

import (
	"context"
	"github.com/sanyam-malik/pulsewatch/internal/config"
	"github.com/sanyam-malik/pulsewatch/internal/infra"
	"github.com/sanyam-malik/pulsewatch/internal/modules/auth"
	"github.com/sanyam-malik/pulsewatch/internal/modules/events"
	"github.com/sanyam-malik/pulsewatch/internal/modules/heartbeat"
	"github.com/sanyam-malik/pulsewatch/internal/modules/monitor"
	"net/http"
	"strings"

	"github.com/zishang520/socket.io/v2/socket"
	"go.uber.org/zap"
)

type Server struct {
	io         *socket.Server
	eventBus   events.EventBus
	tokenMaker *auth.TokenMaker
	identity   auth.Repository
}

type SocketData struct {
	UserId  string
	GroupID string
	Role    string
}

func NewServer(
	cfg *config.Config,
	eventBus events.EventBus,
	tokenMaker *auth.TokenMaker,
	identity auth.Repository,
	monitorService monitor.Service,
	logger *zap.SugaredLogger,
) (*Server, error) {
	opts := socket.DefaultServerOptions()
	io := socket.NewServer(nil, opts)

	server := &Server{
		io:         io,
		eventBus:   eventBus,
		tokenMaker: tokenMaker,
		identity:   identity,
	}

	io.Use(func(s *socket.Socket, next func(*socket.ExtendedError)) {
		access_token, ok := s.Request().Query().Get("token")
		if !ok {
			next(socket.NewExtendedError("access_token is required", "test"))
			return
		}

		// Create context for token verification
		ctx := context.Background()
		claims, err := tokenMaker.VerifyToken(ctx, access_token, "access")
		if err != nil {
			next(socket.NewExtendedError("Unauthorized", nil))
			return
		}

		user, err := identity.FindByID(ctx, claims.UserID)
		if err != nil || user == nil || !user.Active {
			next(socket.NewExtendedError("Unauthorized", nil))
			return
		}
		groupID, _ := s.Request().Query().Get("groupId")
		if groupID == "" {
			groupID = claims.GroupID
		}
		membership, err := identity.ResolveMembership(ctx, user.ID, groupID)
		if err != nil || membership == nil {
			if groupID != "" {
				next(socket.NewExtendedError("Unauthorized", nil))
				return
			}
			membership, err = identity.ResolveMembership(ctx, user.ID, "")
			if err != nil || membership == nil {
				next(socket.NewExtendedError("Unauthorized", nil))
				return
			}
		}
		data := SocketData{UserId: user.ID, GroupID: membership.GroupID, Role: membership.Role}

		s.SetData(data)

		next(nil)
	})

	io.On("connection", func(clients ...interface{}) {
		client := clients[0].(*socket.Socket)
		data := client.Data().(SocketData)
		userId := data.UserId

		logger.Debugf("[WS]connection: %s", userId)

		client.On("join_room", func(args ...interface{}) {
			if len(args) == 0 {
				return
			}
			roomName, ok := args[0].(string)
			if !ok {
				return
			}
			logger.Debugf("join_room: %s", roomName)
			switch {
			case roomName == "monitor:all":
				client.Join(socket.Room("group:" + data.GroupID + ":monitor:all"))
			case strings.HasPrefix(roomName, "monitor:"):
				monitorID := strings.TrimPrefix(roomName, "monitor:")
				ctx := auth.WithIdentity(context.Background(), data.GroupID, data.Role)
				visibleMonitor, err := monitorService.FindByID(ctx, monitorID)
				if err != nil || visibleMonitor == nil {
					logger.Warnw("Rejected websocket room access", "userId", userId, "room", roomName)
					return
				}
				client.Join(socket.Room(roomName))
			default:
				logger.Warnw("Rejected unsupported websocket room", "userId", userId, "room", roomName)
			}
		})

		client.On("leave_room", func(args ...interface{}) {
			if len(args) == 0 {
				return
			}
			roomName, ok := args[0].(string)
			if !ok {
				return
			}
			logger.Debugf("leave_room: %s", roomName)
			if roomName == "monitor:all" {
				client.Leave(socket.Room("group:" + data.GroupID + ":monitor:all"))
			} else {
				client.Leave(socket.Room(roomName))
			}
		})
	})

	// Listen for heartbeat events and broadcast to room
	eventBus.Subscribe(events.HeartbeatEvent, func(event events.Event) {
		hb, ok := infra.UnmarshalEventPayload[heartbeat.Model](event)
		if !ok {
			logger.Warn("Failed to unmarshal heartbeat event payload")
			return
		}
		roomName := "monitor:" + hb.MonitorID
		server.io.To(socket.Room(roomName)).Emit(roomName+":heartbeat", hb)
		mon, err := monitorService.FindByID(context.Background(), hb.MonitorID)
		if err != nil || mon == nil || mon.GroupID == "" {
			return
		}
		server.io.To(socket.Room("group:"+mon.GroupID+":monitor:all")).Emit("monitor:all:heartbeat", hb)
	})

	return server, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.io.ServeHandler(nil).ServeHTTP(w, r)
}

func (s *Server) Close() {
	s.io.Close(nil)
}
