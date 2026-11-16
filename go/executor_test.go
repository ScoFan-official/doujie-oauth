package main

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginabi"
)

func TestConcurrencyGateAppliesConfigUpdates(t *testing.T) {
	oldConfig := getConfig()
	gateMu.Lock()
	oldGates := gates
	gates = map[string]*concurrencyGate{}
	gateMu.Unlock()
	defer func() {
		configMu.Lock()
		currentConfig = oldConfig
		configMu.Unlock()
		gateMu.Lock()
		gates = oldGates
		gateMu.Unlock()
	}()

	if err := configure([]byte("max_concurrency: 1\n")); err != nil {
		t.Fatal(err)
	}
	releaseFirst := acquire("dynamic")
	waiter := make(chan func(), 1)
	go func() { waiter <- acquire("dynamic") }()
	select {
	case release := <-waiter:
		release()
		t.Fatal("second request acquired above the configured limit")
	case <-time.After(100 * time.Millisecond):
	}

	if err := configure([]byte("max_concurrency: 2\n")); err != nil {
		t.Fatal(err)
	}
	select {
	case release := <-waiter:
		release()
	case <-time.After(time.Second):
		t.Fatal("waiting request did not observe the increased limit")
	}
	releaseFirst()
}

func TestModelUnavailableErrorClassification(t *testing.T) {
	for _, message := range []string{
		"model not available for this account",
		"unsupported model",
		"该模型未开通",
	} {
		err := upstreamError(http.StatusBadRequest, []byte(`{"message":"`+message+`"}`))
		typed, ok := err.(*pluginabi.Error)
		if !ok || typed.Code != "model_unavailable" || typed.HTTPStatus != http.StatusBadRequest || typed.Retryable {
			t.Fatalf("unexpected model error for %q: %#v", message, err)
		}
	}
	err := upstreamFailureError("model_not_found")
	typed, ok := err.(*pluginabi.Error)
	if !ok || typed.Code != "model_unavailable" {
		t.Fatalf("stream model error not classified: %#v", err)
	}
}

func TestQuotaFailureClassification(t *testing.T) {
	for _, message := range []string{
		"insufficient_quota",
		"usage limit reached",
		"Too Many Requests",
		"当前账号额度已用完",
		"请求次数已达上限，请稍后再试",
	} {
		err := upstreamFailureError(message)
		var pluginErr *pluginabi.Error
		if !errors.As(err, &pluginErr) || pluginErr.StatusCode() != http.StatusTooManyRequests {
			t.Fatalf("quota message %q mapped to %#v (%v)", message, pluginErr, err)
		}
	}

	err := upstreamFailureError("temporary upstream failure")
	var pluginErr *pluginabi.Error
	if !errors.As(err, &pluginErr) || pluginErr.StatusCode() != http.StatusBadGateway {
		t.Fatalf("non-quota failure mapped to %#v (%v)", pluginErr, err)
	}
}

func TestUpstreamErrorDetectsQuotaBodyOn500(t *testing.T) {
	err := upstreamError(http.StatusInternalServerError, []byte(`{"message":"本月额度已耗尽"}`))
	var pluginErr *pluginabi.Error
	if !errors.As(err, &pluginErr) || pluginErr.StatusCode() != http.StatusTooManyRequests {
		t.Fatalf("quota body mapped to %#v (%v)", pluginErr, err)
	}
}
