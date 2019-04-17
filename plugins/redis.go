package plugins

import (
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	redis "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/redis/v20180412"
)

var RedisActions = make(map[string]Action)

func init() {
	RedisActions["create"] = new(RedisCreateAction)
	RedisActions["terminate"] = new(RedisTerminateAction)
}

func CreateRedisClient(region, secretId, secretKey string) (client *redis.Client, err error) {
	credential := common.NewCredential(secretId, secretKey)

	clientProfile := profile.NewClientProfile()
	clientProfile.HttpProfile.Endpoint = "redis.tencentcloudapi.com"

	return redis.NewClient(credential, region, clientProfile)
}

type RedisInputs struct {
	Inputs []RedisInput `json:"inputs,omitempty"`
}

type RedisInput struct {
	Guid           string `json:"guid,omitempty"`
	ProviderParams string `json:"provider_params,omitempty"`
	ZoneID         int    `json:"zone_id,omitempty"`
	TypeID         int    `json:"type_id,omitempty"`
	MemSize        int    `json:"mem_size,omitempty"`
	GoodsNum       int    `json:"goods_num,omitempty"`
	Period         int    `json:"period,omitempty"`
	Password       string `json:"password,omitempty"`
	BillingMode    int    `json:"billing_mode,omitempty"`
	VpcID          string `json:"vpc_id,omitempty"`
	SubnetID       string `json:"subnet_id,omitempty"`
	InstanceID     string `json:"instance_id,omitempty"`
}

type RedisOutputs struct {
	Outputs []RedisOutput `json:"outputs,omitempty"`
}

type RedisOutput struct {
	RequestId string `json:"request_id,omitempty"`
	Guid      string `json:"guid,omitempty"`
	DealID    string `json:"deal_id,omitempty"`
	TaskID    int    `json:"task_id,omitempty"`
}

type RedisPlugin struct {
}

func (plugin *RedisPlugin) GetActionByName(actionName string) (Action, error) {
	action, found := RedisActions[actionName]

	if !found {
		return nil, fmt.Errorf("Redis plugin,action = %s not found", actionName)
	}

	return action, nil
}

type RedisCreateAction struct {
}

func (action *RedisCreateAction) ReadParam(param interface{}) (interface{}, error) {
	var inputs RedisInputs
	err := UnmarshalJson(param, &inputs)
	if err != nil {
		return nil, err
	}
	return inputs, nil
}

func (action *RedisCreateAction) CheckParam(input interface{}) error {
	rediss, ok := input.(RedisInputs)
	if !ok {
		return fmt.Errorf("RedisCreateAction:input type=%T not right", input)
	}

	for _, redis := range rediss.Inputs {
		if redis.GoodsNum == 0 {
			return errors.New("RedisCreateAction input goodsnum is invalid")
		}
		if redis.Password == "" {
			return errors.New("RedisCreateAction input password is empty")
		}
		if redis.BillingMode != 0 && redis.BillingMode != 1 {
			return errors.New("RedisCreateAction input password is invalid")
		}
	}

	return nil
}

func (action *RedisCreateAction) createRedis(redisInput *RedisInput) (*RedisOutput, error) {
	paramsMap, err := GetMapFromProviderParams(redisInput.ProviderParams)
	client, _ := CreateRedisClient(paramsMap["Region"], paramsMap["SecretID"], paramsMap["SecretKey"])

	request := redis.NewCreateInstancesRequest()
	request.ZoneId = &redisInput.ZoneID
	request.TypeId = &redisInput.TypeID
	request.MemSize = &redisInput.MemSize
	request.GoodsNum = &redisInput.GoodsNum
	request.Period = &redisInput.Period
	request.Password = &redisInput.Password
	request.BillingMode = &redisInput.BillingMode

	if (*redisInput).VpcID != "" {
		request.VpcId = &redisInput.VpcID
	}

	if (*redisInput).SubnetId != "" {
		request.SubnetId = &redisInput.SubnetID
	}

	response, err := client.CreateInstances(request)
	if err != nil {
		logrus.Errorf("failed to create redis, error=%s", err)
		return nil, err
	}

	output := RedisOutput{}
	output.RequestId = *response.Response.RequestId
	output.Guid = redisInput.Guid
	output.DealId = *response.Response.DealId

	return &output, nil
}

func (action *RedisCreateAction) Do(input interface{}) (interface{}, error) {
	rediss, _ := input.(RedisInputs)
	outputs := RedisOutputs{}
	for _, redis := range rediss.Inputs {
		redisOutput, err := action.createRedis(&redis)
		if err != nil {
			return nil, err
		}
		outputs.Outputs = append(outputs.Outputs, *redisOutput)
	}

	logrus.Infof("all rediss = %v are created", rediss)
	return &outputs, nil
}

type RedisTerminateAction struct {
}

func (action *RedisTerminateAction) ReadParam(param interface{}) (interface{}, error) {
	var inputs RedisInputs
	err := UnmarshalJson(param, &inputs)
	if err != nil {
		return nil, err
	}
	return inputs, nil
}

func (action *RedisTerminateAction) CheckParam(input interface{}) error {
	rediss, ok := input.(RedisInputs)
	if !ok {
		return fmt.Errorf("redisTerminateAtion:input type=%T not right", input)
	}

	for _, redis := range rediss.Inputs {
		if redis.InstanceID == "" {
			return errors.New("RedisTerminateAtion input InstanceID is empty")
		}
		if redis.Password == "" {
			return errors.New("RedisTerminateAtion input Password is empty")
		}
	}
	return nil
}

func (action *RedisTerminateAction) terminateRedis(redisInput *RedisInput) (*RedisOutput, error) {
	paramsMap, err := GetMapFromProviderParams(redisInput.ProviderParams)
	client, _ := CreateRedisClient(paramsMap["Region"], paramsMap["SecretID"], paramsMap["SecretKey"])

	request := redis.NewClearInstanceRequest()
	request.InstanceId = &redisInput.InstanceID
	request.Password = &redisInput.Password

	response, err := client.ClearInstance(request)
	if err != nil {
		return nil, fmt.Errorf("Failed to ClearInstance(InstanceId=%v), error=%s", redisInput.InstanceID, err)
	}
	output := RedisOutput{}
	output.RequestId = *response.Response.RequestId
	output.Guid = redisInput.Guid
	output.TaskID = *response.Response.TaskId

	return &output, nil
}

func (action *RedisTerminateAction) Do(input interface{}) (interface{}, error) {
	rediss, _ := input.(RedisInputs)
	outputs := RedisOutputs{}
	for _, redis := range rediss.Inputs {
		output, err := action.terminateRedis(&redis)
		if err != nil {
			return nil, err
		}
		outputs.Outputs = append(outputs.Outputs, *output)
	}

	return &outputs, nil
}
