package handler

import (
	"errors"
	"math"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"

	"github.com/zhufuyi/sponge/pkg/gin/middleware"
	"github.com/zhufuyi/sponge/pkg/gin/response"
	"github.com/zhufuyi/sponge/pkg/logger"
	"github.com/zhufuyi/sponge/pkg/utils"

	"gogogo/service_user/internal/cache"
	"gogogo/service_user/internal/dao"
	"gogogo/service_user/internal/ecode"
	"gogogo/service_user/internal/model"
	"gogogo/service_user/internal/types"
)

var _ UserJobHandler = (*userJobHandler)(nil)

// UserJobHandler defining the handler interface
type UserJobHandler interface {
	Create(c *gin.Context)
	DeleteByID(c *gin.Context)
	UpdateByID(c *gin.Context)
	GetByID(c *gin.Context)
	List(c *gin.Context)

	DeleteByIDs(c *gin.Context)
	GetByCondition(c *gin.Context)
	ListByIDs(c *gin.Context)
	ListByLastID(c *gin.Context)
}

type userJobHandler struct {
	iDao dao.UserJobDao
}

// NewUserJobHandler creating the handler interface
func NewUserJobHandler() UserJobHandler {
	return &userJobHandler{
		iDao: dao.NewUserJobDao(
			model.GetDB(),
			cache.NewUserJobCache(model.GetCacheType()),
		),
	}
}

// Create a record
// @Summary create userJob
// @Description submit information to create userJob
// @Tags userJob
// @accept json
// @Produce json
// @Param data body types.CreateUserJobRequest true "userJob information"
// @Success 200 {object} types.CreateUserJobReply{}
// @Router /api/v1/userJob [post]
// @Security BearerAuth
func (h *userJobHandler) Create(c *gin.Context) {
	form := &types.CreateUserJobRequest{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		logger.Warn("ShouldBindJSON error: ", logger.Err(err), middleware.GCtxRequestIDField(c))
		response.Error(c, ecode.InvalidParams)
		return
	}

	userJob := &model.UserJob{}
	err = copier.Copy(userJob, form)
	if err != nil {
		response.Error(c, ecode.ErrCreateUserJob)
		return
	}
	// Note: if copier.Copy cannot assign a value to a field, add it here

	ctx := middleware.WrapCtx(c)
	err = h.iDao.Create(ctx, userJob)
	if err != nil {
		logger.Error("Create error", logger.Err(err), logger.Any("form", form), middleware.GCtxRequestIDField(c))
		response.Output(c, ecode.InternalServerError.ToHTTPCode())
		return
	}

	response.Success(c, gin.H{"id": userJob.ID})
}

// DeleteByID delete a record by id
// @Summary delete userJob
// @Description delete userJob by id
// @Tags userJob
// @accept json
// @Produce json
// @Param id path string true "id"
// @Success 200 {object} types.DeleteUserJobByIDReply{}
// @Router /api/v1/userJob/{id} [delete]
// @Security BearerAuth
func (h *userJobHandler) DeleteByID(c *gin.Context) {
	_, id, isAbort := getUserJobIDFromPath(c)
	if isAbort {
		response.Error(c, ecode.InvalidParams)
		return
	}

	ctx := middleware.WrapCtx(c)
	err := h.iDao.DeleteByID(ctx, id)
	if err != nil {
		logger.Error("DeleteByID error", logger.Err(err), logger.Any("id", id), middleware.GCtxRequestIDField(c))
		response.Output(c, ecode.InternalServerError.ToHTTPCode())
		return
	}

	response.Success(c)
}

// UpdateByID update information by id
// @Summary update userJob
// @Description update userJob information by id
// @Tags userJob
// @accept json
// @Produce json
// @Param id path string true "id"
// @Param data body types.UpdateUserJobByIDRequest true "userJob information"
// @Success 200 {object} types.UpdateUserJobByIDReply{}
// @Router /api/v1/userJob/{id} [put]
// @Security BearerAuth
func (h *userJobHandler) UpdateByID(c *gin.Context) {
	_, id, isAbort := getUserJobIDFromPath(c)
	if isAbort {
		response.Error(c, ecode.InvalidParams)
		return
	}

	form := &types.UpdateUserJobByIDRequest{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		logger.Warn("ShouldBindJSON error: ", logger.Err(err), middleware.GCtxRequestIDField(c))
		response.Error(c, ecode.InvalidParams)
		return
	}
	form.ID = id

	userJob := &model.UserJob{}
	err = copier.Copy(userJob, form)
	if err != nil {
		response.Error(c, ecode.ErrUpdateByIDUserJob)
		return
	}
	// Note: if copier.Copy cannot assign a value to a field, add it here

	ctx := middleware.WrapCtx(c)
	err = h.iDao.UpdateByID(ctx, userJob)
	if err != nil {
		logger.Error("UpdateByID error", logger.Err(err), logger.Any("form", form), middleware.GCtxRequestIDField(c))
		response.Output(c, ecode.InternalServerError.ToHTTPCode())
		return
	}

	response.Success(c)
}

// GetByID get a record by id
// @Summary get userJob detail
// @Description get userJob detail by id
// @Tags userJob
// @Param id path string true "id"
// @Accept json
// @Produce json
// @Success 200 {object} types.GetUserJobByIDReply{}
// @Router /api/v1/userJob/{id} [get]
// @Security BearerAuth
func (h *userJobHandler) GetByID(c *gin.Context) {
	_, id, isAbort := getUserJobIDFromPath(c)
	if isAbort {
		response.Error(c, ecode.InvalidParams)
		return
	}

	ctx := middleware.WrapCtx(c)
	userJob, err := h.iDao.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, model.ErrRecordNotFound) {
			logger.Warn("GetByID not found", logger.Err(err), logger.Any("id", id), middleware.GCtxRequestIDField(c))
			response.Error(c, ecode.NotFound)
		} else {
			logger.Error("GetByID error", logger.Err(err), logger.Any("id", id), middleware.GCtxRequestIDField(c))
			response.Output(c, ecode.InternalServerError.ToHTTPCode())
		}
		return
	}

	data := &types.UserJobObjDetail{}
	err = copier.Copy(data, userJob)
	if err != nil {
		response.Error(c, ecode.ErrGetByIDUserJob)
		return
	}
	// Note: if copier.Copy cannot assign a value to a field, add it here

	response.Success(c, gin.H{"userJob": data})
}

// List of records by query parameters
// @Summary list of userJobs by query parameters
// @Description list of userJobs by paging and conditions
// @Tags userJob
// @accept json
// @Produce json
// @Param data body types.Params true "query parameters"
// @Success 200 {object} types.ListUserJobsReply{}
// @Router /api/v1/userJob/list [post]
// @Security BearerAuth
func (h *userJobHandler) List(c *gin.Context) {
	form := &types.ListUserJobsRequest{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		logger.Warn("ShouldBindJSON error: ", logger.Err(err), middleware.GCtxRequestIDField(c))
		response.Error(c, ecode.InvalidParams)
		return
	}

	ctx := middleware.WrapCtx(c)
	userJobs, total, err := h.iDao.GetByColumns(ctx, &form.Params)
	if err != nil {
		logger.Error("GetByColumns error", logger.Err(err), logger.Any("form", form), middleware.GCtxRequestIDField(c))
		response.Output(c, ecode.InternalServerError.ToHTTPCode())
		return
	}

	data, err := convertUserJobs(userJobs)
	if err != nil {
		response.Error(c, ecode.ErrListUserJob)
		return
	}

	response.Success(c, gin.H{
		"userJobs": data,
		"total":    total,
	})
}

// DeleteByIDs delete records by batch id
// @Summary delete userJobs
// @Description delete userJobs by batch id
// @Tags userJob
// @Param data body types.DeleteUserJobsByIDsRequest true "id array"
// @Accept json
// @Produce json
// @Success 200 {object} types.DeleteUserJobsByIDsReply{}
// @Router /api/v1/userJob/delete/ids [post]
// @Security BearerAuth
func (h *userJobHandler) DeleteByIDs(c *gin.Context) {
	form := &types.DeleteUserJobsByIDsRequest{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		logger.Warn("ShouldBindJSON error: ", logger.Err(err), middleware.GCtxRequestIDField(c))
		response.Error(c, ecode.InvalidParams)
		return
	}

	ctx := middleware.WrapCtx(c)
	err = h.iDao.DeleteByIDs(ctx, form.IDs)
	if err != nil {
		logger.Error("GetByIDs error", logger.Err(err), logger.Any("form", form), middleware.GCtxRequestIDField(c))
		response.Output(c, ecode.InternalServerError.ToHTTPCode())
		return
	}

	response.Success(c)
}

// GetByCondition get a record by condition
// @Summary get userJob by condition
// @Description get userJob by condition
// @Tags userJob
// @Param data body types.Conditions true "query condition"
// @Accept json
// @Produce json
// @Success 200 {object} types.GetUserJobByConditionReply{}
// @Router /api/v1/userJob/condition [post]
// @Security BearerAuth
func (h *userJobHandler) GetByCondition(c *gin.Context) {
	form := &types.GetUserJobByConditionRequest{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		logger.Warn("ShouldBindJSON error: ", logger.Err(err), middleware.GCtxRequestIDField(c))
		response.Error(c, ecode.InvalidParams)
		return
	}
	err = form.Conditions.CheckValid()
	if err != nil {
		logger.Warn("Parameters error: ", logger.Err(err), middleware.GCtxRequestIDField(c))
		response.Error(c, ecode.InvalidParams)
		return
	}

	ctx := middleware.WrapCtx(c)
	userJob, err := h.iDao.GetByCondition(ctx, &form.Conditions)
	if err != nil {
		if errors.Is(err, model.ErrRecordNotFound) {
			logger.Warn("GetByCondition not found", logger.Err(err), logger.Any("form", form), middleware.GCtxRequestIDField(c))
			response.Error(c, ecode.NotFound)
		} else {
			logger.Error("GetByCondition error", logger.Err(err), logger.Any("form", form), middleware.GCtxRequestIDField(c))
			response.Output(c, ecode.InternalServerError.ToHTTPCode())
		}
		return
	}

	data := &types.UserJobObjDetail{}
	err = copier.Copy(data, userJob)
	if err != nil {
		response.Error(c, ecode.ErrGetByIDUserJob)
		return
	}
	// Note: if copier.Copy cannot assign a value to a field, add it here

	response.Success(c, gin.H{"userJob": data})
}

// ListByIDs list of records by batch id
// @Summary list of userJobs by batch id
// @Description list of userJobs by batch id
// @Tags userJob
// @Param data body types.ListUserJobsByIDsRequest true "id array"
// @Accept json
// @Produce json
// @Success 200 {object} types.ListUserJobsByIDsReply{}
// @Router /api/v1/userJob/list/ids [post]
// @Security BearerAuth
func (h *userJobHandler) ListByIDs(c *gin.Context) {
	form := &types.ListUserJobsByIDsRequest{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		logger.Warn("ShouldBindJSON error: ", logger.Err(err), middleware.GCtxRequestIDField(c))
		response.Error(c, ecode.InvalidParams)
		return
	}

	ctx := middleware.WrapCtx(c)
	userJobMap, err := h.iDao.GetByIDs(ctx, form.IDs)
	if err != nil {
		logger.Error("GetByIDs error", logger.Err(err), logger.Any("form", form), middleware.GCtxRequestIDField(c))
		response.Output(c, ecode.InternalServerError.ToHTTPCode())
		return
	}

	userJobs := []*types.UserJobObjDetail{}
	for _, id := range form.IDs {
		if v, ok := userJobMap[id]; ok {
			record, err := convertUserJob(v)
			if err != nil {
				response.Error(c, ecode.ErrListUserJob)
				return
			}
			userJobs = append(userJobs, record)
		}
	}

	response.Success(c, gin.H{
		"userJobs": userJobs,
	})
}

// ListByLastID get records by last id and limit
// @Summary list of userJobs by last id and limit
// @Description list of userJobs by last id and limit
// @Tags userJob
// @accept json
// @Produce json
// @Param lastID query int true "last id, default is MaxInt32" default(0)
// @Param limit query int false "number per page" default(10)
// @Param sort query string false "sort by column name of table, and the "-" sign before column name indicates reverse order" default(-id)
// @Success 200 {object} types.ListUserJobsReply{}
// @Router /api/v1/userJob/list [get]
// @Security BearerAuth
func (h *userJobHandler) ListByLastID(c *gin.Context) {
	lastID := utils.StrToUint64(c.Query("lastID"))
	if lastID == 0 {
		lastID = math.MaxInt32
	}
	limit := utils.StrToInt(c.Query("limit"))
	if limit == 0 {
		limit = 10
	}
	sort := c.Query("sort")

	ctx := middleware.WrapCtx(c)
	userJobs, err := h.iDao.GetByLastID(ctx, lastID, limit, sort)
	if err != nil {
		logger.Error("GetByLastID error", logger.Err(err), logger.Uint64("latsID", lastID), logger.Int("limit", limit), middleware.GCtxRequestIDField(c))
		response.Output(c, ecode.InternalServerError.ToHTTPCode())
		return
	}

	data, err := convertUserJobs(userJobs)
	if err != nil {
		response.Error(c, ecode.ErrListByLastIDUserJob)
		return
	}

	response.Success(c, gin.H{
		"userJobs": data,
	})
}

func getUserJobIDFromPath(c *gin.Context) (string, uint64, bool) {
	idStr := c.Param("id")
	id, err := utils.StrToUint64E(idStr)
	if err != nil || id == 0 {
		logger.Warn("StrToUint64E error: ", logger.String("idStr", idStr), middleware.GCtxRequestIDField(c))
		return "", 0, true
	}

	return idStr, id, false
}

func convertUserJob(userJob *model.UserJob) (*types.UserJobObjDetail, error) {
	data := &types.UserJobObjDetail{}
	err := copier.Copy(data, userJob)
	if err != nil {
		return nil, err
	}
	// Note: if copier.Copy cannot assign a value to a field, add it here

	return data, nil
}

func convertUserJobs(fromValues []*model.UserJob) ([]*types.UserJobObjDetail, error) {
	toValues := []*types.UserJobObjDetail{}
	for _, v := range fromValues {
		data, err := convertUserJob(v)
		if err != nil {
			return nil, err
		}
		toValues = append(toValues, data)
	}

	return toValues, nil
}
