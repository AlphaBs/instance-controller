package api

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gin-gonic/gin"
)

type EC2Client interface {
	DescribeInstances(context.Context, *ec2.DescribeInstancesInput, ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
	StartInstances(context.Context, *ec2.StartInstancesInput, ...func(*ec2.Options)) (*ec2.StartInstancesOutput, error)
	StopInstances(context.Context, *ec2.StopInstancesInput, ...func(*ec2.Options)) (*ec2.StopInstancesOutput, error)
}

type Handler struct {
	client     EC2Client
	instanceID string
}

type InstanceResponse struct {
	InstanceID string   `json:"instance_id" example:"i-0123456789abcdef0"`
	State      string   `json:"state" example:"running"`
	PublicIPv4 *string  `json:"public_ipv4" example:"203.0.113.10"`
	PublicIPv6 []string `json:"public_ipv6"`
}

type ChangeStateRequest struct {
	Action string `json:"action" binding:"required,oneof=start stop" example:"start"`
}

type ChangeStateResponse struct {
	InstanceID   string `json:"instance_id" example:"i-0123456789abcdef0"`
	Action       string `json:"action" example:"start"`
	CurrentState string `json:"current_state" example:"stopped"`
	TargetState  string `json:"target_state" example:"pending"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"internal server error"`
}

func NewHandler(client EC2Client, instanceID string) *Handler {
	return &Handler{client: client, instanceID: instanceID}
}

// GetInstance godoc
// @Summary Get the configured EC2 instance state and public IP addresses
// @Tags instance
// @Produce json
// @Security BasicAuth
// @Success 200 {object} InstanceResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 502 {object} ErrorResponse
// @Router /instance [get]
func (h *Handler) GetInstance(c *gin.Context) {
	output, err := h.client.DescribeInstances(c.Request.Context(), &ec2.DescribeInstancesInput{
		InstanceIds: []string{h.instanceID},
	})
	if err != nil {
		slog.Error("failed to describe EC2 instance", "instance_id", h.instanceID, "error", err)
		c.JSON(http.StatusBadGateway, ErrorResponse{Error: "failed to query EC2 instance"})
		return
	}

	instance, ok := firstInstance(output)
	if !ok {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "EC2 instance not found"})
		return
	}

	response := InstanceResponse{
		InstanceID: h.instanceID,
		State:      stateName(instance.State),
		PublicIPv4: instance.PublicIpAddress,
		PublicIPv6: publicIPv6Addresses(instance),
	}
	if response.PublicIPv6 == nil {
		response.PublicIPv6 = []string{}
	}
	c.JSON(http.StatusOK, response)
}

// ChangeInstanceState godoc
// @Summary Start or stop the configured EC2 instance
// @Tags instance
// @Accept json
// @Produce json
// @Security BasicAuth
// @Param request body ChangeStateRequest true "Requested state transition"
// @Success 202 {object} ChangeStateResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 502 {object} ErrorResponse
// @Router /instance/state [post]
func (h *Handler) ChangeInstanceState(c *gin.Context) {
	var request ChangeStateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "action must be either start or stop"})
		return
	}
	request.Action = strings.ToLower(request.Action)

	response := ChangeStateResponse{InstanceID: h.instanceID, Action: request.Action}
	var err error
	if request.Action == "start" {
		var output *ec2.StartInstancesOutput
		output, err = h.client.StartInstances(c.Request.Context(), &ec2.StartInstancesInput{InstanceIds: []string{h.instanceID}})
		if err == nil && output != nil && len(output.StartingInstances) > 0 {
			response.CurrentState = stateName(output.StartingInstances[0].PreviousState)
			response.TargetState = stateName(output.StartingInstances[0].CurrentState)
		}
	} else {
		var output *ec2.StopInstancesOutput
		output, err = h.client.StopInstances(c.Request.Context(), &ec2.StopInstancesInput{InstanceIds: []string{h.instanceID}})
		if err == nil && output != nil && len(output.StoppingInstances) > 0 {
			response.CurrentState = stateName(output.StoppingInstances[0].PreviousState)
			response.TargetState = stateName(output.StoppingInstances[0].CurrentState)
		}
	}
	if err != nil {
		slog.Error("failed to change EC2 instance state", "instance_id", h.instanceID, "action", request.Action, "error", err)
		c.JSON(http.StatusBadGateway, ErrorResponse{Error: "failed to change EC2 instance state"})
		return
	}

	c.JSON(http.StatusAccepted, response)
}

func firstInstance(output *ec2.DescribeInstancesOutput) (types.Instance, bool) {
	if output == nil {
		return types.Instance{}, false
	}
	for _, reservation := range output.Reservations {
		if len(reservation.Instances) > 0 {
			return reservation.Instances[0], true
		}
	}
	return types.Instance{}, false
}

func publicIPv6Addresses(instance types.Instance) []string {
	var addresses []string
	for _, networkInterface := range instance.NetworkInterfaces {
		for _, address := range networkInterface.Ipv6Addresses {
			if address.Ipv6Address != nil && *address.Ipv6Address != "" {
				addresses = append(addresses, *address.Ipv6Address)
			}
		}
	}
	return addresses
}

func stateName(state *types.InstanceState) string {
	if state == nil {
		return ""
	}
	return string(state.Name)
}
