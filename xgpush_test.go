package xgpush

import (
	"fmt"
	"testing"
	"time"
)

func TestSign(t *testing.T) {
	fmt.Println(``)
	fmt.Println(`//////////////////// TestSign \\\\\\\\\\\\\\\\\\\\`)
	parameters := XGPushParameters{
		Param_ios_access_id:      "123",
		Param_android_access_id:  "123",
		Param_ios_secret_key:     "abcde",
		Param_android_secret_key: "abcde",
		Param_connections:        5,
		Param_queue_size:         10000,
		Param_timeout:            time.Second,
	}
	xgpush := NewXGPush(&parameters)
	params := make(map[string]string)
	params["Param1"] = "Value1"
	params["Param2"] = "Value2"
	params["timestamp"] = "1386691200"
	xgpush.sign("push/single_device", XGPushDeviceType_Android, params)
	if params["sign"] != "ccafecaef6be07493cfe75ebc43b7d53" {
		t.Error("Sign error")
	}
	fmt.Println(`\\\\\\\\\\\\\\\\\\\\ TestSign ////////////////////`)
	fmt.Println(``)
}

func TestXGPush(t *testing.T) {
	fmt.Println(``)
	fmt.Println(`//////////////////// TestXGPush \\\\\\\\\\\\\\\\\\\\`)
	//parameters := XGPushParameters{
	//	Param_ios_access_id:      "<Your access_id>",
	//	Param_android_access_id:  "<Your access_id>",
	//	Param_ios_secret_key:     "<Your secret_key>",
	//	Param_android_secret_key: "<Your secret_key>",
	//	Param_connections:        5,
	//	Param_queue_size:         10000,
	//	Param_timeout:            time.Second,
	//	Param_environment:        XGPushEnviroment_Develop,
	//}
	//xgpush := NewXGPush(&parameters)
	//xgpush.PushNotificationToSingleIOSAccount("test1", `{"aps":{"alert":"to single ios account"}}`)
	//time.Sleep(time.Second)
	//xgpush.PushNotificationToSingleIOSDevice("c59431b77469bf9ad01ef54199f14f333fdce0b34db634cf1bdf8f19daf6082b",
	//	`{"aps":{"alert":"to single ios device"}}`)
	//time.Sleep(time.Second)
	//xgpush.PushNotificationToIOSAccountList([]string{"test1", "test2"}, `{"aps":{"alert":"to ios account list"}}`)
	//time.Sleep(time.Second)
	//xgpush.PushNotificationToAllIOSDevice(`{"aps":{"alert":"To all ios device"}}`)
	//time.Sleep(time.Second)
	//deviceNum, err := xgpush.GetAppDeviceNum()
	//if err != nil {
	//	t.Error("GetAppDeviceNum err:", err.Error())
	//}
	//fmt.Printf("device num: %d\n", deviceNum)
	fmt.Println(`\\\\\\\\\\\\\\\\\\\\ TestXGPush ////////////////////`)
	fmt.Println(``)
}
