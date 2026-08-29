package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/ksi/instance-controller/internal/config"
)

type mockEC2Client struct {
	describeOutput *ec2.DescribeInstancesOutput
	startOutput    *ec2.StartInstancesOutput
	stopOutput     *ec2.StopInstancesOutput
	describeCalls  int
	startCalls     int
	stopCalls      int
}

func (m *mockEC2Client) DescribeInstances(context.Context, *ec2.DescribeInstancesInput, ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	m.describeCalls++
	return m.describeOutput, nil
}

func (m *mockEC2Client) StartInstances(context.Context, *ec2.StartInstancesInput, ...func(*ec2.Options)) (*ec2.StartInstancesOutput, error) {
	m.startCalls++
	return m.startOutput, nil
}

func (m *mockEC2Client) StopInstances(context.Context, *ec2.StopInstancesInput, ...func(*ec2.Options)) (*ec2.StopInstancesOutput, error) {
	m.stopCalls++
	return m.stopOutput, nil
}

func testRouter(client EC2Client) http.Handler {
	return NewRouter(client, config.Config{
		EC2InstanceID:     "i-test",
		BasicAuthUser:     "user",
		BasicAuthPassword: "secret",
	})
}

func TestHealthCheckDoesNotRequireAuthentication(t *testing.T) {
	recorder := httptest.NewRecorder()
	testRouter(&mockEC2Client{}).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestAPIRequiresBasicAuthentication(t *testing.T) {
	recorder := httptest.NewRecorder()
	client := &mockEC2Client{}
	testRouter(client).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/instance", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	if client.describeCalls != 0 {
		t.Fatalf("DescribeInstances called %d times, want 0", client.describeCalls)
	}
}

func TestGetInstance(t *testing.T) {
	client := &mockEC2Client{describeOutput: &ec2.DescribeInstancesOutput{
		Reservations: []types.Reservation{{Instances: []types.Instance{{
			InstanceId:      aws.String("i-test"),
			State:           &types.InstanceState{Name: types.InstanceStateNameRunning},
			PublicIpAddress: aws.String("203.0.113.10"),
			NetworkInterfaces: []types.InstanceNetworkInterface{{
				Ipv6Addresses: []types.InstanceIpv6Address{{Ipv6Address: aws.String("2001:db8::10")}},
			}},
		}}}},
	}}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/instance", nil)
	request.SetBasicAuth("user", "secret")
	testRouter(client).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	var response InstanceResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.State != "running" || response.PublicIPv4 == nil || *response.PublicIPv4 != "203.0.113.10" {
		t.Fatalf("unexpected response: %+v", response)
	}
	if len(response.PublicIPv6) != 1 || response.PublicIPv6[0] != "2001:db8::10" {
		t.Fatalf("unexpected IPv6 addresses: %v", response.PublicIPv6)
	}
}

func TestChangeInstanceState(t *testing.T) {
	client := &mockEC2Client{startOutput: &ec2.StartInstancesOutput{
		StartingInstances: []types.InstanceStateChange{{
			PreviousState: &types.InstanceState{Name: types.InstanceStateNameStopped},
			CurrentState:  &types.InstanceState{Name: types.InstanceStateNamePending},
		}},
	}}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/instance/state", strings.NewReader(`{"action":"start"}`))
	request.Header.Set("Content-Type", "application/json")
	request.SetBasicAuth("user", "secret")
	testRouter(client).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusAccepted {
		body, _ := io.ReadAll(recorder.Body)
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusAccepted, body)
	}
	if client.startCalls != 1 || client.stopCalls != 0 {
		t.Fatalf("start calls = %d, stop calls = %d", client.startCalls, client.stopCalls)
	}
}

func TestChangeInstanceStateRejectsInvalidAction(t *testing.T) {
	client := &mockEC2Client{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/instance/state", strings.NewReader(`{"action":"reboot"}`))
	request.Header.Set("Content-Type", "application/json")
	request.SetBasicAuth("user", "secret")
	testRouter(client).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if client.startCalls != 0 || client.stopCalls != 0 {
		t.Fatal("EC2 state API must not be called for an invalid action")
	}
}
