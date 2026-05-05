package handler

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"mangahub/pkg/models"
	"mangahub/pkg/utils"
)

// ListManga godoc
// @Summary List manga
// @Description Returns manga list with optional filters.
// @Tags manga
// @Produce json
// @Param q query string false "Search query"
// @Param genres query string false "Comma-separated genres"
// @Param genre query string false "Single genre"
// @Param status query string false "Manga status"
// @Param year_min query int false "Minimum year"
// @Param year_max query int false "Maximum year"
// @Param rating_min query number false "Minimum rating"
// @Param sort query string false "Sort field"
// @Param limit query int false "Result limit"
// @Success 200 {object} mangaListResponse
// @Failure 500 {object} errorResponse
// @Router /manga [get]
func (h *Handler) ListManga(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	yearMin, _ := strconv.Atoi(c.DefaultQuery("year_min", "0"))
	yearMax, _ := strconv.Atoi(c.DefaultQuery("year_max", "0"))
	ratingMin, _ := strconv.ParseFloat(c.DefaultQuery("rating_min", "0"), 64)

	genresParam := strings.TrimSpace(c.Query("genres"))
	genres := make([]string, 0)
	if genresParam != "" {
		for _, genre := range strings.Split(genresParam, ",") {
			genre = strings.TrimSpace(genre)
			if genre != "" {
				genres = append(genres, genre)
			}
		}
	}
	if genre := strings.TrimSpace(c.Query("genre")); genre != "" {
		genres = append(genres, genre)
	}

	items, err := h.mangaService.List(c.Request.Context(), models.MangaQuery{
		Query: c.Query("q"),
		Filters: models.SearchFilters{
			Genres:    genres,
			Status:    c.Query("status"),
			YearRange: [2]int{yearMin, yearMax},
			Rating:    ratingMin,
			SortBy:    c.Query("sort"),
		},
		Limit: limit,
	})
	if err != nil {
		log.Printf("list manga error: %v", err)
		utils.Error(c, http.StatusInternalServerError, "failed to fetch manga")
		return
	}

	utils.OK(c, http.StatusOK, gin.H{"items": items})
}

// GetManga godoc
// @Summary Get manga
// @Description Returns manga details by ID.
// @Tags manga
// @Produce json
// @Param id path string true "Manga ID"
// @Success 200 {object} models.Manga
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /manga/{id} [get]
func (h *Handler) GetManga(c *gin.Context) {
	mangaID := c.Param("id")

	item, err := h.mangaService.GetByID(c.Request.Context(), mangaID)
	if err != nil {
		if isNotFound(err) {
			utils.Error(c, http.StatusNotFound, "manga not found")
			return
		}
		utils.Error(c, http.StatusInternalServerError, "failed to fetch manga")
		return
	}

	utils.OK(c, http.StatusOK, item)
}

// CreateManga godoc
// @Summary Create manga
// @Description Creates a new manga record.
// @Tags manga
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body models.CreateMangaRequest true "Create manga payload"
// @Success 201 {object} models.Manga
// @Failure 400 {object} errorResponse
// @Failure 409 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /manga [post]
func (h *Handler) CreateManga(c *gin.Context) {
	var req models.CreateMangaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	item, err := h.mangaService.Create(c.Request.Context(), req)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			utils.Error(c, http.StatusConflict, "manga id already exists")
			return
		}
		utils.Error(c, http.StatusInternalServerError, "failed to create manga")
		return
	}

	utils.OK(c, http.StatusCreated, item)
}

// UpdateManga godoc
// @Summary Update manga
// @Description Updates an existing manga by ID.
// @Tags manga
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Manga ID"
// @Param body body models.UpdateMangaRequest true "Update manga payload"
// @Success 200 {object} models.Manga
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /manga/{id} [put]
func (h *Handler) UpdateManga(c *gin.Context) {
	var req models.UpdateMangaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	mangaID := c.Param("id")

	item, err := h.mangaService.Update(c.Request.Context(), mangaID, req)
	if err != nil {
		if isNotFound(err) {
			utils.Error(c, http.StatusNotFound, "manga not found")
			return
		}
		utils.Error(c, http.StatusInternalServerError, "failed to update manga")
		return
	}

	utils.OK(c, http.StatusOK, item)
}

// DeleteManga godoc
// @Summary Delete manga
// @Description Deletes a manga by ID.
// @Tags manga
// @Produce json
// @Security BearerAuth
// @Param id path string true "Manga ID"
// @Success 200 {object} messageResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /manga/{id} [delete]
func (h *Handler) DeleteManga(c *gin.Context) {
	mangaID := c.Param("id")

	err := h.mangaService.Delete(c.Request.Context(), mangaID)
	if err != nil {
		if isNotFound(err) {
			utils.Error(c, http.StatusNotFound, "manga not found")
			return
		}
		utils.Error(c, http.StatusInternalServerError, "failed to delete manga")
		return
	}

	utils.OK(c, http.StatusOK, gin.H{"message": "manga deleted"})
}
