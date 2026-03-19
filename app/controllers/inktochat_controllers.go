package controllers

import (
	"DataTracker/app/services"
	"DataTracker/ratelimit"
	"net/http"
	"github.com/gin-gonic/gin"
)

type InkController struct {
	chatService      *services.ChatService
	authService      *services.InkAuthService
	analyticsService *services.AnalyticsService
	rateLimiter      *ratelimit.Manager
}

func NewInkController(cs *services.ChatService, as *services.InkAuthService, ans *services.AnalyticsService, rl *ratelimit.Manager) *InkController {
	return &InkController{chatService: cs, authService: as, analyticsService: ans, rateLimiter: rl}
}

func (ic *InkController) GetToken(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid format"})
		return
	}

	if !ic.rateLimiter.Allow(c.ClientIP()) {
		c.JSON(429, gin.H{"error": "Too Many Requests"})
		return
	}

	token, err := ic.authService.GetToken(c.Request.Context(), req.Username)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"token": token})
}

func (ic *InkController) PostMessage(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Room     string `json:"room" binding:"required"`
		Text     string `json:"text" binding:"required"`
		Color    string `json:"color"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid format"})
		return
	}

	if !ic.rateLimiter.Allow("msg_" + req.Username) {
		c.JSON(429, gin.H{"error": "Rate limit exceeded"})
		return
	}

	if err := ic.chatService.ProcessMessage(c.Request.Context(), req.Room, req.Username, req.Text, req.Color); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "success"})
}

func (ic *InkController) PostDrawing(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Room     string `json:"room" binding:"required"`
		Blob     []byte `json:"blob" binding:"required"`
		Color    string `json:"color"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid format"})
		return
	}

	if !ic.rateLimiter.Allow("draw_" + req.Username) {
		c.JSON(429, gin.H{"error": "Rate limit exceeded"})
		return
	}

	if err := ic.chatService.ProcessDrawing(c.Request.Context(), req.Room, req.Username, req.Color, req.Blob); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "success"})
}

func (ic *InkController) GetAnalytics(c *gin.Context) {
	stats, err := ic.analyticsService.GetAllStats(c.Request.Context())
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to fetch stats"})
		return
	}
	c.JSON(200, stats)
}

func (ic *InkController) ResetSystem(c *gin.Context) {
	// (destructive analytics reset from user logic)
	ic.authService.ResetIdentityFilter()
	c.JSON(200, gin.H{"status": "system reset complete"})
}

func (ic *InkController) ClearLobbies(c *gin.Context) {
	if err := ic.chatService.ClearLobbies(c.Request.Context()); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "all chat histories cleared"})
}

func (ic *InkController) SavePersonalDrawing(c *gin.Context) {
	userID, _ := c.Get("userID")
	var req struct {
		Blob []byte `json:"blob" binding:"required"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid format"})
		return
	}
	if err := ic.chatService.PersonalSaveDrawing(c.Request.Context(), userID.(string), req.Blob); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "success"})
}

func (ic *InkController) GetPersonalDrawings(c *gin.Context) {
	userID, _ := c.Get("userID")
	drawings, err := ic.chatService.GetUserDrawings(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, drawings)
}

func (ic *InkController) DeletePersonalDrawing(c *gin.Context) {
	userID, _ := c.Get("userID")
	drawingID := c.Param("id")
	if err := ic.chatService.PersonalDeleteDrawing(c.Request.Context(), userID.(string), drawingID); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "deleted"})
}

func (ic *InkController) GetSpamDrops(c *gin.Context) {
	drops := ic.rateLimiter.GetDrops()
	c.JSON(http.StatusOK, gin.H{"drops": drops})
}
