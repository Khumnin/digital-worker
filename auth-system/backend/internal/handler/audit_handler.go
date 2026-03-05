// internal/handler/audit_handler.go
package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"tigersoft/auth-system/internal/domain"
	"tigersoft/auth-system/internal/service"
)

// normalizeIP strips the IPv6-mapped IPv4 prefix (::ffff:) so that
// addresses like "::ffff:192.168.1.1" are returned as "192.168.1.1".
func normalizeIP(ip string) string {
	const prefix = "::ffff:"
	if strings.HasPrefix(ip, prefix) {
		return ip[len(prefix):]
	}
	return ip
}

// AuditHandler handles audit log query endpoints.
type AuditHandler struct {
	auditSvc service.AuditService
}

// NewAuditHandler constructs an AuditHandler with its required service dependency.
func NewAuditHandler(svc service.AuditService) *AuditHandler {
	return &AuditHandler{auditSvc: svc}
}

// List handles GET /api/v1/admin/audit-log.
// Supports pagination via page and page_size query parameters.
// Supports optional filtering via action (event_type), actor_id, target_id, from, to.
// Date range params accept ISO date strings (YYYY-MM-DD) or RFC3339 timestamps.
// Maximum page size is capped at 500 to protect against runaway queries.
func (h *AuditHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 500 {
		pageSize = 50
	}

	offset := (page - 1) * pageSize

	filter := domain.AuditFilter{
		Limit:  pageSize,
		Offset: offset,
	}

	// Accept "action" as the canonical name; fall back to legacy "event_type" for compatibility.
	if et := c.Query("action"); et != "" {
		filter.EventType = &et
	} else if et := c.Query("event_type"); et != "" {
		filter.EventType = &et
	}

	if aid := c.Query("actor_id"); aid != "" {
		id, err := uuid.Parse(aid)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_PARAM", "message": "actor_id must be a valid UUID"}})
			return
		}
		filter.ActorID = &id
	}

	// Accept "target_id" as the canonical name; fall back to legacy "target_user_id".
	if tid := c.Query("target_id"); tid != "" {
		id, err := uuid.Parse(tid)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_PARAM", "message": "target_id must be a valid UUID"}})
			return
		}
		filter.TargetUserID = &id
	} else if tid := c.Query("target_user_id"); tid != "" {
		id, err := uuid.Parse(tid)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_PARAM", "message": "target_user_id must be a valid UUID"}})
			return
		}
		filter.TargetUserID = &id
	}

	// Parse date range — try RFC3339 first (carries timezone offset), fall back
	// to plain ISO date (YYYY-MM-DD) which is interpreted as UTC midnight.
	// Trying RFC3339 first ensures that a frontend sending e.g.
	// "2025-03-05T00:00:00+07:00" is honoured correctly and not silently
	// truncated to UTC, which would exclude early-morning local-time events.
	if from := c.Query("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			filter.From = &t
		} else if t, err := time.Parse("2006-01-02", from); err == nil {
			filter.From = &t
		}
	}

	if to := c.Query("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			filter.To = &t
		} else if t, err := time.Parse("2006-01-02", to); err == nil {
			// End of the given date (inclusive): advance to start of next day.
			endOfDay := t.Add(24*time.Hour - time.Nanosecond)
			filter.To = &endOfDay
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
			"id":         e.ID.String(),
			"action":     string(e.EventType),
			"ip_address": normalizeIP(e.ActorIP),
			"metadata":   e.Metadata,
			"created_at": e.OccurredAt,
		}

		// Optional FK fields — only include them when present to keep the
		// payload clean and avoid surfacing nil UUID values to callers.
		if e.ActorID != nil {
			item["actor_id"] = e.ActorID.String()
		}
		if e.ActorEmail != nil {
			item["actor_email"] = *e.ActorEmail
		}
		if e.TargetUserID != nil {
			item["target_id"] = e.TargetUserID.String()
		}
		if e.TargetEmail != nil {
			item["target_email"] = *e.TargetEmail
		}

		items[i] = item
	}

	totalPages := total / pageSize
	if total%pageSize != 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        items,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
	})
}
