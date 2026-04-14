package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"notification-center/recipients/internal/service"
)

type UnsubscribeHandler struct {
	unsubscribeService *service.UnsubscribeService
}

func NewUnsubscribeHandler(unsubscribeService *service.UnsubscribeService) *UnsubscribeHandler {
	return &UnsubscribeHandler{unsubscribeService: unsubscribeService}
}

// Unsubscribe handles POST /api/recipients/unsubscribe
func (h *UnsubscribeHandler) Unsubscribe(c *gin.Context) {
	var req service.UnsubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный запрос"})
		return
	}

	err := h.unsubscribeService.Unsubscribe(c.Request.Context(), &req)
	if err != nil {
		if err == service.ErrUnsubscribeTokenInvalid {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "успешно отписался"})
}
