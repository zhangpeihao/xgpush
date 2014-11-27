package xgpush

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"
)

const (
	XGPUSH_SCHEMA                     = "http://"
	XGPUSH_HOST                       = "openapi.xg.qq.com"
	XGPUSH_METHOD                     = "POST"
	XGPUSH_V2_BASE_URL                = XGPUSH_HOST + `/v2/`
	XGPUSH_V2_BASE_URL_WITH_SCHEMA    = XGPUSH_SCHEMA + XGPUSH_HOST + `/v2/`
	XGPUSH_VALID_TIME                 = "600"
	XGPUSH_POST_CONTENT_TYPE          = "application/x-www-form-urlencoded"
	XGPUSH_PUSH_SINGLE_ACCOUNT_METHOD = "push/single_account"
	XGPUSH_PUSH_SINGLE_DEVICE_METHOD  = "push/single_device"
	XGPUSH_PUSH_ACCOUNT_LIST_METHOD   = "push/account_list"
	XGPUSH_PUSH_ALL_DEVICE_METHOD     = "push/all_device"
	XGPUSH_GET_APP_DEVICE_NUM_METHOD  = "application/get_app_device_num"
)

const (
	XGPushEnviroment_Product = "1"
	XGPushEnviroment_Develop = "2"

	XGPushMessageType_Notification = "1"
	XGPushMessageType_Message      = "2"

	XGPushDeviceType_IOS     = 1
	XGPushDeviceType_Android = 2
)

var (
	Environment = XGPushEnviroment_Develop
)

type XGPushMsg struct {
	Method string
	Params map[string]string
}

type XGPushConn struct {
	xgpush *XGPush
	queue  <-chan *XGPushMsg
}

type XGPushParameters struct {
	Param_access_id   string
	Param_secret_key  string
	Param_connections int
	Param_queue_size  int
	Param_timeout     time.Duration
}

type XGPush struct {
	XGPushParameters
	queue chan *XGPushMsg
	conns chan *XGPushConn
}

// ========================================
// Protocol
type PushResponse struct {
	RetCode int         `json:"ret_code"`
	ErrMsg  string      `json:"err_msg"`
	Result  interface{} `json:"result"`
}
type GetAppDeviceNumResult struct {
	DeviceNum int `json:"device_num"`
}
type GetAppDeviceNumResponse struct {
	RetCode int                   `json:"ret_code"`
	ErrMsg  string                `json:"err_msg"`
	Result  GetAppDeviceNumResult `json:"result"`
}

// ========================================
// XGPush
func NewXGPush(parameters *XGPushParameters) (xgpush *XGPush) {
	xgpush = &XGPush{
		XGPushParameters: *parameters,
		queue:            make(chan *XGPushMsg, parameters.Param_queue_size),
		conns:            make(chan *XGPushConn, parameters.Param_connections),
	}
	for i := 0; i < parameters.Param_connections; i++ {
		xgpush.conns <- NewXGPushConn(xgpush, xgpush.queue)
	}
	return
}

// 内容签名。生成规则：
// A）提取请求方法method（GET或POST）；
// B）提取请求url信息，包括Host字段的IP或域名和URI的path部分，注意不包括Host的端口和Path的querystring。请在请求中带上Host字段，否则将视为无效请求。比如openapi.xg.qq.com/v2/push/single_device或者10.198.18.239/v2/push/single_device;
// C）将请求参数（不包括sign参数）格式化成K=V方式，注意：计算sign时所有参数不应进行urlencode；
// D）将格式化后的参数以K的字典序升序排列，拼接在一起，注意字典序中大写字母在前；
// E）拼接请求方法、url、排序后格式化的字符串以及应用的secret_key；
// F）将E形成字符串计算MD5值，形成一个32位的十六进制（字母小写）字符串，即为本次请求sign（签名）的值；
// Sign=MD5($http_method$url$k1=$v1$k2=$v2$secret_key); 该签名值基本可以保证请求是合法者发送且参数没有被修改，但无法保证不被偷窥。
// 例如： POST请求到接口http://openapi.xg.qq.com/v2/push/single_device，有四个参数，access_id=123，timestamp=1386691200，Param1=Value1，Param2=Value2，secret_key为abcde。则上述E步骤拼接出的字符串为POSTopenapi.xg.qq.com/v2/push/single_deviceParam1=Value1Param2=Value2access_id=123timestamp=1386691200abcde，注意字典序中大写在前。计算出该字符串的MD5为ccafecaef6be07493cfe75ebc43b7d53，以此作为sign参数的值
func (xgpush *XGPush) sign(method string, params map[string]string) (buf *bytes.Buffer) {
	params["access_id"] = xgpush.Param_access_id
	if _, found := params["timestamp"]; !found {
		params["timestamp"] = strconv.FormatInt(time.Now().Unix(), 10)
	}
	//params["valid_time"] = XGPUSH_VALID_TIME
	h := md5.New()
	io.WriteString(h, XGPUSH_METHOD)
	io.WriteString(h, XGPUSH_V2_BASE_URL)
	io.WriteString(h, method)
	var kvs []string
	for key, value := range params {
		kvs = append(kvs, key+"="+value)
	}
	sort.Sort(sort.StringSlice(kvs))
	for _, kv := range kvs {
		io.WriteString(h, kv)
	}
	io.WriteString(h, xgpush.Param_secret_key)
	params["sign"] = fmt.Sprintf("%x", h.Sum(nil))
	return
}

func (xgpush *XGPush) Post(msg *XGPushMsg) (resp *http.Response, err error) {
	xgpush.sign(msg.Method, msg.Params)
	buf := new(bytes.Buffer)
	first := true
	for key, value := range msg.Params {
		if first {
			first = false
		} else {
			buf.WriteByte('&')
		}
		buf.WriteString(key + "=" + url.QueryEscape(value))
	}

	resp, err = http.Post(XGPUSH_V2_BASE_URL_WITH_SCHEMA+msg.Method, XGPUSH_POST_CONTENT_TYPE, buf)
	return
}

func (xgpush *XGPush) PushMessage(msg *XGPushMsg) {
	// Todo: Queue full check
	xgpush.queue <- msg
}

// --------------------------------------------
// push/single_account
func (xgpush *XGPush) PushToSingleAccount(device_type int, account, message, message_type string) {
	params := make(map[string]string)
	params["account"] = account
	params["message_type"] = message_type
	params["message"] = message
	if device_type == XGPushDeviceType_IOS {
		params["environment"] = Environment
	}
	xgpush.PushMessage(&XGPushMsg{
		Method: XGPUSH_PUSH_SINGLE_ACCOUNT_METHOD,
		Params: params,
	})
}
func (xgpush *XGPush) PushNotificationToSingleAccount(device_type int, account, message string) {
	xgpush.PushToSingleAccount(device_type, account, message, XGPushMessageType_Notification)
}
func (xgpush *XGPush) PushNotificationToSingleIOSAccount(account, message string) {
	xgpush.PushToSingleAccount(XGPushDeviceType_IOS, account, message, XGPushMessageType_Notification)
}

// --------------------------------------------
// push/single_device
func (xgpush *XGPush) PushToSingleDevice(device_type int, device_token, message, message_type string) {
	params := make(map[string]string)
	params["device_token"] = device_token
	params["message_type"] = message_type
	params["message"] = message
	if device_type == XGPushDeviceType_IOS {
		params["environment"] = Environment
	}
	xgpush.PushMessage(&XGPushMsg{
		Method: XGPUSH_PUSH_SINGLE_DEVICE_METHOD,
		Params: params,
	})
}
func (xgpush *XGPush) PushNotificationToSingleDevice(device_type int, device_token, message string) {
	xgpush.PushToSingleDevice(device_type, device_token, message, XGPushMessageType_Notification)
}
func (xgpush *XGPush) PushNotificationToSingleIOSDevice(device_token, message string) {
	xgpush.PushToSingleDevice(XGPushDeviceType_IOS, device_token, message, XGPushMessageType_Notification)
}

// --------------------------------------------
// push/account_list
func (xgpush *XGPush) PushToAccountList(device_type int, account_list []string,
	message, message_type string) error {
	params := make(map[string]string)
	jsondata, err := json.Marshal(account_list)
	if err != nil {
		return err
	}
	params["account_list"] = string(jsondata)
	params["message_type"] = message_type
	params["message"] = message
	if device_type == XGPushDeviceType_IOS {
		params["environment"] = Environment
	}
	xgpush.PushMessage(&XGPushMsg{
		Method: XGPUSH_PUSH_ACCOUNT_LIST_METHOD,
		Params: params,
	})
	return nil
}
func (xgpush *XGPush) PushNotificationToAccountList(device_type int, account_list []string,
	message string) error {
	return xgpush.PushToAccountList(device_type, account_list, message, XGPushMessageType_Notification)
}
func (xgpush *XGPush) PushNotificationToIOSAccountList(account_list []string, message string) error {
	return xgpush.PushToAccountList(XGPushDeviceType_IOS, account_list, message, XGPushMessageType_Notification)
}

// --------------------------------------------
// push/all_device
func (xgpush *XGPush) PushToAllDevice(device_type int, message, message_type string) {
	params := make(map[string]string)
	params["message_type"] = message_type
	params["message"] = message
	if device_type == XGPushDeviceType_IOS {
		params["environment"] = Environment
	}
	xgpush.PushMessage(&XGPushMsg{
		Method: XGPUSH_PUSH_ALL_DEVICE_METHOD,
		Params: params,
	})
}
func (xgpush *XGPush) PushNotificationToAllDevice(device_type int, message string) {
	xgpush.PushToAllDevice(device_type, message, XGPushMessageType_Notification)
}
func (xgpush *XGPush) PushNotificationToAllIOSDevice(message string) {
	xgpush.PushNotificationToAllDevice(XGPushDeviceType_IOS, message)
}

func (xgpush *XGPush) PushToAllDeviceWithLoop(device_type int, message, message_type string,
	loop_times, loop_interval uint) {
	params := make(map[string]string)
	params["message_type"] = message_type
	params["message"] = message
	params["loop_times"] = fmt.Sprintf("%d", loop_times)
	params["loop_interval"] = fmt.Sprintf("%d", loop_interval)
	if device_type == XGPushDeviceType_IOS {
		params["environment"] = Environment
	}
	xgpush.PushMessage(&XGPushMsg{
		Method: XGPUSH_PUSH_ALL_DEVICE_METHOD,
		Params: params,
	})
}
func (xgpush *XGPush) PushNotificationToAllDeviceWithLoop(device_type int, message string,
	loop_times, loop_interval uint) {
	xgpush.PushToAllDeviceWithLoop(device_type, message, XGPushMessageType_Notification, loop_times, loop_interval)
}
func (xgpush *XGPush) PushNotificationToAllIOSDeviceWithLoop(message string,
	loop_times, loop_interval uint) {
	xgpush.PushNotificationToAllDeviceWithLoop(XGPushDeviceType_IOS, message, loop_times, loop_interval)
}

// --------------------------------------------
// application/get_app_device_num
func (xgpush *XGPush) GetAppDeviceNum() (num int, err error) {
	httpresp, err := xgpush.Post(&XGPushMsg{
		Method: XGPUSH_GET_APP_DEVICE_NUM_METHOD,
		Params: make(map[string]string),
	})
	if err != nil {
		log.Println("GetAppDeviceNum err", err.Error())
		return
	}
	defer httpresp.Body.Close()
	var resp GetAppDeviceNumResponse
	jsondec := json.NewDecoder(httpresp.Body)
	err = jsondec.Decode(&resp)
	if err != nil {
		return
	}
	fmt.Printf("GetAppDeviceNum resp: %+v\n", resp)
	if resp.RetCode != 0 {
		return 0, errors.New(fmt.Sprintf("response %d - %s", resp.RetCode, resp.ErrMsg))
	}
	num = resp.Result.DeviceNum
	return
}

// ========================================
// XGPushConn
func NewXGPushConn(xgpush *XGPush, queue <-chan *XGPushMsg) (conn *XGPushConn) {
	conn = &XGPushConn{
		xgpush: xgpush,
		queue:  xgpush.queue,
	}
	go conn.run()
	return
}

func (conn *XGPushConn) run() {
	for {
		msg := <-conn.queue
		// Send msg
		resp, err := conn.xgpush.Post(msg)
		if err != nil {
			log.Println("post err", err.Error())
			continue
		}
		buf := new(bytes.Buffer)
		_, err = buf.ReadFrom(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Printf("read response err: %s\n", err.Error())
			continue
		}
		log.Printf("Response: %s\n", string(buf.Bytes()))
		jsondec := json.NewDecoder(buf)
		var pushResp PushResponse
		err = jsondec.Decode(&pushResp)
		if err != nil {
			log.Printf("Decode push response err: %s\n", err.Error())
			continue
		}
		if pushResp.RetCode != 0 {
			log.Printf("Push response: %d - %s\n", pushResp.RetCode, pushResp.ErrMsg)
			continue
		}
		// Todo: push_id check
	}
}
