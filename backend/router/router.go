package router

import (
	"github.com/milabo0718/offer-pilot/backend/controller/user"

	"github.com/gin-gonic/gin"
)

func InitRouter(userController *user.UserController) *gin.Engine {

	r := gin.Default()
	enterRouter := r.Group("/api/v1")
	{
		RegisterUserRouter(enterRouter.Group("/user"), userController)
	}

	return r
}
