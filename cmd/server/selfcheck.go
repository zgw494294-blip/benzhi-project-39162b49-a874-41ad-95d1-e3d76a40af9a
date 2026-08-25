package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"benzhi-project-39162b49-a874-41ad-95d1-e3d76a40af9a/internal/domain"
)

func runSelfCheck(ctx context.Context, configuration config, logger *slog.Logger) error {
	temp, err := os.CreateTemp("", "rigging-selfcheck-*.db")
	if err != nil {
		return fmt.Errorf("创建自检数据库：%w", err)
	}
	database := temp.Name()
	if err := temp.Close(); err != nil {
		return fmt.Errorf("关闭自检数据库文件：%w", err)
	}
	defer os.Remove(database)
	repository, handler, err := buildHandler(ctx, database, logger)
	if err != nil {
		return err
	}
	defer repository.Close()
	listener, err := net.Listen("tcp", configuration.address)
	if err != nil {
		return fmt.Errorf("监听自检地址 %s：%w", configuration.address, err)
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 3 * time.Second}
	serveErrors := make(chan error, 1)
	go func() { serveErrors <- server.Serve(listener) }()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	client := &http.Client{Timeout: 4 * time.Second}
	baseURL := "http://" + configuration.address
	if err := waitUntilHealthy(ctx, client, baseURL); err != nil {
		return err
	}
	plan, err := selfCheckCreate(ctx, client, baseURL)
	if err != nil {
		return err
	}
	plan, err = selfCheckCommand(ctx, client, baseURL, plan.ID, "checks", map[string]any{
		"version": plan.Version, "actor": "自检负责人", "requestKey": "selfcheck-checks",
	})
	if err != nil || plan.State != domain.StateRehearsalReady {
		return fmt.Errorf("校核阶段未进入 REHEARSAL_READY：state=%s error=%w", plan.State, err)
	}
	plan, err = selfCheckCommand(ctx, client, baseURL, plan.ID, "rehearsals", map[string]any{
		"version": plan.Version, "observer": "自检联排监督员", "outcome": "PASSED",
		"observations": "全行程运行稳定，限位与净空复测通过", "evidenceRefs": []string{"SELF-CHECK-VIDEO-001"},
		"requestKey": "selfcheck-rehearsal",
	})
	if err != nil || plan.State != domain.StateReviewPending {
		return fmt.Errorf("联排阶段未进入 REVIEW_PENDING：state=%s error=%w", plan.State, err)
	}
	plan, err = selfCheckCommand(ctx, client, baseURL, plan.ID, "reviews", map[string]any{
		"version": plan.Version, "reviewer": "自检独立评审员", "decision": "APPROVED",
		"comment": "载荷、动作和联排证据完整", "requestKey": "selfcheck-review",
	})
	if err != nil || plan.State != domain.StateAuthorized || plan.AuthorizationCode == "" {
		return fmt.Errorf("评审阶段未生成启用单：state=%s error=%w", plan.State, err)
	}
	var verification struct {
		Valid bool `json:"valid"`
	}
	if err := selfCheckGET(ctx, client, baseURL+"/api/v1/authorizations/"+plan.AuthorizationCode, &verification); err != nil {
		return err
	}
	if !verification.Valid {
		return errors.New("授权码与冻结修订摘要验证失败")
	}
	select {
	case err := <-serveErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("自检 HTTP 服务异常：%w", err)
		}
	default:
	}
	return nil
}

func waitUntilHealthy(ctx context.Context, client *http.Client, baseURL string) error {
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		var health map[string]string
		if err := selfCheckGET(ctx, client, baseURL+"/healthz", &health); err == nil && health["status"] == "ok" {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("等待自检服务就绪：%w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func selfCheckCreate(ctx context.Context, client *http.Client, baseURL string) (domain.RiggingPlan, error) {
	payload := map[string]any{
		"requestKey": "selfcheck-create", "title": "自检飞景吊挂方案", "venue": "回环测试剧场",
		"performanceDate": "2030-10-01", "owner": "自检负责人", "changeReason": "首版安全方案",
		"submittedBy": "自检方案提交者",
		"loadPoints": []map[string]any{
			{"name": "LX-01", "ratedCapacityKg": 1200, "plannedLoadKg": 400, "angleDeg": 5, "safetyFactor": 1.5, "position": "舞台左"},
			{"name": "RX-01", "ratedCapacityKg": 1200, "plannedLoadKg": 420, "angleDeg": 5, "safetyFactor": 1.5, "position": "舞台右"},
		},
		"cues": []map[string]any{
			{"cueNo": 1, "label": "飞景升起", "startOffsetMs": 0, "durationMs": 3000, "movingPoints": []string{"LX-01"}, "clearanceCm": 55, "operator": "甲"},
			{"cueNo": 2, "label": "侧幕收拢", "startOffsetMs": 3500, "durationMs": 2500, "movingPoints": []string{"RX-01"}, "clearanceCm": 48, "operator": "乙"},
		},
	}
	var plan domain.RiggingPlan
	err := selfCheckJSON(ctx, client, http.MethodPost, baseURL+"/api/v1/rigging-plans", payload, &plan)
	return plan, err
}

func selfCheckCommand(ctx context.Context, client *http.Client, baseURL, planID, resource string, payload any) (domain.RiggingPlan, error) {
	var plan domain.RiggingPlan
	err := selfCheckJSON(ctx, client, http.MethodPost, baseURL+"/api/v1/rigging-plans/"+planID+"/"+resource, payload, &plan)
	return plan, err
}

func selfCheckGET(ctx context.Context, client *http.Client, url string, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s 返回 %s", url, response.Status)
	}
	return json.NewDecoder(response.Body).Decode(target)
}

func selfCheckJSON(ctx context.Context, client *http.Client, method, url string, payload, target any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var body any
		_ = json.NewDecoder(response.Body).Decode(&body)
		return fmt.Errorf("%s %s 返回 %s：%v", method, url, response.Status, body)
	}
	return json.NewDecoder(response.Body).Decode(target)
}
