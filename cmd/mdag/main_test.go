package main

import (
	"strings"
	"testing"
)

// 测试缺少命令参数时会返回中文用法提示。
func TestRunUsageIsChinese(t *testing.T) {
	err := run(nil)
	if err == nil {
		t.Fatal("缺少参数时应该返回用法错误")
	}
	if !strings.Contains(err.Error(), "用法:") {
		t.Fatalf("应该返回中文用法提示，实际为 %q", err.Error())
	}
}
