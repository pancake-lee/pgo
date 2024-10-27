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

var _ UserDeptHandler = (*userDeptHandler)(nil)

// UserDeptHandler defining the handler interface
type UserDeptHandler interface {
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

type userDeptHandler struct {
	iDao dao.UserDeptDao
}

// NewUserDeptHandler creating the handler interface
func NewUserDeptHandler() UserDeptHandler {
	return &userDeptHandler{
		iDao: dao.NewUserDeptDao(
			model.GetDB(),
			cache.NewUserDeptCache(model.GetCacheType()),
		),
	}
}

// Create a record
// @Summary create userDept
// @Description submit information to create userDept
// @Tags userDept
// @accept json
// @Produce json
// @Param data body types.CreateUserDeptRequest true "userDept information"
// @Success 200 {object} types.CreateUserDeptReply{}
// @Router /api/v1/userDept [post]
// @Security BearerAuth
func (h *userDeptHandler) Create(c *gin.Context) {
	form := &types.CreateUserDeptRequest{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		logger.Warn("ShouldBindJSON error: ", logger.Err(err), middleware.GCtxRequestIDField(c))
		response.Error(c, ecode.InvalidParams)
		return
	}

	userDept := &model.UserDept{}
	err = copier.Copy(userDept, form)
	if err != nil {
		response.Error(c, ecode.ErrCreateUserDept)
		return
	}
	// Note: if copier.Copy cannot assign a value to a field, add it here

	ctx := middleware.WrapCtx(c)
	err = h.iDao.Create(ctx, userDept)
	if err != nil {
		logger.Error("Create error", logger.Err(err), logger.Any("form", form), middleware.GCtxRequestIDField(c))
		response.Output(c, ecode.InternalServerError.ToHTTPCode())
		return
	}

	response.Success(c, gin.H{"id": userDept.ID})
}

// DeleteByID delete a record by id
// @Summary delete userDept
// @Description delete userDept by id
// @Tags userDept
// @accept json
// @Produce json
// @Param id path string true "id"
// @Success 200 {object} types.DeleteUserDeptByIDReply{}
// @Router /api/v1/userDept/{id} [delete]
// @Security BearerAuth
func (h *userDeptHandler) DeleteByID(c *gin.Context) {
	_, id, isAbort := getUserDeptIDFromPath(c)
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
// @Summary update userDept
// @Description update userDept information by id
// @Tags userDept
// @accept json
// @Produce json
// @Param id path string true "id"
// @Param data body types.UpdateUserDeptByIDRequest true "userDept information"
// @Success 200 {object} types.UpdateUserDeptByIDReply{}
// @Router /api/v1/userDept/{id} [put]
// @Security BearerAuth
func (h *userDeptHandler) UpdateByID(c *gin.Context) {
	_, id, isAbort := getUserDeptIDFromPath(c)
	if isAbort {
		response.Error(c, ecode.InvalidParams)
		return
	}

	form := &types.UpdateUserDeptByIDRequest{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		logger.Warn("ShouldBindJSON error: ", logger.Err(err), middleware.GCtxRequestIDField(c))
		response.Error(c, ecode.InvalidParams)
		return
	}
	form.ID = id

	userDept := &model.UserDept{}
	err = copier.Copy(userDept, form)
	if err != nil {
		response.Error(c, ecode.ErrUpdateByIDUserDept)
		return
	}
	// Note: if copier.Copy cannot assign a value to a field, add it here

	ctx := middleware.WrapCtx(c)
	err = h.iDao.UpdateByID(ctx, userDept)
	if err != nil {
		logger.Error("UpdateByID error", logger.Err(err), logger.Any("form", form), middleware.GCtxRequestIDField(c))
		response.Output(c, ecode.InternalServerError.ToHTTPCode())
		return
	}

	response.Success(c)
}

// GetByID get a record by id
// @Summary get userDept detail
// @Description get userDept detail by id
// @Tags userDept
// @Param id path string true "id"
// @Accept json
// @Produce json
// @Success 200 {object} types.GetUserDeptByIDReply{}
// @Router /api/v1/userDept/{id} [get]
// @Security BearerAuth
func (h *userDeptHandler) GetByID(c *gin.Context) {
	_, id, isAbort := getUserDeptIDFromPath(c)
	if isAbort {
		response.Error(c, ecode.InvalidParams)
		return
	}

	ctx := middleware.WrapCtx(c)
	userDept, err := h.iDao.GetByID(ctx, id)
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

	data := &types.UserDeptObjDetail{}
	err = copier.Copy(data, userDept)
	if err != nil {
		response.Error(c, ecode.ErrGetByIDUserDept)
		return
	}
	// Note: if copier.Copy cannot assign a value to a field, add it here

	response.Success(c, gin.H{"userDept": data})
}

// List of records by query parameters
// @Summary list of userDepts by query parameters
// @Description list of userDepts by paging and conditions
// @Tags userDept
// @accept json
// @Produce json
// @Param data body types.Params true "query parameters"
// @Success 200 {object} types.ListUserDeptsReply{}
// @Router /api/v1/userDept/list [post]
// @Security BearerAuth
func (h *userDeptHandler) List(c *gin.Context) {
	form := &types.ListUserDeptsRequest{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		logger.Warn("ShouldBindJSON error: ", logger.Err(err), middleware.GCtxRequestIDField(c))
		response.Error(c, ecode.InvalidParams)
		return
	}

	ctx := middleware.WrapCtx(c)
	userDepts, total, err := h.iDao.GetByColumns(ctx, &form.Params)
	if err != nil {
		logger.Error("GetByColumns error", logger.Err(err), logger.Any("form", form), middleware.GCtxRequestIDField(c))
		response.Output(c, ecode.InternalServerError.ToHTTPCode())
		return
	}

	data, err := convertUserDepts(userDepts)
	if err != nil {
		response.Error(c, ecode.ErrListUserDept)
		return
	}

	response.Success(c, gin.H{
		"userDepts": data,
		"total":     total,
	})
}

// DeleteByIDs delete records by batch id
// @Summary delete userDepts
// @Description delete userDepts by batch id
// @Tags userDept
// @Param data body types.DeleteUserDeptsByIDsRequest true "id array"
// @Accept json
// @Produce json
// @Success 200 {object} types.DeleteUserDeptsByIDsReply{}
// @Router /api/v1/userDept/delete/ids [post]
// @Security BearerAuth
func (h *userDeptHandler) DeleteByIDs(c *gin.Context) {
	form := &types.DeleteUserDeptsByIDsRequest{}
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
// @Summary get userDept by condition
// @Description get userDept by condition
// @Tags userDept
// @Param data body types.Conditions true "query condition"
// @Accept json
// @Produce json
// @Success 200 {object} types.GetUserDeptByConditionReply{}
// @Router /api/v1/userDept/condition [post]
// @Security BearerAuth
func (h *userDeptHandler) GetByCondition(c *gin.Context) {
	form := &types.GetUserDeptByConditionRequest{}
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
	userDept, err := h.iDao.GetByCondition(ctx, &form.Conditions)
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

	data := &types.UserDeptObjDetail{}
	err = copier.Copy(data, userDept)
	if err != nil {
		response.Error(c, ecode.ErrGetByIDUserDept)
		return
	}
	// Note: if copier.Copy cannot assign a value to a field, add it here

	response.Success(c, gin.H{"userDept": data})
}

// ListByIDs list of records by batch id
// @Summary list of userDepts by batch id
// @Description list of userDepts by batch id
// @Tags userDept
// @Param data body types.ListUserDeptsByIDsRequest true "id array"
// @Accept json
// @Produce json
// @Success 200 {object} types.ListUserDeptsByIDsReply{}
// @Router /api/v1/userDept/list/ids [post]
// @Security BearerAuth
func (h *userDeptHandler) ListByIDs(c *gin.Context) {
	form := &types.ListUserDeptsByIDsRequest{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		logger.Warn("ShouldBindJSON error: ", logger.Err(err), middleware.GCtxRequestIDField(c))
		response.Error(c, ecode.InvalidParams)
		return
	}

	ctx := middleware.WrapCtx(c)
	userDeptMap, err := h.iDao.GetByIDs(ctx, form.IDs)
	if err != nil {
		logger.Error("GetByIDs error", logger.Err(err), logger.Any("form", form), middleware.GCtxRequestIDField(c))
		response.Output(c, ecode.InternalServerError.ToHTTPCode())
		return
	}

	userDepts := []*types.UserDeptObjDetail{}
	for _, id := range form.IDs {
		if v, ok := userDeptMap[id]; ok {
			record, err := convertUserDept(v)
			if err != nil {
				response.Error(c, ecode.ErrListUserDept)
				return
			}
			userDepts = append(userDepts, record)
		}
	}

	response.Success(c, gin.H{
		"userDepts": userDepts,
	})
}

// ListByLastID get records by last id and limit
// @Summary list of userDepts by last id and limit
// @Description list of userDepts by last id and limit
// @Tags userDept
// @accept json
// @Produce json
// @Param lastID query int true "last id, default is MaxInt32" default(0)
// @Param limit query int false "number per page" default(10)
// @Param sort query string false "sort by column name of table, and the "-" sign before column name indicates reverse order" default(-id)
// @Success 200 {object} types.ListUserDeptsReply{}
// @Router /api/v1/userDept/list [get]
// @Security BearerAuth
func (h *userDeptHandler) ListByLastID(c *gin.Context) {
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
	userDepts, err := h.iDao.GetByLastID(ctx, lastID, limit, sort)
	if err != nil {
		logger.Error("GetByLastID error", logger.Err(err), logger.Uint64("latsID", lastID), logger.Int("limit", limit), middleware.GCtxRequestIDField(c))
		response.Output(c, ecode.InternalServerError.ToHTTPCode())
		return
	}

	data, err := convertUserDepts(userDepts)
	if err != nil {
		response.Error(c, ecode.ErrListByLastIDUserDept)
		return
	}

	response.Success(c, gin.H{
		"userDepts": data,
	})
}

func getUserDeptIDFromPath(c *gin.Context) (string, uint64, bool) {
	idStr := c.Param("id")
	id, err := utils.StrToUint64E(idStr)
	if err != nil || id == 0 {
		logger.Warn("StrToUint64E error: ", logger.String("idStr", idStr), middleware.GCtxRequestIDField(c))
		return "", 0, true
	}

	return idStr, id, false
}

func convertUserDept(userDept *model.UserDept) (*types.UserDeptObjDetail, error) {
	data := &types.UserDeptObjDetail{}
	err := copier.Copy(data, userDept)
	if err != nil {
		return nil, err
	}
	// Note: if copier.Copy cannot assign a value to a field, add it here

	return data, nil
}

func convertUserDepts(fromValues []*model.UserDept) ([]*types.UserDeptObjDetail, error) {
	toValues := []*types.UserDeptObjDetail{}
	for _, v := range fromValues {
		data, err := convertUserDept(v)
		if err != nil {
			return nil, err
		}
		toValues = append(toValues, data)
	}

	return toValues, nil
}
