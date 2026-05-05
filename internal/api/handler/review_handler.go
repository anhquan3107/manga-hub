package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"mangahub/pkg/models"
	"mangahub/pkg/utils"
)

// ListReviews godoc
// @Summary List manga reviews
// @Description Returns reviews for a manga.
// @Tags reviews
// @Produce json
// @Param id path string true "Manga ID"
// @Param limit query int false "Max reviews (1-200)"
// @Param sort query string false "Sort by (recent, helpful)"
// @Success 200 {object} reviewListResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /manga/{id}/reviews [get]
func (h *Handler) ListReviews(c *gin.Context) {
	mangaID := c.Param("id")

	limit := 50
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			utils.Error(c, http.StatusBadRequest, "invalid limit")
			return
		}
		if parsed > 200 {
			parsed = 200
		}
		limit = parsed
	}

	sortBy := strings.ToLower(strings.TrimSpace(c.DefaultQuery("sort", "recent")))
	if sortBy != "recent" && sortBy != "helpful" {
		utils.Error(c, http.StatusBadRequest, "invalid sort")
		return
	}

	reviews, err := h.reviewService.ListReviews(c.Request.Context(), mangaID, limit, sortBy)
	if err != nil {
		if isNotFound(err) {
			utils.Error(c, http.StatusNotFound, "manga not found")
			return
		}
		utils.Error(c, http.StatusInternalServerError, "failed to fetch reviews")
		return
	}

	utils.OK(c, http.StatusOK, gin.H{"items": reviews})
}

// UpsertReview godoc
// @Summary Create or update a review
// @Description Creates a new review or updates the existing one for the user.
// @Tags reviews
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Manga ID"
// @Param body body models.CreateReviewRequest true "Review payload"
// @Success 200 {object} models.Review
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /manga/{id}/reviews [post]
func (h *Handler) UpsertReview(c *gin.Context) {
	var req models.CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	userID := currentUserID(c)
	if userID == "" {
		utils.Error(c, http.StatusUnauthorized, "missing user id")
		return
	}

	mangaID := c.Param("id")
	review, err := h.reviewService.UpsertReview(c.Request.Context(), userID, mangaID, req)
	if err != nil {
		if isNotFound(err) {
			utils.Error(c, http.StatusNotFound, "manga not found")
			return
		}
		utils.Error(c, http.StatusInternalServerError, "failed to save review")
		return
	}

	utils.OK(c, http.StatusOK, review)
}

// GetMyReview godoc
// @Summary Get my review
// @Description Returns the authenticated user's review for a manga.
// @Tags reviews
// @Produce json
// @Security BearerAuth
// @Param id path string true "Manga ID"
// @Success 200 {object} models.Review
// @Failure 404 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /manga/{id}/reviews/me [get]
func (h *Handler) GetMyReview(c *gin.Context) {
	userID := currentUserID(c)
	if userID == "" {
		utils.Error(c, http.StatusUnauthorized, "missing user id")
		return
	}

	mangaID := c.Param("id")
	review, err := h.reviewService.GetReview(c.Request.Context(), userID, mangaID)
	if err != nil {
		if isNotFound(err) {
			utils.Error(c, http.StatusNotFound, "review not found")
			return
		}
		utils.Error(c, http.StatusInternalServerError, "failed to fetch review")
		return
	}

	utils.OK(c, http.StatusOK, review)
}

// MarkReviewHelpful godoc
// @Summary Mark review as helpful
// @Description Increments helpful votes for a review.
// @Tags reviews
// @Produce json
// @Security BearerAuth
// @Param id path string true "Manga ID"
// @Param user_id path string true "Review author user ID"
// @Success 200 {object} models.Review
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /manga/{id}/reviews/{user_id}/helpful [post]
func (h *Handler) MarkReviewHelpful(c *gin.Context) {
	currentID := currentUserID(c)
	if currentID == "" {
		utils.Error(c, http.StatusUnauthorized, "missing user id")
		return
	}

	mangaID := c.Param("id")
	reviewUserID := c.Param("user_id")
	if reviewUserID == "" {
		utils.Error(c, http.StatusBadRequest, "review user id required")
		return
	}
	if reviewUserID == currentID {
		utils.Error(c, http.StatusBadRequest, "cannot vote helpful on your own review")
		return
	}

	review, err := h.reviewService.IncrementHelpful(c.Request.Context(), reviewUserID, mangaID)
	if err != nil {
		if isNotFound(err) {
			utils.Error(c, http.StatusNotFound, "review not found")
			return
		}
		utils.Error(c, http.StatusInternalServerError, "failed to update helpful count")
		return
	}

	utils.OK(c, http.StatusOK, review)
}
