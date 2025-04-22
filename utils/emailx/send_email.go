// Copyright 2025 eventmatrix.cn
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package emailx

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"

	"github.com/garrickvan/event-matrix/utils/logx"
)

// sendToMail 发送电子邮件的函数，支持不同的邮件类型和服务器
func sendToMail(user, sendUserName, password, host, to, subject, body, mailtype string) error {
	logx.SugarLog().Debug("SendToMail: ", map[string]interface{}{
		"user":         user,
		"sendUserName": sendUserName,
		"password":     password,
		"host":         host,
		"to":           to,
		"subject":      subject,
		"body":         body,
		"mailtype":     mailtype,
	})
	hp := strings.Split(host, ":")
	if len(hp) != 2 {
		return errors.New("host格式错误")
	}
	auth := smtp.PlainAuth("", user, password, hp[0])
	var content_type string
	if mailtype == "html" {
		content_type = "Content-Type: text/" + mailtype + "; charset=UTF-8"
	} else {
		content_type = "Content-Type: text/plain" + "; charset=UTF-8"
	}
	msg := []byte("To: " + to + "\r\nFrom: " + sendUserName + "<" + user + ">" + "\r\nSubject: " + subject + "\r\n" + content_type + "\r\n\r\n" + body)
	send_to := strings.Split(to, ";")
	if len(send_to) != 1 {
		return errors.New("不支持群发")
	}
	// 根据邮箱类型执行不同的发送方法
	if strings.HasPrefix(host, "smtp.126") {
		return smtp.SendMail(host, auth, user, send_to, msg)
	}
	if strings.HasPrefix(host, "smtp.qcloudmail") {
		return sendEmailByQCloud(hp[0], hp[1], user, password, send_to[0], sendUserName, subject, body)
	}
	return errors.New("unsupported email server")
}

// sendEmailByQCloud 使用腾讯云邮件服务发送电子邮件的函数
func sendEmailByQCloud(host, port, email, password, toEmail, sendUserName, subject, body string) error {
	header := make(map[string]string)
	header["From"] = sendUserName + " <" + email + ">"
	header["To"] = toEmail
	header["Subject"] = subject
	header["Content-Type"] = "text/html; charset=UTF-8"

	message := ""
	for k, v := range header {
		message += (k + ": " + v + "\r\n")
	}
	message += "\r\n" + body

	auth := smtp.PlainAuth(
		"",
		email,
		password,
		host,
	)

	err := sendMailWithTLS(
		host+":"+port,
		auth,
		email,
		[]string{toEmail},
		[]byte(message),
	)

	if err != nil {
		fmt.Println("Send email error:", err)
	} else {
		fmt.Println("Send mail success!")
	}

	return err
}

// Dial 返回一个 SMTP 客户端
func dial(addr string) (*smtp.Client, error) {
	conn, err := tls.Dial("tcp", addr, nil)
	if err != nil {
		return nil, err
	}

	host, _, _ := net.SplitHostPort(addr)
	return smtp.NewClient(conn, host)
}

// sendMailWithTLS 使用 TLS 发送电子邮件的函数
func sendMailWithTLS(addr string, auth smtp.Auth, from string,
	to []string, msg []byte) (err error) {
	c, err := dial(addr)
	if err != nil {
		logx.Log().Error("Dial Error: " + err.Error() + ", addr: " + addr)
		return err
	}
	defer c.Close()

	if auth != nil {
		if ok, _ := c.Extension("AUTH"); ok {
			if err = c.Auth(auth); err != nil {
				logx.Log().Error("Auth error: " + err.Error() + ", addr: " + addr)
				return err
			}
		}
	}

	if err = c.Mail(from); err != nil {
		return err
	}

	for _, addr := range to {
		if err = c.Rcpt(addr); err != nil {
			return err
		}
	}

	w, err := c.Data()
	if err != nil {
		return err
	}

	_, err = w.Write(msg)
	if err != nil {
		return err
	}

	err = w.Close()
	if err != nil {
		return err
	}

	return c.Quit()
}

// SendHTMLEmail 发送 HTML 格式的电子邮件的函数
func SendHTMLEmail(email, subject, sendUserName, body, emailSvrUser, emailSvrPass, emailSvrHost, code string) error {
	return sendToMail(
		strings.TrimSpace(emailSvrUser),
		sendUserName,
		strings.TrimSpace(emailSvrPass),
		strings.TrimSpace(emailSvrHost),
		email,
		subject,
		body,
		"html")
}
