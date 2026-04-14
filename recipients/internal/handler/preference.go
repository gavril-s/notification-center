package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"notification-center/recipients/internal/domain"
	"notification-center/recipients/internal/middleware"
	"notification-center/recipients/internal/service"
)

type PreferenceHandler struct {
	prefService *service.PreferenceService
}

func NewPreferenceHandler(prefService *service.PreferenceService) *PreferenceHandler {
	return &PreferenceHandler{prefService: prefService}
}

type PreferenceResponse struct {
	PreferenceID    string   `json:"preference_id"`
	ContactID       string   `json:"contact_id"`
	SenderID        *string  `json:"sender_id,omitempty"`
	ScopeType       string   `json:"scope_type"`
	ScopeID         *string  `json:"scope_id,omitempty"`
	Enabled         bool     `json:"enabled"`
	QuietFrom       *string  `json:"quiet_from,omitempty"`
	QuietTo         *string  `json:"quiet_to,omitempty"`
	BlockedChannels []string `json:"blocked_channels"`
}

func preferenceToResponse(pref *domain.Preference) PreferenceResponse {
	return PreferenceResponse{
		PreferenceID:    pref.ID,
		ContactID:       pref.ContactID,
		SenderID:        pref.SenderID,
		ScopeType:       pref.ScopeType,
		ScopeID:         pref.ScopeID,
		Enabled:         pref.Enabled,
		QuietFrom:       pref.QuietFrom,
		QuietTo:         pref.QuietTo,
		BlockedChannels: pref.BlockedChannels,
	}
}

// List handles GET /api/recipients/preferences
func (h *PreferenceHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не авторизован"})
		return
	}

	preferences, err := h.prefService.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка сервера"})
		return
	}

	response := make([]PreferenceResponse, 0, len(preferences))
	for _, pref := range preferences {
		response = append(response, preferenceToResponse(&pref))
	}

	c.JSON(http.StatusOK, gin.H{
		"items": response,
		"page":  1,
		"size":  len(response),
		"total": len(response),
	})
}

// Update handles PUT /api/recipients/preferences
func (h *PreferenceHandler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не авторизован"})
		return
	}

	var req service.UpdatePreferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный запрос"})
		return
	}

	pref, err := h.prefService.Update(c.Request.Context(), userID, &req)
	if err != nil {
		if err == service.ErrContactNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, preferenceToResponse(pref))
}
