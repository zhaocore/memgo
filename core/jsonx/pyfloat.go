// Package jsonx: Python 兼容 JSON 序列化助手。
package jsonx

import (
	"math"
	"strconv"
)

// PyFloat 复刻 Python float 的 JSON 文本形状 (整数值也带 ".0", 如 0.0 / 2.0)。
// 契约基线按字节归一化比较, jq 保留小数点痕迹, 故 Go 侧必须同形输出。
type PyFloat float64

// MarshalJSON 实现 Python repr 规则: 最短往返表示, 无小数点/指数时补 ".0"。
func (f PyFloat) MarshalJSON() ([]byte, error) {
	v := float64(f)
	if math.IsInf(v, 0) || math.IsNaN(v) {
		// Python json 默认允许 Infinity/NaN 字面量
		if math.IsInf(v, 1) {
			return []byte("Infinity"), nil
		}
		if math.IsInf(v, -1) {
			return []byte("-Infinity"), nil
		}
		return []byte("NaN"), nil
	}
	s := strconv.FormatFloat(v, 'g', -1, 64)
	// Python repr: 整数值浮点带 ".0"; 指数形态 'e+05' 保持
	if !containsAny(s, ".eE") {
		s += ".0"
	}
	return []byte(s), nil
}

// Repr 返回 Python str(float) 文本 (错误文案用, 如 "Invalid threshold: 2.0")。
func Repr(v float64) string {
	b, _ := PyFloat(v).MarshalJSON()
	return string(b)
}

func containsAny(s, chars string) bool {
	for _, c := range s {
		for _, k := range chars {
			if c == k {
				return true
			}
		}
	}
	return false
}
