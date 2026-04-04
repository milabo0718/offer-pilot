package router

import (
	"github.com/milabo0718/offer-pilot/backend/controller/user"

	"github.com/gin-gonic/gin"
)

func RegisterUserRouter(r *gin.RouterGroup, uc *user.UserController) {
	{
		r.POST("/register", uc.Register)
		r.POST("/login", uc.Login)
		r.POST("/captcha", uc.HandleCaptcha)
	}
}
