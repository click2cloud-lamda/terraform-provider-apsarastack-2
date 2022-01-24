package hbase

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

// GetMultimodeCmsUrl invokes the hbase.GetMultimodeCmsUrl API synchronously
func (client *Client) GetMultimodeCmsUrl(request *GetMultimodeCmsUrlRequest) (response *GetMultimodeCmsUrlResponse, err error) {
	response = CreateGetMultimodeCmsUrlResponse()
	err = client.DoAction(request, response)
	return
}

// GetMultimodeCmsUrlWithChan invokes the hbase.GetMultimodeCmsUrl API asynchronously
func (client *Client) GetMultimodeCmsUrlWithChan(request *GetMultimodeCmsUrlRequest) (<-chan *GetMultimodeCmsUrlResponse, <-chan error) {
	responseChan := make(chan *GetMultimodeCmsUrlResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.GetMultimodeCmsUrl(request)
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

// GetMultimodeCmsUrlWithCallback invokes the hbase.GetMultimodeCmsUrl API asynchronously
func (client *Client) GetMultimodeCmsUrlWithCallback(request *GetMultimodeCmsUrlRequest, callback func(response *GetMultimodeCmsUrlResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *GetMultimodeCmsUrlResponse
		var err error
		defer close(result)
		response, err = client.GetMultimodeCmsUrl(request)
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

// GetMultimodeCmsUrlRequest is the request struct for api GetMultimodeCmsUrl
type GetMultimodeCmsUrlRequest struct {
	*requests.RpcRequest
	ClusterId string `position:"Query" name:"ClusterId"`
}

// GetMultimodeCmsUrlResponse is the response struct for api GetMultimodeCmsUrl
type GetMultimodeCmsUrlResponse struct {
	*responses.BaseResponse
	RequestId      string `json:"RequestId" xml:"RequestId"`
	ClusterId      string `json:"ClusterId" xml:"ClusterId"`
	MultimodCmsUrl string `json:"MultimodCmsUrl" xml:"MultimodCmsUrl"`
}

// CreateGetMultimodeCmsUrlRequest creates a request to invoke GetMultimodeCmsUrl API
func CreateGetMultimodeCmsUrlRequest() (request *GetMultimodeCmsUrlRequest) {
	request = &GetMultimodeCmsUrlRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("HBase", "2019-01-01", "GetMultimodeCmsUrl", "hbase", "openAPI")
	request.Method = requests.POST
	return
}

// CreateGetMultimodeCmsUrlResponse creates a response to parse from GetMultimodeCmsUrl response
func CreateGetMultimodeCmsUrlResponse() (response *GetMultimodeCmsUrlResponse) {
	response = &GetMultimodeCmsUrlResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
