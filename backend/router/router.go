package router

import (
	ragcontroller "github.com/milabo0718/offer-pilot/backend/controller/rag"
	"github.com/milabo0718/offer-pilot/backend/controller/session"
	sttcontroller "github.com/milabo0718/offer-pilot/backend/controller/stt"
	ttscontroller "github.com/milabo0718/offer-pilot/backend/controller/tts"
	"github.com/milabo0718/offer-pilot/backend/controller/user"

	"github.com/gin-gonic/gin"
	"github.com/milabo0718/offer-pilot/backend/middleware/jwt"
	"github.com/milabo0718/offer-pilot/backend/utils/myjwt"
)

func InitRouter(
	userController *user.UserController,
	sessionController *session.SessionController,
	ragController *ragcontroller.RAGController,
	ttsController *ttscontroller.TTSController,
	sttController *sttcontroller.STTController,
	jwtManager *myjwt.JWTManager,
) *gin.Engine {

	r := gin.Default()
	enterRouter := r.Group("/api/v1")
	{
		usergroup := enterRouter.Group("/user")
		RegisterUserRouter(usergroup, userController)
	}
	{
		aigroup := enterRouter.Group("/ai")
		aigroup.Use(jwt.Auth(jwtManager))
		AIRouter(aigroup, sessionController, ragController, ttsController, sttController)
	}
	return r
}
