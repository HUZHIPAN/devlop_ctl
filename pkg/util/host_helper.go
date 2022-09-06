package util

import (
	"fmt"
	"net"
	"os"
)

func GetLocalMac() (mac string) {
	// 获取本机的MAC地址
	interfaces, err := net.Interfaces()
	if err != nil {
		panic("Poor soul, here is what you got: " + err.Error())
	}
	for _, inter := range interfaces {
		fmt.Println(inter.Name)
		mac := inter.HardwareAddr //获取本机MAC地址
		fmt.Println("MAC = ", mac)
	}
	return mac
}

// 获取当前程序执行者uid
func GetCurrentRunUID() string {
	return fmt.Sprintf("%v", os.Getuid())
}
