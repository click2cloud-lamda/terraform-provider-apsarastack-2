package adb

//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.
//
// Code generated by Alibaba Cloud SDK Code Generator.
// Changes may cause incorrect behavior and will be lost if the code is regenerated.

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
)

// DescribeElasticDailyPlan invokes the adb.DescribeElasticDailyPlan API synchronously
func (client *Client) DescribeElasticDailyPlan(request *DescribeElasticDailyPlanRequest) (response *DescribeElasticDailyPlanResponse, err error) {
	response = CreateDescribeElasticDailyPlanResponse()
	err = client.DoAction(request, response)
	return
}

// DescribeElasticDailyPlanWithChan invokes the adb.DescribeElasticDailyPlan API asynchronously
func (client *Client) DescribeElasticDailyPlanWithChan(request *DescribeElasticDailyPlanRequest) (<-chan *DescribeElasticDailyPlanResponse, <-chan error) {
	responseChan := make(chan *DescribeElasticDailyPlanResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.DescribeElasticDailyPlan(request)
		if err != nil {
			errChan <- err
		} else {
			responseChan <- response
		}
	})
	if err != nil {
		errChan <- err
		close(responseChan)
		close(errChan)
	}
	return responseChan, errChan
}

// DescribeElasticDailyPlanWithCallback invokes the adb.DescribeElasticDailyPlan API asynchronously
func (client *Client) DescribeElasticDailyPlanWithCallback(request *DescribeElasticDailyPlanRequest, callback func(response *DescribeElasticDailyPlanResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *DescribeElasticDailyPlanResponse
		var err error
		defer close(result)
		response, err = client.DescribeElasticDailyPlan(request)
		callback(response, err)
		result <- 1
	})
	if err != nil {
		defer close(result)
		callback(nil, err)
		result <- 0
	}
	return result
}

// DescribeElasticDailyPlanRequest is the request struct for api DescribeElasticDailyPlan
type DescribeElasticDailyPlanRequest struct {
	*requests.RpcRequest
	ResourceOwnerId            requests.Integer `position:"Query" name:"ResourceOwnerId"`
	ElasticDailyPlanStatusList string           `position:"Query" name:"ElasticDailyPlanStatusList"`
	ElasticDailyPlanDay        string           `position:"Query" name:"ElasticDailyPlanDay"`
	ResourceOwnerAccount       string           `position:"Query" name:"ResourceOwnerAccount"`
	DBClusterId                string           `position:"Query" name:"DBClusterId"`
	OwnerAccount               string           `position:"Query" name:"OwnerAccount"`
	OwnerId                    requests.Integer `position:"Query" name:"OwnerId"`
	ElasticPlanName            string           `position:"Query" name:"ElasticPlanName"`
	ResourcePoolName           string           `position:"Query" name:"ResourcePoolName"`
}

// DescribeElasticDailyPlanResponse is the response struct for api DescribeElasticDailyPlan
type DescribeElasticDailyPlanResponse struct {
	*responses.BaseResponse
	RequestId            string                 `json:"RequestId" xml:"RequestId"`
	ElasticDailyPlanList []ElasticDailyPlanInfo `json:"ElasticDailyPlanList" xml:"ElasticDailyPlanList"`
}

// CreateDescribeElasticDailyPlanRequest creates a request to invoke DescribeElasticDailyPlan API
func CreateDescribeElasticDailyPlanRequest() (request *DescribeElasticDailyPlanRequest) {
	request = &DescribeElasticDailyPlanRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("adb", "2019-03-15", "DescribeElasticDailyPlan", "ads", "openAPI")
	request.Method = requests.POST
	return
}

// CreateDescribeElasticDailyPlanResponse creates a response to parse from DescribeElasticDailyPlan response
func CreateDescribeElasticDailyPlanResponse() (response *DescribeElasticDailyPlanResponse) {
	response = &DescribeElasticDailyPlanResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
