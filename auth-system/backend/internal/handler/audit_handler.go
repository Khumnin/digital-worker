// internal/handler/audit_handler.go
package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"tigersoft/auth-system/internal/domain"
	"tigersoft/auth-system/internal/service"
)

// AuditHandler handles audit log query endpoints.
type AuditHandler struct {
	auditSvc service.AuditService
}

// NewAuditHandler constructs an AuditHandler with its required service dependency.
func NewAuditHandler(svc service.AuditService) *AuditHandler {
	return &AuditHandler{auditSvc: svc}
}

// List handles GET /api/v1/admin/audit-log.
// Supports pagination via limit and offset query parameters.
// Supports optional filtering via event_type, actor_id, target_user_id, from, to.
// Maximum page size is capped at 500 to protect against runaway queries.
func (h *AuditHandler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit <= 0 || limit > 500 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	filter := domain.AuditFilter{
		Limit:  limit,
		Offset: offset,
	}

	if et := c.Query("event_type"); et != "" {
		filter.EventType = &et
	}

	if aid := c.Query("actor_id"); aid != "" {
		if id, err := uuid.Parse(aid); err == nil {
			filter.ActorID = &id
		}
	}

	if tid := c.Query("target_user_id"); tid != "" {
		if id, err := uuid.Parse(tid); err == nil {
			filter.TargetUserID = &id
		}
	}

	if from := c.Query("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			filter.From = &t
		}
	}

	if to := c.Query("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			filter.To = &t
		}
	}

	events, total, err := h.auditSvc.List(c.Request.Context(), filter)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	items := make([]gin.H, len(events))
	for i, e := range events {
		item := gin.H{
			"id":          e.ID.String(),
			"event_type":  string(e.EventType),
			"actor_ip":    e.ActorIP,
			"metadata":    e.Metadata,
			"occurred_at": e.OccurredAt,
		}

		// Optional FK fields — only include them when present to keep the
		// payload clean and avoid surfacing nil UUID values to callers.
		if e.ActorID != nil {
			item["actor_id"] = e.ActorID.String()
		}
		if e.TargetUserID != nil {
			item["target_user_id"] = e.TargetUserID.String()
		}

		items[i] = item
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}
