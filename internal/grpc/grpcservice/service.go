package grpcservice

import (
	"context"
	"net"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"mangahub/internal/grpc/pb"
	"mangahub/internal/manga"
	"mangahub/internal/user"
	"mangahub/pkg/models"
)

type Server struct {
	addr       string
	grpcServer *grpc.Server
	manga      *manga.Service
	user       *user.Service
}

func New(addr string, mangaService *manga.Service, userService *user.Service) *Server {
	return &Server{
		addr:  addr,
		manga: mangaService,
		user:  userService,
	}
}

func (s *Server) Start(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

	s.grpcServer = grpc.NewServer()
	pb.RegisterMangaServiceServer(s.grpcServer, &mangaServer{manga: s.manga, user: s.user})
	pb.RegisterUserServiceServer(s.grpcServer, &userServer{user: s.user})

	go func() {
		<-ctx.Done()
		s.grpcServer.GracefulStop()
	}()

	return s.grpcServer.Serve(listener)
}

func (s *Server) Stop() {
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}
}

type mangaServer struct {
	pb.UnimplementedMangaServiceServer
	manga *manga.Service
	user  *user.Service
}

func (s *mangaServer) GetManga(ctx context.Context, req *pb.GetMangaRequest) (*pb.MangaResponse, error) {
	mangaID := strings.TrimSpace(req.GetId())
	if mangaID == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	result, err := s.manga.GetByID(ctx, mangaID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "manga not found")
	}

	return &pb.MangaResponse{
		Manga:  mapManga(result),
		Status: "ok",
	}, nil
}

func (s *mangaServer) SearchManga(ctx context.Context, req *pb.SearchRequest) (*pb.SearchResponse, error) {
	query := strings.TrimSpace(req.GetQuery())
	if query == "" {
		return nil, status.Error(codes.InvalidArgument, "query is required")
	}

	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 20
	}

	items, err := s.manga.List(ctx, models.MangaQuery{Query: query, Limit: limit})
	if err != nil {
		return nil, status.Error(codes.Internal, "search failed")
	}

	resp := &pb.SearchResponse{Status: "ok"}
	for _, item := range items {
		resp.Items = append(resp.Items, mapManga(item))
	}
	return resp, nil
}

func (s *mangaServer) UpdateProgress(ctx context.Context, req *pb.ProgressRequest) (*pb.ProgressResponse, error) {
	userID := strings.TrimSpace(req.GetUserId())
	mangaID := strings.TrimSpace(req.GetMangaId())
	if userID == "" || mangaID == "" || req.GetChapter() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id, manga_id, and chapter are required")
	}

	result, err := s.user.UpdateProgress(ctx, userID, models.UpdateProgressRequest{
		MangaID:        mangaID,
		CurrentChapter: int(req.GetChapter()),
		CurrentVolume:  int(req.GetVolume()),
		Notes:          req.GetNotes(),
		Force:          req.GetForce(),
	})
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &pb.ProgressResponse{Result: mapProgressResult(result)}, nil
}

type userServer struct {
	pb.UnimplementedUserServiceServer
	user *user.Service
}

func (s *userServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.UserResponse, error) {
	userID := strings.TrimSpace(req.GetUserId())
	username := strings.TrimSpace(req.GetUsername())
	if userID == "" && username == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id or username is required")
	}

	var result models.User
	var err error
	if userID != "" {
		result, err = s.user.GetUserByID(ctx, userID)
	} else {
		result, err = s.user.GetUserByUsername(ctx, username)
	}
	if err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	return &pb.UserResponse{User: mapUser(result)}, nil
}

func (s *userServer) GetLibrary(ctx context.Context, req *pb.GetLibraryRequest) (*pb.LibraryResponse, error) {
	userID := strings.TrimSpace(req.GetUserId())
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	items, err := s.user.GetLibrary(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "library lookup failed")
	}

	resp := &pb.LibraryResponse{}
	for _, item := range items {
		resp.Entries = append(resp.Entries, mapLibraryEntry(item))
	}
	return resp, nil
}

func mapManga(m models.Manga) *pb.Manga {
	return &pb.Manga{
		Id:            m.ID,
		Title:         m.Title,
		Author:        m.Author,
		Genres:        m.Genres,
		Status:        m.Status,
		TotalChapters: int32(m.TotalChapters),
		Description:   m.Description,
		CoverUrl:      m.CoverURL,
	}
}

func mapLibraryEntry(e models.LibraryEntry) *pb.LibraryEntry {
	return &pb.LibraryEntry{
		UserId:         e.UserID,
		MangaId:        e.MangaID,
		Title:          e.Title,
		CurrentChapter: int32(e.CurrentChapter),
		CurrentVolume:  int32(e.CurrentVolume),
		Status:         e.Status,
		Rating:         int32(e.Rating),
		Notes:          e.Notes,
	}
}

func mapProgressResult(r models.ProgressUpdateResult) *pb.ProgressResult {
	return &pb.ProgressResult{
		Entry:           mapLibraryEntry(r.Entry),
		PreviousChapter: int32(r.PreviousChapter),
		PreviousVolume:  int32(r.PreviousVolume),
		TotalChapters:   int32(r.TotalChapters),
		MangaTitle:      r.MangaTitle,
	}
}

func mapUser(u models.User) *pb.User {
	return &pb.User{
		Id:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
