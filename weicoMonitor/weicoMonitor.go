package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/smtp"
	"strings"
	"time"
)

/*
	校验对象
*/
type BaseMonitor struct {
	url string
}

/*
	发起一次http请求，获取url对应的请求体
*/
func (monitor *BaseMonitor) doRequest() (string, error) {
	resp, err := http.Get(monitor.url)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), err
}

/*
	进行校验，判定该服务是否正常
*/
func (monitor *BaseMonitor) checkExpect(realResult string) (string, bool) {
	if realResult == "" { //空校验
		return "result is empty", false
	}
	if !(strings.HasPrefix(realResult, "{") || strings.HasPrefix(realResult, "[")) { //json格式校验
		return "expect start with '{' or '[',but it's not", false
	}
	if strings.Contains(realResult, "error") { //异常校验
		return "find 'error' in result ", false
	}
	return "pass", true
}

/*
	监听总入口
*/
type WeicoMonitor struct {
	monitors       []BaseMonitor //需要进行监控的部分
	mailUser       string        //发邮件必须的信息
	mailPassword   string
	mailHost       string
	notifyEmails   string //邮件接收者
	lastNofityTime int64  //上一次通知的时间，两次通知时间间隔不能太近
	mailtype       string //邮件通知类型，一般是html、plain
}

/*
	发送邮件
*/
func (weicoMonitor *WeicoMonitor) SendToMail(subject, body string) error {
	fmt.Println(body)
	host_port := strings.Split(weicoMonitor.mailHost, ":")[0]
	user := weicoMonitor.mailUser
	auth := smtp.PlainAuth("", user, weicoMonitor.mailPassword, host_port)

	var content_type string
	if weicoMonitor.mailtype == "html" {
		content_type = "Content-Type: text/html; charset=UTF-8"
	} else {
		content_type = "Content-Type: text/plain; charset=UTF-8"
	}

	msg := []byte("To: " + weicoMonitor.notifyEmails + "\r\nFrom: " + user + "<" + user + ">\r\nSubject: " + subject + "\r\n" + content_type + "\r\n\r\n" + body)

	send_to := strings.Split(weicoMonitor.notifyEmails, ";")
	err := smtp.SendMail(weicoMonitor.mailHost, auth, user, send_to, msg)
	return err
}

/*
	检查url访问，如果url返回结果异常，则进行记录并发送通知邮件
*/
func (wm *WeicoMonitor) doCheck() {
	check_fail := false
	check_result := []string{}

	for index, _monitor := range wm.monitors {
		fmt.Println("start check", index, _monitor)
		responseContent, err := _monitor.doRequest()
		if err != nil { //http请求出现问题
			check_result = append(check_result, fmt.Sprintf("reuqest url %v fail:%v \n", _monitor.url, err))
			check_fail = true
			continue
		}
		msg, pass := _monitor.checkExpect(responseContent)
		if !pass { //校验出现问题
			check_result = append(check_result, fmt.Sprintf("reuqest url %v fail:%v \n", _monitor.url, msg))
			check_fail = true
		}
	}

	//判断是否校验失败，如果失败，则集结错误信息，并发送邮件
	if check_fail {
		fmt.Println("find exception,send mail")
		subject := "weico service exception notifition"
		body := fmt.Sprintf("<html><body><div> 服务器发现异常: <br/> %v </div></body></html>",
			strings.Join(check_result, "<br/>"))
		err := wm.SendToMail(subject, body)
		if err != nil {
			fmt.Println("fail to send mail", err)
		}
		wm.lastNofityTime = time.Now().Unix()
	}
}

func main() {
	weicoMonitor := WeicoMonitor{
		mailUser:     "qhandroid@sina.com",                     //发送源
		mailPassword: "qhandroid",                              //密码
		mailHost:     "smtp.sina.com:25",                       //服务器
		notifyEmails: "qihigh@qq.com;zhaoshuai@eicodesign.com", //目标用户
		mailtype:     "html",                                   //邮件通知类型，一般是html、plain
	}

	// weicoMonitor := WeicoMonitor{
	// 	mailUser:     "qihuan@weico.com",                       //发送源
	// 	mailPassword: "XXXXXXXX",                               //密码
	// 	mailHost:     "smtp.ym.163.com:25",                     //服务器
	// 	notifyEmails: "qihigh@qq.com;zhaoshuai@eicodesign.com", //目标用户
	// 	mailtype:     "html",                                   //邮件通知类型，一般是html、plain
	// }

	monitors := []BaseMonitor{}
	//添加多个url监控
	monitors = append(monitors, BaseMonitor{"http://weico3.weico.cc/v3/hot/topic"})
	monitors = append(monitors, BaseMonitor{"http://weico3.weico.cc/v3/skin?platform_id=69"})

	//更新monitor
	weicoMonitor.monitors = monitors

	//开启无限循环，进行校验
	for {
		weicoMonitor.doCheck()
		time.Sleep(10 * time.Minute)
	}

	// subject := "weico service exception notifition"
	// fmt.Println(subject)
	// err := weicoMonitor.SendToMail(subject, "<html><body><div> 服务器发现异常 adfaccccccccsdf  </div></body></html>")
	// if err != nil {
	// 	fmt.Println("fail to send mail", err)
	// }

}
