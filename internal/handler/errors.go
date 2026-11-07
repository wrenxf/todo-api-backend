package handler

import "fmt"

func errPage() error {
	return fmt.Errorf("page 必须为正整数")
}

func errPageSize() error {
	return fmt.Errorf("page_size 取值范围为 1-100")
}

func errStatus() error {
	return fmt.Errorf("status 只能为 pending/completed/cancelled")
}

func errPriority() error {
	return fmt.Errorf("priority 只能为 low/medium/high")
}
