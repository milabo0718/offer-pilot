package user

import (
	"context"

	"github.com/milabo0718/offer-pilot/backend/common/code"
	myemail "github.com/milabo0718/offer-pilot/backend/common/email"
	myredis "github.com/milabo0718/offer-pilot/backend/common/redis"
	"github.com/milabo0718/offer-pilot/backend/dao/user"
	"github.com/milabo0718/offer-pilot/backend/utils"
	"github.com/milabo0718/offer-pilot/backend/utils/myjwt"
)

const (
	CodeMsg     = "GopherAI验证码如下(验证码仅限于2分钟有效): "
	UserNameMsg = "GopherAI的账号如下，请保留好，后续可以用账号进行登录 "
)

type UserService struct {
	userDao     *user.UserDao
	redisStore  *myredis.RedisStore
	jwtManager  *myjwt.JWTManager
	emailSender *myemail.EmailSender
}

func NewUserService(dao *user.UserDao, rStore *myredis.RedisStore, jwtMgr *myjwt.JWTManager, emailSender *myemail.EmailSender) *UserService {
	return &UserService{
		userDao:     dao,
		redisStore:  rStore,
		jwtManager:  jwtMgr,
		emailSender: emailSender,
	}
}

func (s *UserService) Login(ctx context.Context, username, password string) (string, code.Code) {
	ok, userInformation := s.userDao.IsExistUser(ctx, username)
	if !ok {
		return "", code.CodeUserNotExist
	}

	if !utils.CheckPasswordHash(password, userInformation.Password) {
		return "", code.CodeInvalidPassword
	}

	token, err := s.jwtManager.GenerateToken(userInformation.ID, userInformation.Username)
	if err != nil {
		return "", code.CodeServerBusy
	}

	return token, code.CodeSuccess
}

func (s *UserService) Register(ctx context.Context, email, password, captcha string) (string, code.Code) {
	ok, _ := s.userDao.IsExistUser(ctx, email)
	if ok {
		return "", code.CodeUserExist
	}

	ok, _ = s.redisStore.CheckCaptchaForEmail(ctx, email, captcha)
	if !ok {
		return "", code.CodeInvalidCaptcha
	}

	username := utils.GetRandomNumbers(11)

	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return "", code.CodeServerBusy
	}

	userInformation, ok := s.userDao.Register(ctx, username, email, hashedPassword)
	if !ok {
		return "", code.CodeServerBusy
	}

	if err := s.emailSender.SendCaptcha(email, username, UserNameMsg); err != nil {
		return "", code.CodeServerBusy
	}

	token, err := s.jwtManager.GenerateToken(userInformation.ID, userInformation.Username)
	if err != nil {
		return "", code.CodeServerBusy
	}

	return token, code.CodeSuccess
}

func (s *UserService) SendCaptcha(ctx context.Context, email_ string) code.Code {
	send_code := utils.GetRandomNumbers(6)

	if err := s.redisStore.SetCaptchaForEmail(ctx, email_, send_code); err != nil {
		return code.CodeServerBusy
	}

	if err := s.emailSender.SendCaptcha(email_, send_code, CodeMsg); err != nil {
		return code.CodeServerBusy
	}

	return code.CodeSuccess
}
