package routers

import (
	"github.com/gin-gonic/gin"

	"gogogo/service_user/internal/handler"
)

func init() {
	apiV1RouterFns = append(apiV1RouterFns, func(group *gin.RouterGroup) {
		userDeptRouter(group, handler.NewUserDeptHandler())
	})
}

func userDeptRouter(group *gin.RouterGroup, h handler.UserDeptHandler) {
	g := group.Group("/userDept")

	// All the following routes use jwt authentication, you also can use middleware.Auth(middleware.WithVerify(fn))
	//g.Use(middleware.Auth())

	// If jwt authentication is not required for all routes, authentication middleware can be added
	// separately for only certain routes. In this case, g.Use(middleware.Auth()) above should not be used.

	g.POST("/", h.Create)          // [post] /api/v1/userDept
	g.DELETE("/:id", h.DeleteByID) // [delete] /api/v1/userDept/:id
	g.PUT("/:id", h.UpdateByID)    // [put] /api/v1/userDept/:id
	g.GET("/:id", h.GetByID)       // [get] /api/v1/userDept/:id
	g.POST("/list", h.List)        // [post] /api/v1/userDept/list

	g.POST("/delete/ids", h.DeleteByIDs)   // [post] /api/v1/userDept/delete/ids
	g.POST("/condition", h.GetByCondition) // [post] /api/v1/userDept/condition
	g.POST("/list/ids", h.ListByIDs)       // [post] /api/v1/userDept/list/ids
	g.GET("/list", h.ListByLastID)         // [get] /api/v1/userDept/list
}
