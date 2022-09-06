package util

import (
	"math/rand"
	"time"
)

/**
* size 随机码的位数
* kind 0    // 纯数字
       1    // 小写字母
       2    // 大写字母
       3    // 数字、大小写字母
*/
func RandomString(size int, kind int) string {
	ikind, kinds, rsbytes := kind, [][]int{{10, 48}, {26, 97}, {26, 65}}, make([]byte, size)
	isAll := kind > 2 || kind < 0
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < size; i++ {
		if isAll { // random ikind
			ikind = rand.Intn(3)
		}
		scope, base := kinds[ikind][0], kinds[ikind][1]
		rsbytes[i] = uint8(base + rand.Intn(scope))
	}
	return string(rsbytes)
}
