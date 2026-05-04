package grpcservice

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"mangahub/internal/grpc/grpcjson"
	"mangahub/internal/manga"
	"mangahub/internal/user"
	"mangahub/pkg/models"
)

type Service struct {
	addr         string
	mangaService *manga.Service
	userService  *user.Service
	server       *grpc.Server
	listener     net.Listener
	once         sync.Once
}

type MangaRequest struct {
	Id    string `json:"id,omitempty"`
	Query string `json:"query,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

type MangaResponse struct {
	Manga  *models.Manga `json:"manga,omitempty"`
	Items  []models.Manga `json:"items,omitempty"`
	Status string         `json:"status,omitempty"`
	Error  string         `json:"error,omitempty"`
}

type ProgressRequest struct {
	UserID  string `json:"user_id,omitempty"`
	MangaID string `json:"manga_id,omitempty"`
	Chapter int    `json:"chapter,omitempty"`
	Volume  int    `json:"volume,omitempty"`
	Notes   string `json:"notes,omitempty"`
	Force   bool   `json:"force,omitempty"`
}

type ProgressResponse struct {
	Result *models.ProgressUpdateResult `json:"result,omitempty"`
	Error  string                       `json:"error,omitempty"`
}

type mangaHubServer struct {
	mangaService *manga.Service
	userService  *user.Service
}

type mangaHubService interface {
	GetManga(context.Context, *MangaRequest) (*MangaResponse, error)
	SearchManga(context.Context, *MangaRequest) (*MangaResponse, error)
	UpdateProgress(context.Context, *ProgressRequest) (*ProgressResponse, error)
}

func New(addr string, mangaService *manga.Service, userService *user.Service) *Service {
	return &Service{addr: addr, mangaService: mangaService, userService: userService}
}

func (s *Service) Start(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen grpc: %w", err)
	}
	s.listener = listener

	grpcServer := grpc.NewServer(grpc.ForceServerCodec(grpcjson.Codec{}))
	s.server = grpcServer
	grpcServer.RegisterService(&mangaHubServiceDesc, &mangaHubServer{mangaService: s.mangaService, userService: s.userService})

	go func() {
		<-ctx.Done()
		grpcServer.Stop()
		_ = listener.Close()
	}()

	log.Printf("grpc server listening on %s", s.addr)
	return grpcServer.Serve(listener)
}

func (s *Service) Stop() {
	s.once.Do(func() {
		if s.server != nil {
			s.server.Stop()
		}
		if s.listener != nil {
			_ = s.listener.Close()
		}
	})
}

func (h *mangaHubServer) GetManga(ctx context.Context, req *MangaRequest) (*MangaResponse, error) {
	if strings.TrimSpace(req.Id) == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	item, err := h.mangaService.GetByID(ctx, req.Id)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	return &MangaResponse{Manga: &item, Status: "ok"}, nil
}

func (h *mangaHubServer) SearchManga(ctx context.Context, req *MangaRequest) (*MangaResponse, error) {
	items, err := h.mangaService.List(ctx, models.MangaQuery{Query: req.Query, Limit: req.Limit})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &MangaResponse{Items: items, Status: "ok"}, nil
}

func (h *mangaHubServer) UpdateProgress(ctx context.Context, req *ProgressRequest) (*ProgressResponse, error) {
	userID := strings.TrimSpace(req.UserID)
	if userID == "" {
		userID = "default-user"
	}
	result, err := h.userService.UpdateProgress(ctx, userID, models.UpdateProgressRequest{
		MangaID:        req.MangaID,
		CurrentChapter: req.Chapter,
		CurrentVolume:  req.Volume,
		Notes:          req.Notes,
		Force:          req.Force,
	})
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &ProgressResponse{Result: &result}, nil
}

var mangaHubServiceDesc = grpc.ServiceDesc{
	ServiceName: "mangahub.MangaHub",
	HandlerType: (*mangaHubService)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "GetManga", Handler: getMangaHandler},
		{MethodName: "SearchManga", Handler: searchMangaHandler},
		{MethodName: "UpdateProgress", Handler: updateProgressHandler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "mangahub",
}

func getMangaHandler(srv any, ctx context.Context, dec func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
	req := new(MangaRequest)
	if err := dec(req); err != nil {
		return nil, err
	}
	return srv.(*mangaHubServer).GetManga(ctx, req)
}

func searchMangaHandler(srv any, ctx context.Context, dec func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
	req := new(MangaRequest)
	if err := dec(req); err != nil {
		return nil, err
	}
	return srv.(*mangaHubServer).SearchManga(ctx, req)
}

func updateProgressHandler(srv any, ctx context.Context, dec func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
	req := new(ProgressRequest)
	if err := dec(req); err != nil {
		return nil, err
	}
	return srv.(*mangaHubServer).UpdateProgress(ctx, req)
}