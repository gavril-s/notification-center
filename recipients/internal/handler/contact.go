package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"notification-center/recipients/internal/domain"
	"notification-center/recipients/internal/middleware"
	"notification-center/recipients/internal/service"
)

type ContactHandler struct {
	contactService *service.ContactService
}

func NewContactHandler(contactService *service.ContactService) *ContactHandler {
	return &ContactHandler{contactService: contactService}
}

type ContactResponse struct {
	ContactID  string `json:"contact_id"`
	Channel    string `json:"channel"`
	Value      string `json:"value"`
	IsVerified bool   `json:"is_verified"`
	Enabled    bool   `json:"enabled"`
}

func contactToResponse(contact *domain.Contact) ContactResponse {
	return ContactResponse{
		ContactID:  contact.ID,
		Channel:    contact.Channel,
		Value:      contact.Value,
		IsVerified: contact.IsVerified,
		Enabled:    contact.Enabled,
	}
}

// List handles GET /api/recipients/contacts
func (h *ContactHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не авторизован"})
		return
	}

	contacts, err := h.contactService.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка сервера"})
		return
	}

	response := make([]ContactResponse, 0, len(contacts))
	for _, contact := range contacts {
		response = append(response, contactToResponse(&contact))
	}

	c.JSON(http.StatusOK, gin.H{"contacts": response})
}

// Create handles POST /api/recipients/contacts
func (h *ContactHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не авторизован"})
		return
	}

	var req service.CreateContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный запрос"})
		return
	}

	req.UserID = userID

	contact, err := h.contactService.Create(c.Request.Context(), userID, &req)
	if err != nil {
		if err == service.ErrContactExists {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, contactToResponse(contact))
}

// Update handles PUT /api/recipients/contacts/:contact_id
func (h *ContactHandler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не авторизован"})
		return
	}

	contactID := c.Param("contact_id")
	if contactID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "contact_id обязателен"})
		return
	}

	var req service.UpdateContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный запрос"})
		return
	}

	contact, err := h.contactService.Update(c.Request.Context(), contactID, userID, &req)
	if err != nil {
		if err == service.ErrContactNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, contactToResponse(contact))
}

// Delete handles DELETE /api/recipients/contacts/:contact_id
func (h *ContactHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не авторизован"})
		return
	}

	contactID := c.Param("contact_id")
	if contactID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "contact_id обязателен"})
		return
	}

	err := h.contactService.Delete(c.Request.Context(), contactID, userID)
	if err != nil {
		if err == service.ErrContactNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
