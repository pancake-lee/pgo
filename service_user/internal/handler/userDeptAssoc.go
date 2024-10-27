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

var _ UserDeptAssocHandler = (*userDeptAssocHandler)(nil)

// UserDeptAssocHandler defining the handler interface
type UserDeptAssocHandler interface {
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

type userDeptAssocHandler struct {
	iDao dao.UserDeptAssocDao
}

// NewUserDeptAssocHandler creating the handler interface
func NewUserDeptAssocHandler() UserDeptAssocHandler {
	return &userDeptAssocHandler{
		iDao: dao.NewUserDeptAssocDao(
			model.GetDB(),
			cache.NewUserDeptAssocCache(model.GetCacheType()),
		),
	}
}

// Create a record
// @Summary create userDeptAssoc
// @Description submit information to create userDeptAssoc
// @Tags userDeptAssoc
// @accept json
// @Produce json
// @Param data body types.CreateUserDeptAssocRequest true "userDeptAssoc information"
// @Success 200 {object} types.CreateUserDeptAssocReply{}
// @Router /api/v1/userDeptAssoc [post]
// @Security BearerAuth
func (h *userDeptAssocHandler) Create(c *gin.Context) {
	form := &types.CreateUserDeptAssocRequest{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		logger.Warn("ShouldBindJSON error: ", logger.Err(err), middleware.GCtxRequestIDField(c))
		response.Error(c, ecode.InvalidParams)
		return
	}

	userDeptAssoc := &model.UserDeptAssoc{}
	err = copier.Copy(userDeptAssoc, form)
	if err != nil {
		response.Error(c, ecode.ErrCreateUserDeptAssoc)
		return
	}
	// Note: if copier.Copy cannot assign a value to a field, add it here

	ctx := middleware.WrapCtx(c)
	err = h.iDao.Create(ctx, userDeptAssoc)
	if err != nil {
		logger.Error("Create error", logger.Err(err), logger.Any("form", form), middleware.GCtxRequestIDField(c))
		response.Output(c, ecode.InternalServerError.ToHTTPCode())
		return
	}

	response.Success(c, gin.H{"id": userDeptAssoc.ID})
}

// DeleteByID delete a record by id
// @Summary delete userDeptAssoc
// @Description delete userDeptAssoc by id
// @Tags userDeptAssoc
// @accept json
// @Produce json
// @Param id path string true "id"
// @Success 200 {object} types.DeleteUserDeptAssocByIDReply{}
// @Router /api/v1/userDeptAssoc/{id} [delete]
// @Security BearerAuth
func (h *userDeptAssocHandler) DeleteByID(c *gin.Context) {
	_, id, isAbort := getUserDeptAssocIDFromPath(c)
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
// @Summary update userDeptAssoc
// @Description update userDeptAssoc information by id
// @Tags userDeptAssoc
// @accept json
// @Produce json
// @Param id path string true "id"
// @Param data body types.UpdateUserDeptAssocByIDRequest true "userDeptAssoc information"
// @Success 200 {object} types.UpdateUserDeptAssocByIDReply{}
// @Router /api/v1/userDeptAssoc/{id} [put]
// @Security BearerAuth
func (h *userDeptAssocHandler) UpdateByID(c *gin.Context) {
	_, id, isAbort := getUserDeptAssocIDFromPath(c)
	if isAbort {
		response.Error(c, ecode.InvalidParams)
		return
	}

	form := &types.UpdateUserDeptAssocByIDRequest{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		logger.Warn("ShouldBindJSON error: ", logger.Err(err), middleware.GCtxRequestIDField(c))
		response.Error(c, ecode.InvalidParams)
		return
	}
	form.ID = id

	userDeptAssoc := &model.UserDeptAssoc{}
	err = copier.Copy(userDeptAssoc, form)
	if err != nil {
		response.Error(c, ecode.ErrUpdateByIDUserDeptAssoc)
		return
	}
	// Note: if copier.Copy cannot assign a value to a field, add it here

	ctx := middleware.WrapCtx(c)
	err = h.iDao.UpdateByID(ctx, userDeptAssoc)
	if err != nil {
		logger.Error("UpdateByID error", logger.Err(err), logger.Any("form", form), middleware.GCtxRequestIDField(c))
		response.Output(c, ecode.InternalServerError.ToHTTPCode())
		return
	}

	response.Success(c)
}

// GetByID get a record by id
// @Summary get userDeptAssoc detail
// @Description get userDeptAssoc detail by id
// @Tags userDeptAssoc
// @Param id path string true "id"
// @Accept json
// @Produce json
// @Success 200 {object} types.GetUserDeptAssocByIDReply{}
// @Router /api/v1/userDeptAssoc/{id} [get]
// @Security BearerAuth
func (h *userDeptAssocHandler) GetByID(c *gin.Context) {
	_, id, isAbort := getUserDeptAssocIDFromPath(c)
	if isAbort {
		response.Error(c, ecode.InvalidParams)
		return
	}

	ctx := middleware.WrapCtx(c)
	userDeptAssoc, err := h.iDao.GetByID(ctx, id)
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

	data := &types.UserDeptAssocObjDetail{}
	err = copier.Copy(data, userDeptAssoc)
	if err != nil {
		response.Error(c, ecode.ErrGetByIDUserDeptAssoc)
		return
	}
	// Note: if copier.Copy cannot assign a value to a field, add it here

	response.Success(c, gin.H{"userDeptAssoc": data})
}

// List of records by query parameters
// @Summary list of userDeptAssocs by query parameters
// @Description list of userDeptAssocs by paging and conditions
// @Tags userDeptAssoc
// @accept json
// @Produce json
// @Param data body types.Params true "query parameters"
// @Success 200 {object} types.ListUserDeptAssocsReply{}
// @Router /api/v1/userDeptAssoc/list [post]
// @Security BearerAuth
func (h *userDeptAssocHandler) List(c *gin.Context) {
	form := &types.ListUserDeptAssocsRequest{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		logger.Warn("ShouldBindJSON error: ", logger.Err(err), middleware.GCtxRequestIDField(c))
		response.Error(c, ecode.InvalidParams)
		return
	}

	ctx := middleware.WrapCtx(c)
	userDeptAssocs, total, err := h.iDao.GetByColumns(ctx, &form.Params)
	if err != nil {
		logger.Error("GetByColumns error", logger.Err(err), logger.Any("form", form), middleware.GCtxRequestIDField(c))
		response.Output(c, ecode.InternalServerError.ToHTTPCode())
		return
	}

	data, err := convertUserDeptAssocs(userDeptAssocs)
	if err != nil {
		response.Error(c, ecode.ErrListUserDeptAssoc)
		return
	}

	response.Success(c, gin.H{
		"userDeptAssocs": data,
		"total":        total,
	})
}

// DeleteByIDs delete records by batch id
// @Summary delete userDeptAssocs
// @Description delete userDeptAssocs by batch id
// @Tags userDeptAssoc
// @Param data body types.DeleteUserDeptAssocsByIDsRequest true "id array"
// @Accept json
// @Produce json
// @Success 200 {object} types.DeleteUserDeptAssocsByIDsReply{}
// @Router /api/v1/userDeptAssoc/delete/ids [post]
// @Security BearerAuth
func (h *userDeptAssocHandler) DeleteByIDs(c *gin.Context) {
	form := &types.DeleteUserDeptAssocsByIDsRequest{}
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
// @Summary get userDeptAssoc by condition
// @Description get userDeptAssoc by condition
// @Tags userDeptAssoc
// @Param data body types.Conditions true "query condition"
// @Accept json
// @Produce json
// @Success 200 {object} types.GetUserDeptAssocByConditionReply{}
// @Router /api/v1/userDeptAssoc/condition [post]
// @Security BearerAuth
func (h *userDeptAssocHandler) GetByCondition(c *gin.Context) {
	form := &types.GetUserDeptAssocByConditionRequest{}
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
	userDeptAssoc, err := h.iDao.GetByCondition(ctx, &form.Conditions)
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

	data := &types.UserDeptAssocObjDetail{}
	err = copier.Copy(data, userDeptAssoc)
	if err != nil {
		response.Error(c, ecode.ErrGetByIDUserDeptAssoc)
		return
	}
	// Note: if copier.Copy cannot assign a value to a field, add it here

	response.Success(c, gin.H{"userDeptAssoc": data})
}

// ListByIDs list of records by batch id
// @Summary list of userDeptAssocs by batch id
// @Description list of userDeptAssocs by batch id
// @Tags userDeptAssoc
// @Param data body types.ListUserDeptAssocsByIDsRequest true "id array"
// @Accept json
// @Produce json
// @Success 200 {object} types.ListUserDeptAssocsByIDsReply{}
// @Router /api/v1/userDeptAssoc/list/ids [post]
// @Security BearerAuth
func (h *userDeptAssocHandler) ListByIDs(c *gin.Context) {
	form := &types.ListUserDeptAssocsByIDsRequest{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		logger.Warn("ShouldBindJSON error: ", logger.Err(err), middleware.GCtxRequestIDField(c))
		response.Error(c, ecode.InvalidParams)
		return
	}

	ctx := middleware.WrapCtx(c)
	userDeptAssocMap, err := h.iDao.GetByIDs(ctx, form.IDs)
	if err != nil {
		logger.Error("GetByIDs error", logger.Err(err), logger.Any("form", form), middleware.GCtxRequestIDField(c))
		response.Output(c, ecode.InternalServerError.ToHTTPCode())
		return
	}

	userDeptAssocs := []*types.UserDeptAssocObjDetail{}
	for _, id := range form.IDs {
		if v, ok := userDeptAssocMap[id]; ok {
			record, err := convertUserDeptAssoc(v)
			if err != nil {
				response.Error(c, ecode.ErrListUserDeptAssoc)
				return
			}
			userDeptAssocs = append(userDeptAssocs, record)
		}
	}

	response.Success(c, gin.H{
		"userDeptAssocs": userDeptAssocs,
	})
}

// ListByLastID get records by last id and limit
// @Summary list of userDeptAssocs by last id and limit
// @Description list of userDeptAssocs by last id and limit
// @Tags userDeptAssoc
// @accept json
// @Produce json
// @Param lastID query int true "last id, default is MaxInt32" default(0)
// @Param limit query int false "number per page" default(10)
// @Param sort query string false "sort by column name of table, and the "-" sign before column name indicates reverse order" default(-id)
// @Success 200 {object} types.ListUserDeptAssocsReply{}
// @Router /api/v1/userDeptAssoc/list [get]
// @Security BearerAuth
func (h *userDeptAssocHandler) ListByLastID(c *gin.Context) {
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
	userDeptAssocs, err := h.iDao.GetByLastID(ctx, lastID, limit, sort)
	if err != nil {
		logger.Error("GetByLastID error", logger.Err(err), logger.Uint64("latsID", lastID), logger.Int("limit", limit), middleware.GCtxRequestIDField(c))
		response.Output(c, ecode.InternalServerError.ToHTTPCode())
		return
	}

	data, err := convertUserDeptAssocs(userDeptAssocs)
	if err != nil {
		response.Error(c, ecode.ErrListByLastIDUserDeptAssoc)
		return
	}

	response.Success(c, gin.H{
		"userDeptAssocs": data,
	})
}

func getUserDeptAssocIDFromPath(c *gin.Context) (string, uint64, bool) {
	idStr := c.Param("id")
	id, err := utils.StrToUint64E(idStr)
	if err != nil || id == 0 {
		logger.Warn("StrToUint64E error: ", logger.String("idStr", idStr), middleware.GCtxRequestIDField(c))
		return "", 0, true
	}

	return idStr, id, false
}

func convertUserDeptAssoc(userDeptAssoc *model.UserDeptAssoc) (*types.UserDeptAssocObjDetail, error) {
	data := &types.UserDeptAssocObjDetail{}
	err := copier.Copy(data, userDeptAssoc)
	if err != nil {
		return nil, err
	}
	// Note: if copier.Copy cannot assign a value to a field, add it here

	return data, nil
}

func convertUserDeptAssocs(fromValues []*model.UserDeptAssoc) ([]*types.UserDeptAssocObjDetail, error) {
	toValues := []*types.UserDeptAssocObjDetail{}
	for _, v := range fromValues {
		data, err := convertUserDeptAssoc(v)
		if err != nil {
			return nil, err
		}
		toValues = append(toValues, data)
	}

	return toValues, nil
}
