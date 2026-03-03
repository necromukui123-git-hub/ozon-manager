package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"ozon-manager/internal/dto"
	"ozon-manager/internal/middleware"
	"ozon-manager/internal/model"
	"ozon-manager/internal/service"
)

type AutomationHandler struct {
	automationService *service.AutomationService
	shopService       *service.ShopService
}

func NewAutomationHandler(automationService *service.AutomationService, shopService *service.ShopService) *AutomationHandler {
	return &AutomationHandler{
		automationService: automationService,
		shopService:       shopService,
	}
}

func (h *AutomationHandler) CreateJob(c *gin.Context) {
	var req dto.CreateAutomationJobRequest
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

	job, err := h.automationService.CreateJob(claims.UserID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{Code: 500, Message: "failed to create automation job: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "automation job created",
		Data:    buildAutomationJobDetail(job),
	})
}

func (h *AutomationHandler) GetJobs(c *gin.Context) {
	var req dto.AutomationJobListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{Code: 400, Message: "invalid query params"})
		return
	}

	claims := middleware.GetCurrentUser(c)
	if err := h.shopService.CheckUserAccessByRole(claims.UserID, req.ShopID, claims.Role); err != nil {
		c.JSON(http.StatusForbidden, dto.Response{Code: 403, Message: "no access to this shop"})
		return
	}

	jobs, total, err := h.automationService.ListJobs(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{Code: 500, Message: "failed to list automation jobs"})
		return
	}

	items := make([]dto.AutomationJobSummary, 0, len(jobs))
	for _, job := range jobs {
		var completedAt *string
		if job.CompletedAt != nil {
			formatted := job.CompletedAt.Format("2006-01-02 15:04:05")
			completedAt = &formatted
		}

		items = append(items, dto.AutomationJobSummary{
			ID:           job.ID,
			ShopID:       job.ShopID,
			CreatedBy:    job.CreatedBy,
			JobType:      job.JobType,
			Status:       job.Status,
			DryRun:       job.DryRun,
			TotalItems:   job.TotalItems,
			SuccessItems: job.SuccessItems,
			FailedItems:  job.FailedItems,
			CreatedAt:    job.CreatedAt.Format("2006-01-02 15:04:05"),
			CompletedAt:  completedAt,
		})
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "success",
		Data: dto.AutomationJobListResponse{
			Total: total,
			Items: items,
		},
	})
}

func (h *AutomationHandler) GetJobDetail(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{Code: 400, Message: "invalid job id"})
		return
	}

	shopID, err := strconv.ParseUint(c.Query("shop_id"), 10, 32)
	if err != nil || shopID == 0 {
		c.JSON(http.StatusBadRequest, dto.Response{Code: 400, Message: "missing shop_id"})
		return
	}

	claims := middleware.GetCurrentUser(c)
	if err := h.shopService.CheckUserAccessByRole(claims.UserID, uint(shopID), claims.Role); err != nil {
		c.JSON(http.StatusForbidden, dto.Response{Code: 403, Message: "no access to this shop"})
		return
	}

	job, err := h.automationService.GetJobDetail(uint(shopID), uint(jobID))
	if err != nil {
		c.JSON(http.StatusNotFound, dto.Response{Code: 404, Message: "automation job not found"})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "success",
		Data:    buildAutomationJobDetail(job),
	})
}

func (h *AutomationHandler) AgentHeartbeat(c *gin.Context) {
	var req dto.AgentHeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{Code: 400, Message: "invalid request payload"})
		return
	}

	agent, err := h.automationService.AgentHeartbeat(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{Code: 500, Message: "failed to update heartbeat"})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "heartbeat accepted",
		Data: map[string]interface{}{
			"agent_id":  agent.ID,
			"agent_key": agent.AgentKey,
			"status":    agent.Status,
		},
	})
}

func (h *AutomationHandler) AgentPoll(c *gin.Context) {
	var req dto.AgentPollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{Code: 400, Message: "invalid request payload"})
		return
	}

	job, err := h.automationService.AgentPoll(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{Code: 500, Message: "failed to poll job: " + err.Error()})
		return
	}

	if job == nil {
		c.JSON(http.StatusOK, dto.Response{
			Code:    200,
			Message: "no job",
			Data:    dto.AgentPollResponse{},
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
	case model.AutomationJobTypeSyncActionProducts:
		if artifact, err := h.automationService.GetLatestArtifact(job.ID, "sync_action_products_meta"); err == nil {
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
		Data: dto.AgentPollResponse{
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

func (h *AutomationHandler) AgentReport(c *gin.Context) {
	var req dto.AgentReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{Code: 400, Message: "invalid request payload"})
		return
	}

	if err := h.automationService.AgentReport(&req); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{Code: 500, Message: "failed to report job: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.Response{Code: 200, Message: "job reported"})
}

func (h *AutomationHandler) ConfirmJob(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{Code: 400, Message: "invalid job id"})
		return
	}

	var req dto.ConfirmAutomationJobRequest
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

	if err := h.automationService.ConfirmJob(claims.UserID, req.ShopID, uint(jobID)); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{Code: 400, Message: "failed to confirm job: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.Response{Code: 200, Message: "job confirmed"})
}

func (h *AutomationHandler) CancelJob(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{Code: 400, Message: "invalid job id"})
		return
	}

	var req dto.CancelAutomationJobRequest
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

	if err := h.automationService.CancelJob(claims.UserID, req.ShopID, uint(jobID)); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{Code: 400, Message: "failed to cancel job: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.Response{Code: 200, Message: "job canceled"})
}

func (h *AutomationHandler) RetryFailedItems(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{Code: 400, Message: "invalid job id"})
		return
	}

	var req dto.RetryFailedAutomationJobRequest
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

	if err := h.automationService.RetryFailedItems(claims.UserID, req.ShopID, uint(jobID)); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{Code: 400, Message: "failed to retry job: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.Response{Code: 200, Message: "retry requested"})
}

func (h *AutomationHandler) GetEvents(c *gin.Context) {
	var req dto.AutomationEventListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{Code: 400, Message: "invalid query params"})
		return
	}

	claims := middleware.GetCurrentUser(c)
	if err := h.shopService.CheckUserAccessByRole(claims.UserID, req.ShopID, claims.Role); err != nil {
		c.JSON(http.StatusForbidden, dto.Response{Code: 403, Message: "no access to this shop"})
		return
	}

	events, total, err := h.automationService.ListEvents(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{Code: 500, Message: "failed to list events"})
		return
	}

	items := make([]dto.AutomationEventItem, 0, len(events))
	for _, event := range events {
		items = append(items, dto.AutomationEventItem{
			ID:        event.ID,
			JobID:     event.JobID,
			EventType: event.EventType,
			Message:   event.Message,
			CreatedBy: event.CreatedBy,
			CreatedAt: event.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "success",
		Data: dto.AutomationEventListResponse{
			Total: total,
			Items: items,
		},
	})
}

func (h *AutomationHandler) GetAgentStatus(c *gin.Context) {
	agents, err := h.automationService.ListAgents()
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{Code: 500, Message: "failed to list agents"})
		return
	}

	items := make([]dto.AgentStatusItem, 0, len(agents))
	for _, agent := range agents {
		lastHeartbeatAt := service.FormatAutomationTime(agent.LastHeartbeatAt)
		items = append(items, dto.AgentStatusItem{
			ID:              agent.ID,
			AgentKey:        agent.AgentKey,
			Name:            agent.Name,
			Hostname:        agent.Hostname,
			Status:          agent.Status,
			LastHeartbeatAt: lastHeartbeatAt,
			UpdatedAt:       agent.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, dto.Response{Code: 200, Message: "success", Data: items})
}

func (h *AutomationHandler) GetExtensionStatus(c *gin.Context) {
	items, err := h.automationService.GetExtensionStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{Code: 500, Message: "failed to list extension status"})
		return
	}
	c.JSON(http.StatusOK, dto.Response{Code: 200, Message: "success", Data: items})
}

func buildAutomationJobDetail(job *model.AutomationJob) dto.AutomationJobDetailResponse {
	startedAt := service.FormatAutomationTime(job.StartedAt)
	completedAt := service.FormatAutomationTime(job.CompletedAt)

	return dto.AutomationJobDetailResponse{
		ID:                   job.ID,
		ShopID:               job.ShopID,
		CreatedBy:            job.CreatedBy,
		JobType:              job.JobType,
		Status:               job.Status,
		DryRun:               job.DryRun,
		RequiresConfirmation: job.RequiresConfirmation,
		RateLimit:            job.RateLimit,
		TotalItems:           job.TotalItems,
		SuccessItems:         job.SuccessItems,
		FailedItems:          job.FailedItems,
		ErrorMessage:         job.ErrorMessage,
		CreatedAt:            job.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:            job.UpdatedAt.Format("2006-01-02 15:04:05"),
		StartedAt:            startedAt,
		CompletedAt:          completedAt,
		Items:                mapAutomationJobItems(job.Items),
	}
}

func mapAutomationJobItems(items []model.AutomationJobItem) []dto.AutomationJobItemDetail {
	result := make([]dto.AutomationJobItemDetail, 0, len(items))
	for _, item := range items {
		result = append(result, dto.AutomationJobItemDetail{
			ID:                item.ID,
			ProductID:         item.ProductID,
			SourceSKU:         item.SourceSKU,
			TargetPrice:       item.TargetPrice,
			OverallStatus:     item.OverallStatus,
			StepExitStatus:    item.StepExitStatus,
			StepRepriceStatus: item.StepRepriceStatus,
			StepReaddStatus:   item.StepReaddStatus,
			StepExitError:     item.StepExitError,
			StepRepriceError:  item.StepRepriceError,
			StepReaddError:    item.StepReaddError,
		})
	}
	return result
}
