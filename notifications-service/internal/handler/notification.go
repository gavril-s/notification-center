package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"notification-center/notifications-service/internal/domain"
	"notification-center/notifications-service/internal/service"
)

type NotificationHandler struct {
	service *service.NotificationService
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Details   any    `json:"details,omitempty"`
	RequestID string `json:"request_id"`
}

type SendRequest struct {
	TemplateID     string         `json:"template_id" binding:"required"`
	CampaignID     *string        `json:"campaign_id"`
	ContactIDs     []string       `json:"contact_ids" binding:"required,min=1"`
	Channels       []string       `json:"channels" binding:"required,min=1,max=1"`
	Variables      map[string]any `json:"variables"`
	ScheduledAt    *time.Time     `json:"scheduled_at"`
	IdempotencyKey string         `json:"idempotency_key" binding:"required"`
}

func NewNotificationHandler(svc *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{service: svc}
}

func (h *NotificationHandler) Send(c *gin.Context) {
	var req SendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:      "invalid_request",
				Message:   "некорректный запрос",
				Details:   err.Error(),
				RequestID: c.GetString("request_id"),
			},
		})
		return
	}

	// Get integration key from header
	integrationKey := c.GetHeader("X-Integration-Key")
	if integrationKey == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: ErrorDetail{
				Code:      "unauthorized",
				Message:   "отсутствует заголовок X-Integration-Key",
				RequestID: c.GetString("request_id"),
			},
		})
		return
	}

	traceID := c.GetHeader("X-Trace-ID")
	if traceID == "" {
		traceID = c.GetString("request_id")
	}

	channels := make([]domain.Channel, len(req.Channels))
	for i, ch := range req.Channels {
		channels[i] = domain.Channel(ch)
	}

	serviceReq := service.SendNotificationRequest{
		TemplateID:     req.TemplateID,
		CampaignID:     req.CampaignID,
		ContactIDs:     req.ContactIDs,
		Channels:       channels,
		Variables:      req.Variables,
		ScheduledAt:    req.ScheduledAt,
		IdempotencyKey: req.IdempotencyKey,
	}

	notifications, err := h.service.Send(c.Request.Context(), serviceReq, integrationKey, traceID)
	if err != nil {
		code := "internal_error"
		message := "внутренняя ошибка сервера"

		switch err {
		case service.ErrSenderNotFound:
			code = "sender_not_found"
			message = "отправитель не найден"
		case service.ErrTemplateNotFound:
			code = "template_not_found"
			message = "шаблон не найден"
		case service.ErrChannelMismatch:
			code = "channel_mismatch"
			message = "канал не соответствует шаблону"
		case service.ErrMultipleChannels:
			code = "multiple_channels"
			message = "разрешен только один канал"
		case service.ErrNoContactsProvided:
			code = "no_contacts"
			message = "не указаны получатели"
		case service.ErrInvalidIdempotency:
			code = "invalid_idempotency_key"
			message = "некорректный ключ идемпотентности"
		}

		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:      code,
				Message:   message,
				RequestID: traceID,
			},
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"notifications": notifications,
		"request_id":    traceID,
	})
}

func (h *NotificationHandler) GetByID(c *gin.Context) {
	notificationID := c.Param("notification_id")

	notification, err := h.service.GetByID(c.Request.Context(), notificationID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error: ErrorDetail{
				Code:      "not_found",
				Message:   "уведомление не найдено",
				RequestID: c.GetString("request_id"),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"notification": notification,
		"request_id":   c.GetString("request_id"),
	})
}

func (h *NotificationHandler) GetHistory(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	mode := c.DefaultQuery("mode", "recipient")

	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	var items any
	var total int
	var err error

	if mode == "operator" {
		// Operator mode - requires sender_id and permission check
		senderID := c.Query("sender_id")
		if senderID == "" {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error: ErrorDetail{
					Code:      "invalid_request",
					Message:   "sender_id обязателен для режима оператора",
					RequestID: c.GetString("request_id"),
				},
			})
			return
		}

		// In production, verify operator permission here
		campaignID := c.Query("campaign_id")
		status := c.Query("status")

		items, total, err = h.service.GetHistoryBySenderID(c.Request.Context(), senderID, &campaignID, &status, page, size)
	} else {
		// Recipient mode - get user from JWT and resolve contacts
		// For now, we'll need to extract user ID from auth context
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: ErrorDetail{
					Code:      "unauthorized",
					Message:   "требуется авторизация",
					RequestID: c.GetString("request_id"),
				},
			})
			return
		}

		items, total, err = h.service.GetHistoryByUserID(c.Request.Context(), userID, page, size)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorDetail{
				Code:      "internal_error",
				Message:   "внутренняя ошибка сервера",
				RequestID: c.GetString("request_id"),
			},
		})
		return
	}

	if items == nil {
		items = []any{}
	}

	c.JSON(http.StatusOK, gin.H{
		"items":      items,
		"page":       page,
		"size":       size,
		"total":      total,
		"request_id": c.GetString("request_id"),
	})
}

func (h *NotificationHandler) GetAnalytics(c *gin.Context) {
	senderID := c.Query("sender_id")
	if senderID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:      "invalid_request",
				Message:   "sender_id обязателен",
				RequestID: c.GetString("request_id"),
			},
		})
		return
	}

	fromStr := c.DefaultQuery("from", time.Now().AddDate(0, -1, 0).Format("2006-01-02"))
	toStr := c.DefaultQuery("to", time.Now().Format("2006-01-02"))

	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:      "invalid_request",
				Message:   "некорректный формат даты from",
				RequestID: c.GetString("request_id"),
			},
		})
		return
	}

	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:      "invalid_request",
				Message:   "некорректный формат даты to",
				RequestID: c.GetString("request_id"),
			},
		})
		return
	}

	analytics, err := h.service.GetAnalytics(c.Request.Context(), senderID, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorDetail{
				Code:      "internal_error",
				Message:   "внутренняя ошибка сервера",
				RequestID: c.GetString("request_id"),
			},
		})
		return
	}

	if analytics == nil {
		analytics = []*domain.SenderAnalyticsDaily{}
	}

	c.JSON(http.StatusOK, gin.H{
		"analytics":  analytics,
		"request_id": c.GetString("request_id"),
	})
}

func (h *NotificationHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/notifications/send", h.Send)
	r.GET("/notifications/:notification_id", h.GetByID)
	r.GET("/notifications/history", h.GetHistory)
	r.GET("/notifications/analytics", h.GetAnalytics)
}
