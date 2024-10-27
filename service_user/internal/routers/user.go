package routers

import (
	"github.com/gin-gonic/gin"

	"gogogo/service_user/internal/handler"
)

func init() {
	apiV1RouterFns = append(apiV1RouterFns, func(group *gin.RouterGroup) {
		userRouter(group, handler.NewUserHandler())
	})
}

func userRouter(group *gin.RouterGroup, h handler.UserHandler) {
	g := group.Group("/user")

	// All the following routes use jwt authentication, you also can use middleware.Auth(middleware.WithVerify(fn))
	//g.Use(middleware.Auth())

	// If jwt authentication is not required for all routes, authentication middleware can be added
	// separately for only certain routes. In this case, g.Use(middleware.Auth()) above should not be used.

	g.POST("/", h.Create)          // [post] /api/v1/user
	g.DELETE("/:id", h.DeleteByID) // [delete] /api/v1/user/:id
	g.PUT("/:id", h.UpdateByID)    // [put] /api/v1/user/:id
	g.GET("/:id", h.GetByID)       // [get] /api/v1/user/:id
	g.POST("/list", h.List)        // [post] /api/v1/user/list

	g.POST("/delete/ids", h.DeleteByIDs)   // [post] /api/v1/user/delete/ids
	g.POST("/condition", h.GetByCondition) // [post] /api/v1/user/condition
	g.POST("/list/ids", h.ListByIDs)       // [post] /api/v1/user/list/ids
	g.GET("/list", h.ListByLastID)         // [get] /api/v1/user/list
}
