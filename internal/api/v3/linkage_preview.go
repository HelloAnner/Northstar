/**
 * 联动预览接口
 *
 * @author Anner
 * Created on 2026/2/3
 */

package v3

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"northstar/internal/linkage"
	"northstar/internal/store"
)

// PreviewLinkage 预览联动影响范围
// POST /api/linkage/preview
func (h *Handler) PreviewLinkage(c *gin.Context) {
	year, month, err := h.store.GetCurrentYearMonth()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "system not initialized"})
		return
	}

	var req linkage.AnchorPreviewRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	wrRecords, err := h.store.GetWRByYearMonth(store.WRQueryOptions{
		DataYear:  &year,
		DataMonth: &month,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	acRecords, err := h.store.GetACByYearMonth(store.ACQueryOptions{
		DataYear:  &year,
		DataMonth: &month,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	index, err := linkage.LoadTemplateIndex()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	graph, err := linkage.BuildGraph(linkage.BuildGraphOptions{
		WRRecords:     wrRecords,
		ACRecords:     acRecords,
		TemplateIndex: index,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	anchor, err := linkage.ResolveAnchorNode(graph, req.Anchor)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	nodes := linkage.ComputeImpact(graph, anchor)
	c.JSON(http.StatusOK, gin.H{"nodes": nodes})
}
