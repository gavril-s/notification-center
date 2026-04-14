package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"notification-center/sources/internal/model"
	"notification-center/sources/internal/service"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	// Public API routes
	api := r.Group("/api/sources")
	{
		// Senders
		api.GET("/senders", h.ListSenders)
		api.POST("/senders", h.CreateSender)
		api.POST("/senders/:sender_id/operators", h.AddOperator)
		api.POST("/senders/:sender_id/credentials", h.AddCredential)

		// Templates
		api.GET("/templates", h.ListTemplates)
		api.POST("/templates", h.CreateTemplate)
		api.PUT("/templates/:template_id", h.UpdateTemplate)
		api.DELETE("/templates/:template_id", h.DeleteTemplate)

		// Groups
		api.GET("/groups", h.ListGroups)
		api.POST("/groups", h.CreateGroup)
		api.PUT("/groups/:group_id", h.UpdateGroup)
		api.POST("/groups/:group_id/members", h.AddGroupMembers)

		// Campaigns
		api.GET("/campaigns", h.ListCampaigns)
		api.POST("/campaigns", h.CreateCampaign)
		api.PUT("/campaigns/:campaign_id", h.UpdateCampaign)
	}

	// Internal API routes
	internal := r.Group("/internal/sources")
	{
		internal.GET("/templates/:template_id", h.GetTemplate)
		internal.GET("/campaigns/:campaign_id", h.GetCampaign)
		internal.GET("/campaigns/due", h.GetDueCampaigns)
		internal.GET("/groups/:group_id/members", h.GetGroupMembers)
		internal.GET("/senders/:sender_id/operators/:user_id", h.CheckOperatorPermission)
		internal.POST("/senders/resolve-credential", h.ResolveCredential)
	}

	// Health and ready checks
	r.GET("/health", h.Health)
	r.GET("/ready", h.Ready)
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "component": "sources"})
}

func (h *Handler) Ready(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ready", "component": "sources"})
}

// Sender handlers

func (h *Handler) ListSenders(c *gin.Context) {
	senders, err := h.svc.ListSenders(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка сервера"})
		return
	}
	c.JSON(http.StatusOK, model.ListResponse{Items: senders, Total: len(senders)})
}

func (h *Handler) CreateSender(c *gin.Context) {
	var req model.CreateSenderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректные входные данные"})
		return
	}

	sender, err := h.svc.CreateSender(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка сервера"})
		return
	}
	c.JSON(http.StatusCreated, sender)
}

func (h *Handler) AddOperator(c *gin.Context) {
	senderID, err := uuid.Parse(c.Param("sender_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный идентификатор отправителя"})
		return
	}

	var req model.AddOperatorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректные входные данные"})
		return
	}

	op, err := h.svc.AddOperator(c.Request.Context(), senderID, req)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "отправитель не найден"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка сервера"})
		return
	}
	c.JSON(http.StatusCreated, op)
}

func (h *Handler) AddCredential(c *gin.Context) {
	senderID, err := uuid.Parse(c.Param("sender_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный идентификатор отправителя"})
		return
	}

	var req model.AddCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректные входные данные"})
		return
	}

	cred, err := h.svc.AddCredential(c.Request.Context(), senderID, req)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "отправитель не найден"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка сервера"})
		return
	}
	c.JSON(http.StatusCreated, cred)
}

// Template handlers

func (h *Handler) ListTemplates(c *gin.Context) {
	var senderID *uuid.UUID
	if sid := c.Query("sender_id"); sid != "" {
		id, err := uuid.Parse(sid)
		if err == nil {
			senderID = &id
		}
	}

	templates, err := h.svc.ListTemplates(c.Request.Context(), senderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка сервера"})
		return
	}
	c.JSON(http.StatusOK, model.ListResponse{Items: templates, Total: len(templates)})
}

func (h *Handler) GetTemplate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("template_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный идентификатор шаблона"})
		return
	}

	tmpl, err := h.svc.GetTemplate(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка сервера"})
		return
	}
	if tmpl == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "шаблон не найден"})
		return
	}
	c.JSON(http.StatusOK, tmpl)
}

func (h *Handler) CreateTemplate(c *gin.Context) {
	var req model.CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректные входные данные"})
		return
	}

	tmpl, err := h.svc.CreateTemplate(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "отправитель не найден"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка сервера"})
		return
	}
	c.JSON(http.StatusCreated, tmpl)
}

func (h *Handler) UpdateTemplate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("template_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный идентификатор шаблона"})
		return
	}

	var req model.UpdateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректные входные данные"})
		return
	}

	tmpl, err := h.svc.UpdateTemplate(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "шаблон не найден"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка сервера"})
		return
	}
	c.JSON(http.StatusOK, tmpl)
}

func (h *Handler) DeleteTemplate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("template_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный идентификатор шаблона"})
		return
	}

	if err := h.svc.DeleteTemplate(c.Request.Context(), id); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "шаблон не найден"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка сервера"})
		return
	}
	c.Status(http.StatusNoContent)
}

// Group handlers

func (h *Handler) ListGroups(c *gin.Context) {
	var senderID *uuid.UUID
	if sid := c.Query("sender_id"); sid != "" {
		id, err := uuid.Parse(sid)
		if err == nil {
			senderID = &id
		}
	}

	groups, err := h.svc.ListGroups(c.Request.Context(), senderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка сервера"})
		return
	}
	c.JSON(http.StatusOK, model.ListResponse{Items: groups, Total: len(groups)})
}

func (h *Handler) CreateGroup(c *gin.Context) {
	var req model.CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректные входные данные"})
		return
	}

	group, err := h.svc.CreateGroup(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "отправитель не найден"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка сервера"})
		return
	}
	c.JSON(http.StatusCreated, group)
}

func (h *Handler) UpdateGroup(c *gin.Context) {
	id, err := uuid.Parse(c.Param("group_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный идентификатор группы"})
		return
	}

	var req model.UpdateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректные входные данные"})
		return
	}

	group, err := h.svc.UpdateGroup(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "группа не найдена"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка сервера"})
		return
	}
	c.JSON(http.StatusOK, group)
}

func (h *Handler) AddGroupMembers(c *gin.Context) {
	id, err := uuid.Parse(c.Param("group_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный идентификатор группы"})
		return
	}

	var req model.AddGroupMembersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректные входные данные"})
		return
	}

	if err := h.svc.AddGroupMembers(c.Request.Context(), id, req); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "группа не найдена"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка сервера"})
		return
	}
	c.Status(http.StatusCreated)
}

func (h *Handler) GetGroupMembers(c *gin.Context) {
	id, err := uuid.Parse(c.Param("group_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный идентификатор группы"})
		return
	}

	members, err := h.svc.GetGroupMembers(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "группа не найдена"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка сервера"})
		return
	}
	c.JSON(http.StatusOK, members)
}

// Campaign handlers

func (h *Handler) ListCampaigns(c *gin.Context) {
	var senderID *uuid.UUID
	if sid := c.Query("sender_id"); sid != "" {
		id, err := uuid.Parse(sid)
		if err == nil {
			senderID = &id
		}
	}

	campaigns, err := h.svc.ListCampaigns(c.Request.Context(), senderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка сервера"})
		return
	}
	c.JSON(http.StatusOK, model.ListResponse{Items: campaigns, Total: len(campaigns)})
}

func (h *Handler) GetCampaign(c *gin.Context) {
	id, err := uuid.Parse(c.Param("campaign_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный идентификатор кампании"})
		return
	}

	campaign, err := h.svc.GetCampaign(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка сервера"})
		return
	}
	if campaign == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "кампания не найдена"})
		return
	}
	c.JSON(http.StatusOK, campaign)
}

func (h *Handler) CreateCampaign(c *gin.Context) {
	var req model.CreateCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректные входные данные"})
		return
	}

	campaign, err := h.svc.CreateCampaign(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "ресурс не найден"})
			return
		}
		if errors.Is(err, service.ErrChannelMismatch) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка сервера"})
		return
	}
	c.JSON(http.StatusCreated, campaign)
}

func (h *Handler) UpdateCampaign(c *gin.Context) {
	id, err := uuid.Parse(c.Param("campaign_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный идентификатор кампании"})
		return
	}

	var req model.UpdateCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректные входные данные"})
		return
	}

	campaign, err := h.svc.UpdateCampaign(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "кампания не найдена"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка сервера"})
		return
	}
	c.JSON(http.StatusOK, campaign)
}

func (h *Handler) GetDueCampaigns(c *gin.Context) {
	asOf := c.Query("as_of")
	if asOf == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "параметр as_of обязателен"})
		return
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := parseInt(l); err == nil {
			limit = parsed
		}
	}

	var cursor *string
	if c := c.Query("cursor"); c != "" {
		cursor = &c
	}

	campaigns, nextCursor, err := h.svc.GetDueCampaigns(c.Request.Context(), asOf, limit, cursor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка сервера"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"campaigns":   campaigns,
		"next_cursor": nextCursor,
	})
}

// Internal API handlers

func (h *Handler) CheckOperatorPermission(c *gin.Context) {
	senderID, err := uuid.Parse(c.Param("sender_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный идентификатор отправителя"})
		return
	}

	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный идентификатор пользователя"})
		return
	}

	perm, err := h.svc.CheckOperatorPermission(c.Request.Context(), senderID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка сервера"})
		return
	}
	c.JSON(http.StatusOK, perm)
}

func (h *Handler) ResolveCredential(c *gin.Context) {
	var req model.ResolveCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректные входные данные"})
		return
	}

	res, err := h.svc.ResolveCredential(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка сервера"})
		return
	}
	if res == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "учетные данные не найдены или неактивны"})
		return
	}
	c.JSON(http.StatusOK, res)
}

func parseInt(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, errors.New("invalid number")
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}
