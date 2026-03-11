package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"ozon-manager/internal/dto"
	"ozon-manager/internal/middleware"
	"ozon-manager/internal/model"
	"ozon-manager/internal/service"
)

type ExtensionHandler struct {
	automationService *service.AutomationService
	shopService       *service.ShopService
}

func NewExtensionHandler(automationService *service.AutomationService, shopService *service.ShopService) *ExtensionHandler {
	return &ExtensionHandler{
		automationService: automationService,
		shopService:       shopService,
	}
}

func (h *ExtensionHandler) Register(c *gin.Context) {
	var req dto.ExtensionRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{Code: 400, Message: "invalid request payload"})
		return
	}

	claims := middleware.GetCurrentUser(c)
	if err := h.shopService.CheckUserAccessByRole(claims.UserID, req.ShopID, claims.Role); err != nil {
		c.JSON(http.StatusForbidden, dto.Response{Code: 403, Message: "no access to this shop"})
		return
	}

	c.Set("shop_id", req.ShopID)

	data, err := h.automationService.ExtensionRegister(claims.UserID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{Code: 500, Message: "failed to register extension: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.Response{Code: 200, Message: "extension registered", Data: data})
}

func (h *ExtensionHandler) Poll(c *gin.Context) {
	var req dto.ExtensionPollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{Code: 400, Message: "invalid request payload"})
		return
	}

	claims := middleware.GetCurrentUser(c)
	if err := h.shopService.CheckUserAccessByRole(claims.UserID, req.ShopID, claims.Role); err != nil {
		c.JSON(http.StatusForbidden, dto.Response{Code: 403, Message: "no access to this shop"})
		return
	}

	job, err := h.automationService.ExtensionPoll(claims.UserID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{Code: 500, Message: "failed to poll job: " + err.Error()})
		return
	}
	if job == nil {
		c.JSON(http.StatusOK, dto.Response{
			Code:    200,
			Message: "no job",
			Data:    dto.ExtensionPollResponse{},
		})
		return
	}

	items := make([]dto.AutomationJobCreateItem, 0, len(job.Items))
	for _, item := range job.Items {
		items = append(items, dto.AutomationJobCreateItem{
			SourceSKU:   item.SourceSKU,
			TargetPrice: item.TargetPrice,
		})
	}

	meta := map[string]interface{}{}
	switch job.JobType {
	case model.AutomationJobTypeSyncActionCandidates:
		if artifact, err := h.automationService.GetLatestArtifact(job.ID, "sync_action_candidates_meta"); err == nil {
			_ = json.Unmarshal(artifact.Meta, &meta)
		}
	case model.AutomationJobTypeSyncActionProducts:
		if artifact, err := h.automationService.GetLatestArtifact(job.ID, "sync_action_products_meta"); err == nil {
			_ = json.Unmarshal(artifact.Meta, &meta)
		}
	case model.AutomationJobTypeRemoveRepriceReadd:
		if artifact, err := h.automationService.GetLatestArtifact(job.ID, "remove_reprice_readd_meta"); err == nil {
			_ = json.Unmarshal(artifact.Meta, &meta)
		}
	case model.AutomationJobTypeShopActionDeclare, model.AutomationJobTypeShopActionRemove:
		if artifact, err := h.automationService.GetLatestArtifact(job.ID, "shop_action_meta"); err == nil {
			_ = json.Unmarshal(artifact.Meta, &meta)
		}
	case model.AutomationJobTypePromoUnifiedEnroll, model.AutomationJobTypePromoUnifiedRemove:
		if artifact, err := h.automationService.GetLatestArtifact(job.ID, "promo_unified_meta"); err == nil {
			_ = json.Unmarshal(artifact.Meta, &meta)
		}
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "job assigned",
		Data: dto.ExtensionPollResponse{
			Job: &dto.AgentJobPayload{
				JobID:     job.ID,
				ShopID:    job.ShopID,
				JobType:   job.JobType,
				DryRun:    job.DryRun,
				RateLimit: job.RateLimit,
				Items:     items,
				Meta:      meta,
			},
		},
	})
}

func (h *ExtensionHandler) Report(c *gin.Context) {
	var req dto.ExtensionReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{Code: 400, Message: "invalid request payload"})
		return
	}

	claims := middleware.GetCurrentUser(c)
	if err := h.shopService.CheckUserAccessByRole(claims.UserID, req.ShopID, claims.Role); err != nil {
		c.JSON(http.StatusForbidden, dto.Response{Code: 403, Message: "no access to this shop"})
		return
	}

	c.Set("shop_id", req.ShopID)

	if err := h.automationService.ExtensionReport(claims.UserID, &req); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{Code: 500, Message: "failed to report job: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.Response{Code: 200, Message: "job reported"})
}

func (h *ExtensionHandler) Reprice(c *gin.Context) {
	var req dto.ExtensionRepriceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{Code: 400, Message: "invalid request payload"})
		return
	}

	claims := middleware.GetCurrentUser(c)
	if err := h.shopService.CheckUserAccessByRole(claims.UserID, req.ShopID, claims.Role); err != nil {
		c.JSON(http.StatusForbidden, dto.Response{Code: 403, Message: "no access to this shop"})
		return
	}

	c.Set("shop_id", req.ShopID)

	if err := h.automationService.ExtensionRepriceProduct(req.ShopID, req.SourceSKU, req.NewPrice); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{Code: 500, Message: "failed to reprice: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "repriced",
		Data: dto.ExtensionRepriceResponse{
			ShopID:    req.ShopID,
			SourceSKU: req.SourceSKU,
			NewPrice:  req.NewPrice,
		},
	})
}
