package email

import (
	"fmt"

	"github.com/milabo0718/offer-pilot/backend/config"

	"gopkg.in/gomail.v2"
)

const (
	CodeMsg     = "GopherAI验证码如下(验证码仅限于2分钟有效): "
	UserNameMsg = "GopherAI的账号如下，请保留好，后续可以用账号/邮箱登录 "
)

type EmailSender struct {
	email    string
	authCode string
	host     string
	port     int
}

func NewEmailSender(conf *config.EmailConfig) *EmailSender {
	return &EmailSender{
		email:    conf.Email,
		authCode: conf.AuthCode,
		host:     "smtp.qq.com",
		port:     587,
	}
}

func (e *EmailSender) SendCaptcha(email, code, msg string) error {
	m := gomail.NewMessage()

	// 发件人
	m.SetHeader("From", e.email)
	// 收件人
	m.SetHeader("To", email)
	// 主题
	m.SetHeader("Subject", "来自offerpilot的信息")
	// 正文内容
	m.SetBody("text/plain", msg+" "+code)

	// 配置 SMTP 服务器和授权码,587：是 SMTP 的明文/STARTTLS 端口号
	d := gomail.NewDialer("smtp.qq.com", 587, e.email, e.authCode)

	// 发送邮件
	if err := d.DialAndSend(m); err != nil {
		fmt.Printf("DialAndSend err %v:\n", err)
		return err
	}
	fmt.Printf("send mail success\n")
	return nil
}
