package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"notification-center/recipients/internal/service"
)

type InternalHandler struct {
	internalResolveService *service.InternalResolveService
}

func NewInternalHandler(internalResolveService *service.InternalResolveService) *InternalHandler {
	return &InternalHandler{internalResolveService: internalResolveService}
}

type InternalContactResponse struct {
	ContactID string `json:"contact_id"`
	Channel   string `json:"channel"`
	Value     string `json:"value"`
	Enabled   bool   `json:"enabled"`
}

// Resolve handles POST /internal/recipients/resolve
func (h *InternalHandler) Resolve(c *gin.Context) {
	var req service.ResolveContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный запрос"})
		return
	}

	resp, err := h.internalResolveService.Resolve(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка сервера"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetUserContacts handles GET /internal/recipients/users/:user_id/contacts
func (h *InternalHandler) GetUserContacts(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id обязателен"})
		return
	}

	contacts, err := h.internalResolveService.GetUserContacts(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка сервера"})
		return
	}

	response := make([]InternalContactResponse, 0, len(contacts))
	for _, contact := range contacts {
		response = append(response, InternalContactResponse{
			ContactID: contact.ID,
			Channel:   contact.Channel,
			Value:     contact.Value,
			Enabled:   contact.Enabled,
		})
	}

	c.JSON(http.StatusOK, gin.H{"contacts": response})
}
