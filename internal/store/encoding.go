package store

import (
	"encoding/json"
	"fmt"

	"benzhi-project-39162b49-a874-41ad-95d1-e3d76a40af9a/internal/domain"
)

func encodePlan(plan domain.RiggingPlan) ([]byte, error) {
	data, err := json.Marshal(plan)
	if err != nil {
		return nil, fmt.Errorf("序列化吊挂方案：%w", err)
	}
	return data, nil
}

func decodePlan(data []byte) (domain.RiggingPlan, error) {
	var plan domain.RiggingPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return domain.RiggingPlan{}, fmt.Errorf("解析吊挂方案：%w", err)
	}
	return plan, nil
}

func encodeValue(label string, value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("序列化%s：%w", label, err)
	}
	return data, nil
}
