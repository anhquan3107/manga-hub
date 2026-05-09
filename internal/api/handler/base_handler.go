package handler

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	pb "mangahub/proto"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"mangahub/internal/auth"
	"mangahub/internal/chat"
	"mangahub/internal/config"
	"mangahub/internal/manga"
	"mangahub/internal/review"
	"mangahub/internal/user"
	chatws "mangahub/internal/websocket"
	"mangahub/pkg/models"
	"mangahub/pkg/utils"
)

type ProgressBroadcaster interface {
	PublishProgress(update models.ProgressUpdate)
}

type Dependencies struct {
	AuthService   *auth.Service
	ChatService   *chat.Service
	MangaService  *manga.Service
	ReviewService *review.Service
	UserService   *user.Service
	Hub           *chatws.Hub
	Broadcaster   ProgressBroadcaster
	Config        config.Config
}

type Handler struct {
	authService   *auth.Service
	chatService   *chat.Service
	mangaService  *manga.Service
	reviewService *review.Service
	userService   *user.Service
	hub           *chatws.Hub
	broadcaster   ProgressBroadcaster
	config        config.Config
}

func New(deps Dependencies) *Handler {
	return &Handler{
		authService:   deps.AuthService,
		chatService:   deps.ChatService,
		mangaService:  deps.MangaService,
		reviewService: deps.ReviewService,
		userService:   deps.UserService,
		hub:           deps.Hub,
		broadcaster:   deps.Broadcaster,
		config:        deps.Config,
	}
}

// Health godoc
// @Summary Comprehensive health check
// @Description Returns health status of all services (HTTP, gRPC, TCP, UDP).
// @Tags system
// @Produce json
// @Success 200 {object} healthResponse
// @Router /health [get]
func (h *Handler) Health(c *gin.Context) {
	resp := gin.H{
		"status": "ok",
		"services": gin.H{
			"http_api": gin.H{"status": "ok"},
			"grpc":     h.checkGRPC(),
			"tcp":      h.checkTCP(),
			"udp":      h.checkUDP(),
		},
	}
	utils.OK(c, http.StatusOK, resp)
}

func (h *Handler) checkGRPC() gin.H {
	result := gin.H{"status": "ok"}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	grpcAddr := strings.TrimPrefix(h.config.GRPCAddr, "tcp://")
	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		result["status"] = "error"
		result["error"] = err.Error()
		return result
	}
	defer conn.Close()

	client := pb.NewHealthServiceClient(conn)
	reply, err := client.Check(ctx, &pb.HealthCheckRequest{})
	if err != nil {
		result["status"] = "error"
		result["error"] = err.Error()
		return result
	}

	// Validate the response content
	if reply == nil || strings.ToLower(reply.GetStatus()) != "ok" {
		result["status"] = "error"
		if reply != nil {
			result["error"] = "unexpected grpc health status: " + reply.GetStatus()
		} else {
			result["error"] = "grpc health returned empty reply"
		}
		return result
	}

	return result
}

func (h *Handler) checkTCP() gin.H {
	result := gin.H{"status": "ok"}
	_, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, err := net.DialTimeout("tcp", h.config.TCPAddr, 2*time.Second)
	if err != nil {
		result["status"] = "error"
		result["error"] = err.Error()
		return result
	}
	defer conn.Close()

	// Send health check message
	healthMsg := map[string]string{"type": "health"}
	data, _ := json.Marshal(healthMsg)
	data = append(data, '\n')

	if err := conn.SetWriteDeadline(time.Now().Add(2 * time.Second)); err != nil {
		result["status"] = "error"
		result["error"] = err.Error()
		return result
	}
	if _, err := conn.Write(data); err != nil {
		result["status"] = "error"
		result["error"] = err.Error()
		return result
	}

	// Wait for a newline-terminated JSON response from the TCP server
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		result["status"] = "error"
		result["error"] = err.Error()
		return result
	}
	reader := bufio.NewReader(conn)
	respBytes, err := reader.ReadBytes('\n')
	if err != nil {
		result["status"] = "error"
		result["error"] = err.Error()
		return result
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		result["status"] = "error"
		result["error"] = "invalid response: " + err.Error()
		return result
	}
	if t, _ := resp["type"].(string); strings.ToLower(t) != "health_ok" {
		result["status"] = "error"
		result["error"] = "unexpected tcp health response"
		return result
	}

	return result
}

func (h *Handler) checkUDP() gin.H {
	result := gin.H{"status": "ok"}
	_, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	udpAddr, err := net.ResolveUDPAddr("udp", h.config.UDPAddr)
	if err != nil {
		result["status"] = "error"
		result["error"] = err.Error()
		return result
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		result["status"] = "error"
		result["error"] = err.Error()
		return result
	}
	defer conn.Close()

	// Send health check message
	healthMsg := map[string]string{"type": "health"}
	data, _ := json.Marshal(healthMsg)

	if err := conn.SetWriteDeadline(time.Now().Add(2 * time.Second)); err != nil {
		result["status"] = "error"
		result["error"] = err.Error()
		return result
	}
	if _, err := conn.Write(data); err != nil {
		result["status"] = "error"
		result["error"] = err.Error()
		return result
	}

	// Wait for a UDP response and validate
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		result["status"] = "error"
		result["error"] = err.Error()
		return result
	}
	buf := make([]byte, 2048)
	n, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		result["status"] = "error"
		result["error"] = err.Error()
		return result
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(buf[:n], &resp); err != nil {
		result["status"] = "error"
		result["error"] = "invalid response: " + err.Error()
		return result
	}
	if t, _ := resp["type"].(string); strings.ToLower(t) != "health_ok" {
		result["status"] = "error"
		result["error"] = "unexpected udp health response"
		return result
	}

	return result
}

func isNotFound(err error) bool {
	return errors.Is(err, http.ErrMissingFile) || strings.Contains(strings.ToLower(err.Error()), "no rows")
}
