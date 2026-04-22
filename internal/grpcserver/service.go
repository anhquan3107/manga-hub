package grpcserver

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"

	"mangahub/internal/grpcjson"
	"mangahub/pkg/database"
	"mangahub/pkg/models"
)

type MangaServiceServer interface {
	GetManga(context.Context, *GetMangaRequest) (*MangaResponse, error)
	SearchManga(context.Context, *SearchRequest) (*SearchResponse, error)
	UpdateProgress(context.Context, *ProgressRequest) (*ProgressResponse, error)
}

type Service struct {
	store *database.Store
}

type GetMangaRequest struct {
	ID string `json:"id"`
}

type MangaResponse struct {
	Manga models.Manga `json:"manga"`
}

type SearchRequest struct {
	Query  string `json:"query"`
	Genre  string `json:"genre"`
	Status string `json:"status"`
	Limit  int    `json:"limit"`
}

type SearchResponse struct {
	Items []models.Manga `json:"items"`
}

type ProgressRequest struct {
	UserID         string `json:"user_id"`
	MangaID        string `json:"manga_id"`
	CurrentChapter int    `json:"current_chapter"`
	Status         string `json:"status"`
}

type ProgressResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func NewService(store *database.Store) *Service {
	return &Service{store: store}
}

func ServerOptions() []grpc.ServerOption {
	codec := grpcjson.Codec{}
	encoding.RegisterCodec(codec)
	return []grpc.ServerOption{grpc.ForceServerCodec(codec)}
}

func Register(server *grpc.Server, service MangaServiceServer) {
	server.RegisterService(&grpc.ServiceDesc{
		ServiceName: "mangahub.MangaService",
		HandlerType: (*MangaServiceServer)(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: "GetManga",
				Handler:    getMangaHandler(service),
			},
			{
				MethodName: "SearchManga",
				Handler:    searchMangaHandler(service),
			},
			{
				MethodName: "UpdateProgress",
				Handler:    updateProgressHandler(service),
			},
		},
		Streams:  []grpc.StreamDesc{},
		Metadata: "proto/mangahub.proto",
	}, service)
}

func (s *Service) GetManga(ctx context.Context, req *GetMangaRequest) (*MangaResponse, error) {
	manga, err := s.store.GetMangaByID(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	return &MangaResponse{Manga: manga}, nil
}

func (s *Service) SearchManga(ctx context.Context, req *SearchRequest) (*SearchResponse, error) {
	items, err := s.store.ListManga(ctx, models.MangaQuery{
		Query:  req.Query,
		Genre:  req.Genre,
		Status: req.Status,
		Limit:  req.Limit,
	})
	if err != nil {
		return nil, err
	}
	return &SearchResponse{Items: items}, nil
}

func (s *Service) UpdateProgress(ctx context.Context, req *ProgressRequest) (*ProgressResponse, error) {
	_, err := s.store.UpsertLibraryEntry(ctx, req.UserID, models.LibraryEntry{
		MangaID:        req.MangaID,
		CurrentChapter: req.CurrentChapter,
		Status:         req.Status,
	})
	if err != nil {
		return nil, err
	}

	return &ProgressResponse{
		Success: true,
		Message: "progress updated",
	}, nil
}

func getMangaHandler(service MangaServiceServer) grpc.MethodHandler {
	return func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
		in := new(GetMangaRequest)
		if err := dec(in); err != nil {
			return nil, err
		}
		if interceptor == nil {
			return service.GetManga(ctx, in)
		}
		info := &grpc.UnaryServerInfo{
			Server:     srv,
			FullMethod: "/mangahub.MangaService/GetManga",
		}
		handler := func(ctx context.Context, req any) (any, error) {
			return service.GetManga(ctx, req.(*GetMangaRequest))
		}
		return interceptor(ctx, in, info, handler)
	}
}

func searchMangaHandler(service MangaServiceServer) grpc.MethodHandler {
	return func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
		in := new(SearchRequest)
		if err := dec(in); err != nil {
			return nil, err
		}
		if interceptor == nil {
			return service.SearchManga(ctx, in)
		}
		info := &grpc.UnaryServerInfo{
			Server:     srv,
			FullMethod: "/mangahub.MangaService/SearchManga",
		}
		handler := func(ctx context.Context, req any) (any, error) {
			return service.SearchManga(ctx, req.(*SearchRequest))
		}
		return interceptor(ctx, in, info, handler)
	}
}

func updateProgressHandler(service MangaServiceServer) grpc.MethodHandler {
	return func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
		in := new(ProgressRequest)
		if err := dec(in); err != nil {
			return nil, err
		}
		if interceptor == nil {
			return service.UpdateProgress(ctx, in)
		}
		info := &grpc.UnaryServerInfo{
			Server:     srv,
			FullMethod: "/mangahub.MangaService/UpdateProgress",
		}
		handler := func(ctx context.Context, req any) (any, error) {
			return service.UpdateProgress(ctx, req.(*ProgressRequest))
		}
		return interceptor(ctx, in, info, handler)
	}
}
