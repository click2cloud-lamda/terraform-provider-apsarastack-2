package connectivity

import (
	"encoding/json"
	rpc "github.com/alibabacloud-go/tea-rpc/client"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/endpoints"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/adb"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/alidns"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/bssopenapi"
	cdn_new "github.com/aliyun/alibaba-cloud-sdk-go/services/cdn"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/cms"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/cr"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/cr_ee"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/dds"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/edas"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/elasticsearch"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ess"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/gpdb"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/hbase"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/location"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/maxcompute"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ons"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/polardb"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/r-kvstore"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/rds"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/slb"
	slsPop "github.com/aliyun/alibaba-cloud-sdk-go/services/sls"
	sls "github.com/aliyun/aliyun-log-go-sdk"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/aliyun/fc-go-sdk"
	"log"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/kms"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ram"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	"github.com/denverdino/aliyungo/cdn"

	"github.com/denverdino/aliyungo/cs"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"sync"

	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type ApsaraStackClient struct {
	SourceIp          string
	SecureTransport   string
	Region            Region
	RegionId          string
	Domain            string
	AccessKey         string
	SecretKey         string
	Department        string
	ResourceGroup     string
	Config            *Config
	teaSdkConfig      rpc.Config
	accountId         string
	roleId            int
	ecsconn           *ecs.Client
	accountIdMutex    sync.RWMutex
	roleIdMutex       sync.RWMutex
	vpcconn           *vpc.Client
	slbconn           *slb.Client
	csconn            *cs.Client
	polarDBconn       *polardb.Client
	cdnconn           *cdn.CdnClient
	cdnconn_new       *cdn_new.Client
	kmsconn           *kms.Client
	bssopenapiconn    *bssopenapi.Client
	rdsconn           *rds.Client
	ramconn           *ram.Client
	essconn           *ess.Client
	gpdbconn          *gpdb.Client
	elasticsearchconn *elasticsearch.Client
	hbaseconn         *hbase.Client
	adbconn           *adb.Client
	ossconn           *oss.Client
	rkvconn           *r_kvstore.Client
	fcconn            *fc.Client
	ddsconn           *dds.Client
	onsconn           *ons.Client
	logconn           *sls.Client
	logpopconn        *slsPop.Client
	dnsconn           *alidns.Client
	edasconn          *edas.Client
	creeconn          *cr_ee.Client
	crconn            *cr.Client
	cmsconn           *cms.Client
	maxcomputeconn    *maxcompute.Client
	//otsconn                      *ots.Client
	OtsInstanceName string
	//tablestoreconnByInstanceName map[string]*tablestore.TableStoreClient
	//dhconn                       datahub.DataHubApi
}

const (
	ApiVersion20140526 = ApiVersion("2014-05-26")
	ApiVersion20160815 = ApiVersion("2016-08-15")
	ApiVersion20140515 = ApiVersion("2014-05-15")
	ApiVersion20190510 = ApiVersion("2019-05-10")
)

const DefaultClientRetryCountSmall = 5

const Terraform = "HashiCorp-Terraform"

const Provider = "Terraform-Provider"

const Module = "Terraform-Module"

type ApiVersion string

// The main version number that is being run at the moment.

var ProviderVersion = "1.0.16"
var TerraformVersion = strings.TrimSuffix(schema.Provider{}.TerraformVersion, "-dev")
var goSdkMutex = sync.RWMutex{} // The Go SDK is not thread-safe
var loadSdkEndpointMutex = sync.Mutex{}

// Client for ApsaraStackClient
func (c *Config) Client() (*ApsaraStackClient, error) {
	// Get the auth and region. This can fail if keys/regions were not
	// specified and we're attempting to use the environment.
	if !c.SkipRegionValidation {
		err := c.loadAndValidate()
		if err != nil {
			return nil, err
		}
	}
	teaSdkConfig, err := c.getTeaDslSdkConfig(true)
	if err != nil {
		return nil, err
	}
	return &ApsaraStackClient{
		Config:        c,
		teaSdkConfig:  teaSdkConfig,
		Region:        c.Region,
		RegionId:      c.RegionId,
		AccessKey:     c.AccessKey,
		SecretKey:     c.SecretKey,
		Department:    c.Department,
		ResourceGroup: c.ResourceGroup,
		Domain:        c.Domain,
	}, nil
}

func (client *ApsaraStackClient) WithEcsClient(do func(*ecs.Client) (interface{}, error)) (interface{}, error) {
	// Initialize the ECS client if necessary
	if client.ecsconn == nil {
		endpoint := client.Config.EcsEndpoint
		if endpoint == "" {
			return nil, fmt.Errorf("unable to initialize the ecs client: endpoint or domain is not provided for ecs service")
		}
		if endpoint != "" {
			endpoints.AddEndpointMapping(client.Config.RegionId, string(ECSCode), endpoint)
		}
		if strings.HasPrefix(endpoint, "http") {
			endpoint = strings.TrimPrefix(strings.TrimPrefix(endpoint, "http://"), "https://")
		}
		ecsconn, err := ecs.NewClientWithOptions(client.Config.RegionId, client.getSdkConfig().WithTimeout(time.Duration(60)*time.Second), client.Config.getAuthCredential(true))
		if err != nil {
			return nil, fmt.Errorf("unable to initialize the ECS client: %#v", err)
		}

		ecsconn.Domain = endpoint
		ecsconn.AppendUserAgent(Terraform, TerraformVersion)
		ecsconn.AppendUserAgent(Provider, ProviderVersion)
		ecsconn.AppendUserAgent(Module, client.Config.ConfigurationSource)
		ecsconn.SetHTTPSInsecure(client.Config.Insecure)
		if client.Config.Proxy != "" {
			ecsconn.SetHttpsProxy(client.Config.Proxy)
			ecsconn.SetHttpProxy(client.Config.Proxy)
		}
		client.ecsconn = ecsconn
	}

	return do(client.ecsconn)
}

func (client *ApsaraStackClient) WithPolarDBClient(do func(*polardb.Client) (interface{}, error)) (interface{}, error) {
	// Initialize the PolarDB client if necessary
	if client.polarDBconn == nil {
		endpoint := client.Config.PolarDBEndpoint
		if endpoint == "" {
			return nil, fmt.Errorf("unable to initialize the polardb client: endpoint or domain is not provided for polardb service")
		}
		polarDBconn, err := polardb.NewClientWithOptions(client.Config.RegionId, client.getSdkConfig(), client.Config.getAuthCredential(true))
		if err != nil {
			return nil, fmt.Errorf("unable to initialize the PolarDB client: %#v", err)

		}
		polarDBconn.Domain = endpoint
		polarDBconn.AppendUserAgent(Terraform, TerraformVersion)
		polarDBconn.AppendUserAgent(Provider, ProviderVersion)
		polarDBconn.AppendUserAgent(Module, client.Config.ConfigurationSource)
		polarDBconn.SetHTTPSInsecure(client.Config.Insecure)
		if client.Config.Proxy != "" {
			polarDBconn.SetHttpProxy(client.Config.Proxy)
			polarDBconn.SetHTTPSInsecure(client.Config.Insecure)
		}

		client.polarDBconn = polarDBconn
	}

	return do(client.polarDBconn)
}
func (client *ApsaraStackClient) WithElasticsearchClient(do func(*elasticsearch.Client) (interface{}, error)) (interface{}, error) {
	// Initialize the Elasticsearch client if necessary
	if client.elasticsearchconn == nil {
		endpoint := client.Config.ElasticsearchEndpoint
		if endpoint == "" {
			return nil, fmt.Errorf("unable to initialize the ElasticSearch client: endpoint or domain is not provided for ElasticSearch service")
		}
		if endpoint != "" {
			endpoints.AddEndpointMapping(client.Config.RegionId, string(ELASTICSEARCHCode), endpoint)
		}
		elasticsearchconn, err := elasticsearch.NewClientWithOptions(client.Config.RegionId, client.getSdkConfig(), client.Config.getAuthCredential(true))
		if err != nil {
			return nil, fmt.Errorf("unable to initialize the Elasticsearch client: %#v", err)
		}

		elasticsearchconn.AppendUserAgent(Terraform, TerraformVersion)
		elasticsearchconn.AppendUserAgent(Provider, ProviderVersion)
		elasticsearchconn.AppendUserAgent(Module, client.Config.ConfigurationSource)
		elasticsearchconn.SetHTTPSInsecure(client.Config.Insecure)
		if client.Config.Proxy != "" {
			elasticsearchconn.SetHttpProxy(client.Config.Proxy)
		}
		client.elasticsearchconn = elasticsearchconn
	}

	return do(client.elasticsearchconn)
}
func (client *ApsaraStackClient) WithEssClient(do func(*ess.Client) (interface{}, error)) (interface{}, error) {
	// Initialize the ESS client if necessary
	if client.essconn == nil {
		endpoint := client.Config.EssEndpoint
		if endpoint == "" {
			return nil, fmt.Errorf("unable to initialize the ess client: endpoint or domain is not provided for ess service")
		}
		if endpoint != "" {
			endpoints.AddEndpointMapping(client.Config.RegionId, string(ESSCode), endpoint)
		}
		if strings.HasPrefix(endpoint, "http") {
			endpoint = strings.TrimPrefix(strings.TrimPrefix(endpoint, "http://"), "https://")
		}
		essconn, err := ess.NewClientWithOptions(client.Config.RegionId, client.getSdkConfig(), client.Config.getAuthCredential(true))
		if err != nil {
			return nil, fmt.Errorf("unable to initialize the ESS client: %#v", err)
		}
		essconn.Domain = endpoint
		essconn.AppendUserAgent(Terraform, TerraformVersion)
		essconn.AppendUserAgent(Provider, ProviderVersion)
		essconn.AppendUserAgent(Module, client.Config.ConfigurationSource)
		essconn.SetHTTPSInsecure(client.Config.Insecure)
		if client.Config.Proxy != "" {
			essconn.SetHttpsProxy(client.Config.Proxy)
			essconn.SetHttpProxy(client.Config.Proxy)
		}
		client.essconn = essconn
	}

	return do(client.essconn)
}

func (client *ApsaraStackClient) WithRkvClient(do func(*r_kvstore.Client) (interface{}, error)) (interface{}, error) {
	// Initialize the RKV client if necessary
	if client.rkvconn == nil {
		endpoint := client.Config.KVStoreEndpoint
		if endpoint == "" {
			return nil, fmt.Errorf("unable to initialize the kvstore client: endpoint or domain is not provided for logpop service")
		}
		if endpoint != "" {
			endpoints.AddEndpointMapping(client.Config.RegionId, fmt.Sprintf("R-%s", string(KVSTORECode)), endpoint)
		}
		if strings.HasPrefix(endpoint, "http") {
			endpoint = strings.TrimPrefix(strings.TrimPrefix(endpoint, "http://"), "https://")
		}
		rkvconn, err := r_kvstore.NewClientWithOptions(client.Config.RegionId, client.getSdkConfig(), client.Config.getAuthCredential(true))
		if err != nil {
			return nil, fmt.Errorf("unable to initialize the RKV client: %#v", err)
		}
		rkvconn.Domain = endpoint
		rkvconn.AppendUserAgent(Terraform, TerraformVersion)
		rkvconn.AppendUserAgent(Provider, ProviderVersion)
		rkvconn.AppendUserAgent(Module, client.Config.ConfigurationSource)
		rkvconn.SetHTTPSInsecure(client.Config.Insecure)
		if client.Config.Proxy != "" {
			rkvconn.SetHttpProxy(client.Config.Proxy)
		}
		client.rkvconn = rkvconn
	}

	return do(client.rkvconn)
}

func (client *ApsaraStackClient) WithGpdbClient(do func(*gpdb.Client) (interface{}, error)) (interface{}, error) {
	// Initialize the GPDB client if necessary
	if client.gpdbconn == nil {
		endpoint := client.Config.GpdbEndpoint
		if endpoint != "" {
			endpoints.AddEndpointMapping(client.Config.RegionId, string(GPDBCode), endpoint)
		}
		gpdbconn, err := gpdb.NewClientWithOptions(client.Config.RegionId, client.getSdkConfig(), client.Config.getAuthCredential(true))
		if err != nil {
			return nil, fmt.Errorf("unable to initialize the GPDB client: %#v", err)
		}

		gpdbconn.Domain = endpoint
		gpdbconn.AppendUserAgent(Terraform, TerraformVersion)
		gpdbconn.AppendUserAgent(Provider, ProviderVersion)
		gpdbconn.AppendUserAgent(Module, client.Config.ConfigurationSource)
		gpdbconn.SetHTTPSInsecure(client.Config.Insecure)
		if client.Config.Proxy != "" {
			gpdbconn.SetHttpProxy(client.Config.Proxy)
		}
		client.gpdbconn = gpdbconn
	}

	return do(client.gpdbconn)
}
func (client *ApsaraStackClient) WithAdbClient(do func(*adb.Client) (interface{}, error)) (interface{}, error) {
	// Initialize the adb client if necessary
	if client.adbconn == nil {
		endpoint := client.Config.AdbEndpoint
		if endpoint == "" {
			return nil, fmt.Errorf("unable to initialize the  client: endpoint or domain is not provided for  service")
		}
		adbconn, err := adb.NewClientWithOptions(client.Config.RegionId, client.getSdkConfig(), client.Config.getAuthCredential(true))
		if err != nil {
			return nil, fmt.Errorf("unable to initialize the adb client: %#v", err)

		}
		adbconn.Domain = endpoint
		adbconn.AppendUserAgent(Terraform, TerraformVersion)
		adbconn.AppendUserAgent(Provider, ProviderVersion)
		adbconn.AppendUserAgent(Module, client.Config.ConfigurationSource)
		adbconn.SetHTTPSInsecure(client.Config.Insecure)
		if client.Config.Proxy != "" {
			adbconn.SetHttpProxy(client.Config.Proxy)
		}
		client.adbconn = adbconn
	}

	return do(client.adbconn)
}
func (client *ApsaraStackClient) WithHbaseClient(do func(*hbase.Client) (interface{}, error)) (interface{}, error) {
	// Initialize the HBase client if necessary
	if client.hbaseconn == nil {
		endpoint := client.Config.HBaseEndpoint
		if endpoint == "" {
			return nil, fmt.Errorf("unable to initialize the  client: endpoint or domain is not provided for  service")
		}
		if endpoint != "" {
			endpoints.AddEndpointMapping(client.Config.RegionId, string(HBASECode), endpoint)
		}
		hbaseconn, err := hbase.NewClientWithOptions(client.Config.RegionId, client.getSdkConfig(), client.Config.getAuthCredential(true))
		if err != nil {
			return nil, fmt.Errorf("unable to initialize the hbase client: %#v", err)
		}

		hbaseconn.AppendUserAgent(Terraform, TerraformVersion)
		hbaseconn.AppendUserAgent(Provider, ProviderVersion)
		hbaseconn.AppendUserAgent(Module, client.Config.ConfigurationSource)
		hbaseconn.SetHTTPSInsecure(client.Config.Insecure)
		if client.Config.Proxy != "" {
			hbaseconn.SetHttpProxy(client.Config.Proxy)
		}
		client.hbaseconn = hbaseconn
	}

	return do(client.hbaseconn)
}
func (client *ApsaraStackClient) WithFcClient(do func(*fc.Client) (interface{}, error)) (interface{}, error) {
	goSdkMutex.Lock()
	defer goSdkMutex.Unlock()

	// Initialize the FC client if necessary
	if client.fcconn == nil {
		endpoint := client.Config.FcEndpoint
		if endpoint == "" {
			return nil, fmt.Errorf("unable to initialize the  client: endpoint or domain is not provided for  service")
		}
		if strings.HasPrefix(endpoint, "http") {
			endpoint = strings.TrimPrefix(strings.TrimPrefix(endpoint, "http://"), "https://")
		}
		accountId, err := client.AccountId()
		if err != nil {
			return nil, err
		}

		config := client.getSdkConfig()
		clientOptions := []fc.ClientOption{fc.WithSecurityToken(client.Config.SecurityToken), fc.WithTransport(config.HttpTransport),
			fc.WithTimeout(30), fc.WithRetryCount(DefaultClientRetryCountSmall)}
		fcconn, err := fc.NewClient(fmt.Sprintf("https://%s.%s", accountId, endpoint), string(ApiVersion20160815), client.Config.AccessKey, client.Config.SecretKey, clientOptions...)
		if err != nil {
			return nil, fmt.Errorf("unable to initialize the FC client: %#v", err)
		}

		fcconn.Config.UserAgent = client.getUserAgent()
		fcconn.Config.SecurityToken = client.Config.SecurityToken
		client.fcconn = fcconn
	}

	return do(client.fcconn)
}
func (client *ApsaraStackClient) WithVpcClient(do func(*vpc.Client) (interface{}, error)) (interface{}, error) {
	// Initialize the VPC client if necessary
	if client.vpcconn == nil {
		endpoint := client.Config.VpcEndpoint
		if endpoint == "" {
			return nil, fmt.Errorf("unable to initialize the vpc client: endpoint or domain is not provided for vpc service")
		}
		if endpoint != "" {
			endpoints.AddEndpointMapping(client.Config.RegionId, string(VPCCode), endpoint)
		}
		if strings.HasPrefix(endpoint, "http") {
			endpoint = strings.TrimPrefix(strings.TrimPrefix(endpoint, "http://"), "https://")
		}
		vpcconn, err := vpc.NewClientWithOptions(client.Config.RegionId, client.getSdkConfig(), client.Config.getAuthCredential(true))
		if err != nil {
			return nil, fmt.Errorf("unable to initialize the VPC client: %#v", err)
		}
		vpcconn.Domain = endpoint
		vpcconn.AppendUserAgent(Terraform, TerraformVersion)
		vpcconn.AppendUserAgent(Provider, ProviderVersion)
		vpcconn.AppendUserAgent(Module, client.Config.ConfigurationSource)
		vpcconn.SetHTTPSInsecure(client.Config.Insecure)
		if client.Config.Proxy != "" {
			vpcconn.SetHttpsProxy(client.Config.Proxy)
			vpcconn.SetHttpProxy(client.Config.Proxy)
		}
		client.vpcconn = vpcconn
	}

	return do(client.vpcconn)
}

func (client *ApsaraStackClient) WithSlbClient(do func(*slb.Client) (interface{}, error)) (interface{}, error) {
	// Initialize the SLB client if necessary
	if client.slbconn == nil {
		endpoint := client.Config.SlbEndpoint
		if endpoint == "" {
			return nil, fmt.Errorf("unable to initialize the slb client: endpoint or domain is not provided for slb service")
		}

		if endpoint != "" {
			endpoints.AddEndpointMapping(client.Config.RegionId, string(SLBCode), endpoint)
		}
		slbconn, err := slb.NewClientWithOptions(client.Config.RegionId, client.getSdkConfig(), client.Config.getAuthCredential(true))
		if err != nil {
			return nil, fmt.Errorf("unable to initialize the SLB client: %#v", err)
		}
		slbconn.Domain = endpoint
		slbconn.AppendUserAgent(Terraform, TerraformVersion)
		slbconn.AppendUserAgent(Provider, ProviderVersion)
		slbconn.AppendUserAgent(Module, client.Config.ConfigurationSource)
		slbconn.SetHTTPSInsecure(client.Config.Insecure)
		if client.Config.Proxy != "" {
			slbconn.SetHttpsProxy(client.Config.Proxy)
			slbconn.SetHttpProxy(client.Config.Proxy)
		}
		client.slbconn = slbconn
	}

	return do(client.slbconn)
}
func (client *ApsaraStackClient) WithDdsClient(do func(*dds.Client) (interface{}, error)) (interface{}, error) {
	// Initialize the DDS client if necessary
	if client.ddsconn == nil {
		endpoint := client.Config.DdsEndpoint
		if endpoint == "" {
			return nil, fmt.Errorf("unable to initialize the  client: endpoint or domain is not provided for  service")
		}
		if endpoint != "" {
			endpoints.AddEndpointMapping(client.Config.RegionId, string(DDSCode), endpoint)
		}
		ddsconn, err := dds.NewClientWithOptions(client.Config.RegionId, client.getSdkConfig(), client.Config.getAuthCredential(true))
		if err != nil {
			return nil, fmt.Errorf("unable to initialize the DDS client: %#v", err)
		}
		ddsconn.Domain = endpoint
		ddsconn.AppendUserAgent(Terraform, TerraformVersion)
		ddsconn.AppendUserAgent(Provider, ProviderVersion)
		ddsconn.AppendUserAgent(Module, client.Config.ConfigurationSource)
		ddsconn.SetHTTPSInsecure(client.Config.Insecure)
		if client.Config.Proxy != "" {
			ddsconn.SetHttpProxy(client.Config.Proxy)
		}
		client.ddsconn = ddsconn
	}

	return do(client.ddsconn)
}

func (client *ApsaraStackClient) WithOssNewClient(do func(*ecs.Client) (interface{}, error)) (interface{}, error) {
	// Initialize the ECS client if necessary
	if client.ecsconn == nil {
		endpoint := client.Config.OssEndpoint
		if endpoint == "" {
			return nil, fmt.Errorf("unable to initialize the oss client: endpoint or domain is not provided for ecs service")
		}
		if endpoint != "" {
			endpoints.AddEndpointMapping(client.Config.RegionId, string(ECSCode), endpoint)
		}
		if strings.HasPrefix(endpoint, "http") {
			endpoint = strings.TrimPrefix(strings.TrimPrefix(endpoint, "http://"), "https://")
		}
		ecsconn, err := ecs.NewClientWithOptions(client.Config.RegionId, client.getSdkConfig().WithTimeout(time.Duration(60)*time.Second), client.Config.getAuthCredential(true))
		if err != nil {
			return nil, fmt.Errorf("unable to initialize the ECS client: %#v", err)
		}

		ecsconn.Domain = endpoint
		ecsconn.AppendUserAgent(Terraform, TerraformVersion)
		ecsconn.AppendUserAgent(Provider, ProviderVersion)
		ecsconn.AppendUserAgent(Module, client.Config.ConfigurationSource)
		ecsconn.SetHTTPSInsecure(client.Config.Insecure)
		if client.Config.Proxy != "" {
			ecsconn.SetHttpsProxy(client.Config.Proxy)
			ecsconn.SetHttpProxy(client.Config.Proxy)
		}
		client.ecsconn = ecsconn
	}

	return do(client.ecsconn)
}

func (client *ApsaraStackClient) describeEndpointForService(serviceCode string) (*location.Endpoint, error) {
	args := location.CreateDescribeEndpointsRequest()
	args.ServiceCode = serviceCode
	args.Id = client.Config.RegionId
	args.Domain = client.Config.LocationEndpoint

	if args.Domain == "" {
		args.Domain = "location-readonly.aliyuncs.com"
	}

	locationClient, err := location.NewClientWithOptions(client.Config.RegionId, client.getSdkConfig(), client.Config.getAuthCredential(true))
	if err != nil {
		return nil, fmt.Errorf("Unable to initialize the location client: %#v", err)

	}
	locationClient.AppendUserAgent(Terraform, TerraformVersion)
	locationClient.AppendUserAgent(Provider, ProviderVersion)
	locationClient.AppendUserAgent(Module, client.Config.ConfigurationSource)
	locationClient.SetHTTPSInsecure(client.Config.Insecure)
	if client.Config.Proxy != "" {
		locationClient.SetHttpsProxy(client.Config.Proxy)
	}
	endpointsResponse, err := locationClient.DescribeEndpoints(args)
	if err != nil {
		return nil, fmt.Errorf("Describe %s endpoint using region: %#v got an error: %#v.", serviceCode, client.RegionId, err)
	}
	if endpointsResponse != nil && len(endpointsResponse.Endpoints.Endpoint) > 0 {
		for _, e := range endpointsResponse.Endpoints.Endpoint {
			if e.Type == "openAPI" {
				return &e, nil
			}
		}
	}
	return nil, fmt.Errorf("There is no any available endpoint for %s in region %s.", serviceCode, client.RegionId)
}

func (client *ApsaraStackClient) NewCommonRequest(product, serviceCode, schema string, apiVersion ApiVersion) (*requests.CommonRequest, error) {
	request := requests.NewCommonRequest()
	if strings.ToLower(client.Config.Protocol) == "https" {
		request.Scheme = "https"
	} else {
		request.Scheme = "http"
	}
	if client.Config.Insecure {
		request.SetHTTPSInsecure(client.Config.Insecure)
	}

	var endpoint string
	if strings.ToUpper(product) == "SLB" {
		endpoint = client.Config.SlbEndpoint
	}
	if strings.ToUpper(product) == "ECS" {
		endpoint = client.Config.EcsEndpoint
	}
	if strings.ToUpper(product) == "ASCM" {
		endpoint = client.Config.AscmEndpoint
	}

	if endpoint == "" {
		endpointItem, err := client.describeEndpointForService(serviceCode)
		if err != nil {
			return nil, fmt.Errorf("describeEndpointForService got an error: %#v.", err)
		}
		if endpointItem != nil {
			endpoint = endpointItem.Endpoint
		}
	}
	// Use product code to find product domain
	if endpoint != "" {
		request.Domain = endpoint
	} else {
		// When getting endpoint failed by location, using custom endpoint instead
		request.Domain = fmt.Sprintf("%s.%s.aliyuncs.com", strings.ToLower(serviceCode), client.RegionId)
	}
	request.Version = string(apiVersion)
	request.RegionId = client.RegionId
	request.Product = product

	if strings.ToUpper(product) == "SLB" {
		request.QueryParams = map[string]string{"AccessKeySecret": client.SecretKey, "Product": "slb", "Department": client.Department, "ResourceGroup": client.ResourceGroup, "Version": string(apiVersion)}
	}
	if strings.ToUpper(product) == "ECS" {
		request.QueryParams = map[string]string{"AccessKeySecret": client.SecretKey, "Product": "ecs", "Department": client.Department, "ResourceGroup": client.ResourceGroup, "Version": string(apiVersion)}
	}
	if strings.ToUpper(product) == "ASCM" {
		request.QueryParams = map[string]string{"AccessKeySecret": client.SecretKey, "Product": "ascm", "Department": client.Department, "ResourceGroup": client.ResourceGroup, "Version": string(apiVersion)}
	}

	request.AppendUserAgent(Terraform, TerraformVersion)
	request.AppendUserAgent(Provider, ProviderVersion)
	request.AppendUserAgent(Module, client.Config.ConfigurationSource)
	request.SetHTTPSInsecure(client.Config.Insecure)
	return request, nil
}

func (client *ApsaraStackClient) getSdkConfig() *sdk.Config {
	log.Printf("Protocol is set to %s", client.Config.Protocol)
	return sdk.NewConfig().
		WithMaxRetryTime(DefaultClientRetryCountSmall).
		WithTimeout(time.Duration(30) * time.Second).
		WithEnableAsync(true).
		WithGoRoutinePoolSize(100).
		WithMaxTaskQueueSize(10000).
		WithDebug(false).
		WithHttpTransport(client.getTransport()).
		WithScheme(strings.ToLower(client.Config.Protocol))
}

func (client *ApsaraStackClient) getTransport() *http.Transport {
	handshakeTimeout, err := strconv.Atoi(os.Getenv("TLSHandshakeTimeout"))
	if err != nil {
		handshakeTimeout = 120
	}
	transport := &http.Transport{}
	transport.TLSHandshakeTimeout = time.Duration(handshakeTimeout) * time.Second

	return transport
}
func (client *ApsaraStackClient) AccountId() (string, error) {
	client.accountIdMutex.Lock()
	defer client.accountIdMutex.Unlock()

	if client.accountId == "" {
		log.Printf("[DEBUG] account_id not provided, attempting to retrieve it automatically...")
		identity, err := client.GetCallerIdentity()
		if err != nil {
			return "", err
		}
		if identity == "" {
			return "", fmt.Errorf("caller identity doesn't contain any AccountId")
		}
		client.accountId = identity
	}
	return client.accountId, nil
}
func (client *ApsaraStackClient) getHttpProxy() (proxy *url.URL, err error) {
	if client.Config.Protocol == "HTTPS" {
		if rawurl := os.Getenv("HTTPS_PROXY"); rawurl != "" {
			proxy, err = url.Parse(rawurl)
		} else if rawurl := os.Getenv("https_proxy"); rawurl != "" {
			proxy, err = url.Parse(rawurl)
		}
	} else {
		if rawurl := os.Getenv("HTTP_PROXY"); rawurl != "" {
			proxy, err = url.Parse(rawurl)
		} else if rawurl := os.Getenv("http_proxy"); rawurl != "" {
			proxy, err = url.Parse(rawurl)
		}
	}
	return proxy, err
}

func (client *ApsaraStackClient) skipProxy(endpoint string) (bool, error) {
	var urls []string
	if rawurl := os.Getenv("NO_PROXY"); rawurl != "" {
		urls = strings.Split(rawurl, ",")
	} else if rawurl := os.Getenv("no_proxy"); rawurl != "" {
		urls = strings.Split(rawurl, ",")
	}
	for _, value := range urls {
		if strings.HasPrefix(value, "*") {
			value = fmt.Sprintf(".%s", value)
		}
		noProxyReg, err := regexp.Compile(value)
		if err != nil {
			return false, err
		}
		if noProxyReg.MatchString(endpoint) {
			return true, nil
		}
	}
	return false, nil
}
func (client *ApsaraStackClient) WithKmsClient(do func(*kms.Client) (interface{}, error)) (interface{}, error) {
	// Initialize the KMS client if necessary
	if client.kmsconn == nil {

		endpoint := client.Config.KmsEndpoint
		if endpoint == "" {
			return nil, fmt.Errorf("unable to initialize the kms client: endpoint or domain is not provided for KMS service")
		}
		if endpoint != "" {
			endpoints.AddEndpointMapping(client.Config.RegionId, string(KMSCode), endpoint)
		}
		kmsconn, err := kms.NewClientWithOptions(client.Config.RegionId, client.getSdkConfig(), client.Config.getAuthCredential(true))
		if err != nil {
			return nil, fmt.Errorf("unable to initialize the kms client: %#v", err)
		}
		kmsconn.AppendUserAgent(Terraform, TerraformVersion)
		kmsconn.Domain = endpoint
		kmsconn.AppendUserAgent(Provider, ProviderVersion)
		kmsconn.AppendUserAgent(Module, client.Config.ConfigurationSource)
		kmsconn.SetHTTPSInsecure(client.Config.Insecure)
		if client.Config.Proxy != "" {
			kmsconn.SetHttpProxy(client.Config.Proxy)
		}
		client.kmsconn = kmsconn
	}
	return do(client.kmsconn)
}
func (client *ApsaraStackClient) RoleIds() (int, error) {
	client.roleIdMutex.Lock()
	defer client.roleIdMutex.Unlock()

	if client.roleId == 0 {
		log.Printf("[DEBUG] role_ids not provided, attempting to retrieve it automatically...")
		roleId, err := client.GetCallerDefaultRole()
		if err != nil {
			return 0, err
		}
		if roleId == 0 {
			return 0, fmt.Errorf("caller identity doesn't contain default RoleId")
		}
		client.roleId = roleId
	}
	return client.roleId, nil
}
func (client *ApsaraStackClient) GetCallerDefaultRole() (int, error) {

	resp, err := client.GetCallerInfo()
	response := &RoleId{}
	err = json.Unmarshal(resp.GetHttpContentBytes(), response)
	roleId := response.Data.DefaultRole.Id

	if roleId == 0 {
		return 0, fmt.Errorf("default roleId not found")
	}
	return roleId, err
}
func (client *ApsaraStackClient) GetCallerInfo() (*responses.BaseResponse, error) {

	endpoint := client.Config.AscmEndpoint
	if endpoint == "" {
		return nil, fmt.Errorf("unable to initialize the ascm client: endpoint or domain is not provided for ascm service")
	}
	if endpoint != "" {
		endpoints.AddEndpointMapping(client.Config.RegionId, string(ASCMCode), endpoint)
	}
	ascmClient, err := sdk.NewClientWithAccessKey(client.Config.RegionId, client.Config.AccessKey, client.Config.SecretKey)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize the ascm client: %#v", err)
	}

	ascmClient.AppendUserAgent(Terraform, TerraformVersion)
	ascmClient.AppendUserAgent(Provider, ProviderVersion)
	ascmClient.AppendUserAgent(Module, client.Config.ConfigurationSource)
	ascmClient.SetHTTPSInsecure(client.Config.Insecure)
	ascmClient.Domain = endpoint
	if client.Config.Proxy != "" {
		ascmClient.SetHttpProxy(client.Config.Proxy)
	}
	if client.Config.Department == "" || client.Config.ResourceGroup == "" {
		return nil, fmt.Errorf("unable to initialize the ascm client: department or resource_group is not provided")
	}
	request := requests.NewCommonRequest()
	if strings.ToLower(client.Config.Protocol) == "https" {
		request.Scheme = "https"
	} else {
		request.Scheme = "http"
	}
	if client.Config.Insecure {
		request.SetHTTPSInsecure(client.Config.Insecure)
	}
	request.Method = "GET"         // Set request method
	request.Product = "ascm"       // Specify product
	request.Domain = endpoint      // Location Service will not be enabled if the host is specified. For example, service with a Certification type-Bearer Token should be specified
	request.Version = "2019-05-10" // Specify product version
	request.ApiName = "GetUserInfo"
	request.QueryParams = map[string]string{
		"AccessKeySecret":  client.Config.SecretKey,
		"Product":          "ascm",
		"Department":       client.Config.Department,
		"ResourceGroup":    client.Config.ResourceGroup,
		"RegionId":         client.RegionId,
		"Action":           "GetAllNavigationInfo",
		"Version":          "2019-05-10",
		"SignatureVersion": "1.0",
	}
	resp := responses.BaseResponse{}
	request.TransToAcsRequest()
	err = ascmClient.DoAction(request, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

type RoleId struct {
	Data struct {
		DefaultRole struct {
			Id int `json:"id"`
		} `json:"defaultRole"`
	} `json:"data"`
}

func (client *ApsaraStackClient) GetCallerIdentity() (string, error) {

	endpoint := client.Config.AscmEndpoint
	if endpoint == "" {
		return "", fmt.Errorf("unable to initialize the ascm client: endpoint or domain is not provided for ascm service")
	}
	if endpoint != "" {
		endpoints.AddEndpointMapping(client.Config.RegionId, string(ASCMCode), endpoint)
	}
	ascmClient, err := sdk.NewClientWithAccessKey(client.Config.RegionId, client.Config.AccessKey, client.Config.SecretKey)
	if err != nil {
		return "", fmt.Errorf("unable to initialize the ascm client: %#v", err)
	}

	ascmClient.AppendUserAgent(Terraform, TerraformVersion)
	ascmClient.AppendUserAgent(Provider, ProviderVersion)
	ascmClient.AppendUserAgent(Module, client.Config.ConfigurationSource)
	ascmClient.SetHTTPSInsecure(client.Config.Insecure)
	ascmClient.Domain = endpoint
	if client.Config.Proxy != "" {
		ascmClient.SetHttpProxy(client.Config.Proxy)
	}
	if client.Config.Department == "" || client.Config.ResourceGroup == "" {
		return "", fmt.Errorf("unable to initialize the ascm client: department or resource_group is not provided")
	}
	request := requests.NewCommonRequest()
	if strings.ToLower(client.Config.Protocol) == "https" {
		request.Scheme = "https"
	} else {
		request.Scheme = "http"
	}
	if client.Config.Insecure {
		request.SetHTTPSInsecure(client.Config.Insecure)
	}
	request.Method = "GET"         // Set request method
	request.Product = "ascm"       // Specify product
	request.Domain = endpoint      // Location Service will not be enabled if the host is specified. For example, service with a Certification type-Bearer Token should be specified
	request.Version = "2019-05-10" // Specify product version
	request.ApiName = "GetUserInfo"
	request.QueryParams = map[string]string{
		"AccessKeySecret":  client.Config.SecretKey,
		"Product":          "ascm",
		"Department":       client.Config.Department,
		"ResourceGroup":    client.Config.ResourceGroup,
		"RegionId":         client.RegionId,
		"Action":           "GetAllNavigationInfo",
		"Version":          "2019-05-10",
		"SignatureVersion": "1.0",
	}
	resp := responses.BaseResponse{}
	request.TransToAcsRequest()
	err = ascmClient.DoAction(request, &resp)
	if err != nil {
		return "", err
	}
	response := &AccountId{}
	err = json.Unmarshal(resp.GetHttpContentBytes(), response)
	ownerId := response.Data.PrimaryKey

	if ownerId == "" {
		return "", fmt.Errorf("ownerId not found")
	}
	return ownerId, err
}

type AccountId struct {
	Data struct {
		PrimaryKey string `json:"primaryKey"`
	} `json:"data"`
}

func (client *ApsaraStackClient) WithBssopenapiClient(do func(*bssopenapi.Client) (interface{}, error)) (interface{}, error) {
	// Initialize the bssopenapi client if necessary
	if client.bssopenapiconn == nil {
		endpoint := client.Config.BssOpenApiEndpoint
		if endpoint == "" {
			return nil, fmt.Errorf("unable to initialize the bss client: endpoint or domain is not provided for bss service")
		}
		if endpoint != "" {
			endpoints.AddEndpointMapping(client.Config.RegionId, string(BSSOPENAPICode), endpoint)
		}

		bssopenapiconn, err := bssopenapi.NewClientWithOptions(client.Config.RegionId, client.getSdkConfig(), client.Config.getAuthCredential(true))
		if err != nil {
			return nil, fmt.Errorf("unable to initialize the BSSOPENAPI client: %#v", err)
		}
		bssopenapiconn.AppendUserAgent(Terraform, TerraformVersion)
		bssopenapiconn.AppendUserAgent(Provider, ProviderVersion)
		bssopenapiconn.AppendUserAgent(Module, client.Config.ConfigurationSource)
		bssopenapiconn.SetHTTPSInsecure(client.Config.Insecure)
		if client.Config.Proxy != "" {
			bssopenapiconn.SetHttpsProxy(client.Config.Proxy)
		}
		client.bssopenapiconn = bssopenapiconn
	}

	return do(client.bssopenapiconn)
}
func (client *ApsaraStackClient) WithOssClient(do func(*oss.Client) (interface{}, error)) (interface{}, error) {
	goSdkMutex.Lock()
	defer goSdkMutex.Unlock()

	// Initialize the OSS client if necessary
	if client.ossconn == nil {
		schma := "http"
		endpoint := client.Config.OssEndpoint
		if endpoint == "" {
			return nil, fmt.Errorf("unable to initialize the oss client: endpoint or domain is not provided for OSS service")
		}
		if endpoint == "" {
			endpointItem, _ := client.describeEndpointForService(strings.ToLower(string(OSSCode)))
			if endpointItem != nil {
				if len(endpointItem.Protocols.Protocols) > 0 {
					// HTTP or HTTPS
					schma = strings.ToLower(endpointItem.Protocols.Protocols[0])
					for _, p := range endpointItem.Protocols.Protocols {
						if strings.ToLower(p) == "http" {
							schma = strings.ToLower(p)
							break
						}
					}
				}
				endpoint = endpointItem.Endpoint
			}
		}
		if !strings.HasPrefix(endpoint, "http") {
			endpoint = fmt.Sprintf("%s://%s", schma, endpoint)
		}

		clientOptions := []oss.ClientOption{oss.UserAgent(client.getUserAgent()),
			oss.SecurityToken(client.Config.SecurityToken)}
		if client.Config.Proxy != "" {
			clientOptions = append(clientOptions, oss.Proxy(client.Config.Proxy))
		}

		clientOptions = append(clientOptions, oss.UseCname(false))

		ossconn, err := oss.New(endpoint, client.Config.AccessKey, client.Config.SecretKey, clientOptions...)
		if err != nil {
			return nil, fmt.Errorf("unable to initialize the OSS client: %#v", err)
		}

		client.ossconn = ossconn
	}

	return do(client.ossconn)
}

func (client *ApsaraStackClient) WithRamClient(do func(*ram.Client) (interface{}, error)) (interface{}, error) {
	// Initialize the RAM client if necessary
	if client.ramconn == nil {
		endpoint := client.Config.RamEndpoint
		if endpoint == "" {
			return nil, fmt.Errorf("unable to initialize the ram client: endpoint or domain is not provided for ram operation")
		}
		if strings.HasPrefix(endpoint, "http") {
			endpoint = fmt.Sprintf("https://%s", strings.TrimPrefix(endpoint, "http://"))
		}
		if endpoint != "" {
			endpoints.AddEndpointMapping(client.Config.RegionId, string(RAMCode), endpoint)
		}

		ramconn, err := ram.NewClientWithOptions(client.Config.RegionId, client.getSdkConfig(), client.Config.getAuthCredential(true))
		if err != nil {
			return nil, fmt.Errorf("unable to initialize the RAM client: %#v", err)
		}
		ramconn.AppendUserAgent(Terraform, TerraformVersion)
		ramconn.AppendUserAgent(Provider, ProviderVersion)
		ramconn.AppendUserAgent(Module, client.Config.ConfigurationSource)
		ramconn.SetHTTPSInsecure(client.Config.Insecure)
		if client.Config.Proxy != "" {
			ramconn.SetHttpsProxy(client.Config.Proxy)
		}
		client.ramconn = ramconn
	}

	return do(client.ramconn)
}

func (client *ApsaraStackClient) WithRdsClient(do func(*rds.Client) (interface{}, error)) (interface{}, error) {
	// Initialize the RDS client if necessary
	if client.rdsconn == nil {
		endpoint := client.Config.RdsEndpoint
		if endpoint == "" {
			return nil, fmt.Errorf("unable to initialize the rds client: endpoint or domain is not provided for RDS service")
		}
		if endpoint != "" {
			endpoints.AddEndpointMapping(client.Config.RegionId, string(RDSCode), endpoint)
		}
		rdsconn, err := rds.NewClientWithOptions(client.Config.RegionId, client.getSdkConfig(), client.Config.getAuthCredential(true))
		if err != nil {
			return nil, fmt.Errorf("unable to initialize the RDS client: %#v", err)
		}
		rdsconn.Domain = endpoint
		rdsconn.AppendUserAgent(Terraform, TerraformVersion)
		rdsconn.AppendUserAgent(Provider, ProviderVersion)
		rdsconn.AppendUserAgent(Module, client.Config.ConfigurationSource)
		rdsconn.SetHTTPSInsecure(client.Config.Insecure)

		if client.Config.Proxy != "" {
			rdsconn.SetHttpProxy(client.Config.Proxy)
		}

		client.rdsconn = rdsconn
	}

	return do(client.rdsconn)
}

func (client *ApsaraStackClient) WithCdnClient_new(do func(*cdn_new.Client) (interface{}, error)) (interface{}, error) {
	// Initialize the CDN client if necessary
	if client.cdnconn_new == nil {
		endpoint := client.Config.CdnEndpoint
		if endpoint == "" {
			return nil, fmt.Errorf("unable to initialize the CDN client: endpoint or domain is not provided for CDN service")
		}
		if endpoint != "" {
			endpoints.AddEndpointMapping(client.Config.RegionId, string(CDNCode), endpoint)
		}
		cdnconn, err := cdn_new.NewClientWithOptions(client.Config.RegionId, client.getSdkConfig(), client.Config.getAuthCredential(true))
		if err != nil {
			return nil, fmt.Errorf("unable to initialize the CDN client: %#v", err)
		}

		cdnconn.AppendUserAgent(Terraform, TerraformVersion)
		cdnconn.AppendUserAgent(Provider, ProviderVersion)
		cdnconn.AppendUserAgent(Module, client.Config.ConfigurationSource)
		cdnconn.SetHTTPSInsecure(client.Config.Insecure)
		if client.Config.Proxy != "" {
			cdnconn.SetHttpsProxy(client.Config.Proxy)
		}
		client.cdnconn_new = cdnconn
	}

	return do(client.cdnconn_new)
}
func (client *ApsaraStackClient) getUserAgent() string {
	return fmt.Sprintf("%s/%s %s/%s %s/%s", Terraform, TerraformVersion, Provider, ProviderVersion, Module, client.Config.ConfigurationSource)
}
func (client *ApsaraStackClient) WithCsClient(do func(*cs.Client) (interface{}, error)) (interface{}, error) {
	goSdkMutex.Lock()
	defer goSdkMutex.Unlock()

	// Initialize the CS client if necessary
	if client.csconn == nil {
		csconn := cs.NewClientForAussumeRole(client.Config.AccessKey, client.Config.SecretKey, client.Config.SecurityToken)
		csconn.SetUserAgent(client.getUserAgent())
		endpoint := client.Config.CsEndpoint
		if endpoint == "" {
			return nil, fmt.Errorf("unable to initialize the cs client: endpoint or domain is not provided for cs service")
		}
		if endpoint != "" {
			if !strings.HasPrefix(endpoint, "http") {
				endpoint = fmt.Sprintf("https://%s", strings.TrimPrefix(endpoint, "://"))
			}
			csconn.SetEndpoint(endpoint)
		}
		if client.Config.Proxy != "" {
			os.Setenv("http_proxy", client.Config.Proxy)
		}
		client.csconn = csconn
	}

	return do(client.csconn)
}

func (client *ApsaraStackClient) getHttpProxyUrl() *url.URL {
	for _, v := range []string{"HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy"} {
		value := strings.Trim(os.Getenv(v), " ")
		if value != "" {
			if !regexp.MustCompile(`^http(s)?://`).MatchString(value) {
				value = fmt.Sprintf("https://%s", value)
			}
			proxyUrl, err := url.Parse(value)
			if err == nil {
				return proxyUrl
			}
			break
		}
	}
	return nil
}

func (client *ApsaraStackClient) WithOssBucketByName(bucketName string, do func(*oss.Bucket) (interface{}, error)) (interface{}, error) {
	return client.WithOssClient(func(ossClient *oss.Client) (interface{}, error) {
		bucket, err := client.ossconn.Bucket(bucketName)

		if err != nil {
			return nil, fmt.Errorf("unable to get the bucket %s: %#v", bucketName, err)
		}
		return do(bucket)
	})
}

func (client *ApsaraStackClient) WithOnsClient(do func(*ons.Client) (interface{}, error)) (interface{}, error) {
	// Initialize the ons client if necessary
	if client.onsconn == nil {
		endpoint := client.Config.OnsEndpoint
		if endpoint == "" {
			return nil, fmt.Errorf("unable to initialize the ons client: endpoint or domain is not provided for ons service")
		}
		if endpoint != "" {
			endpoints.AddEndpointMapping(client.Config.RegionId, string(ONSCode), endpoint)
		}
		if strings.HasPrefix(endpoint, "http") {
			endpoint = strings.TrimPrefix(strings.TrimPrefix(endpoint, "http://"), "https://")
		}
		onsconn, err := ons.NewClientWithAccessKey(client.RegionId, client.AccessKey, client.SecretKey)
		if err != nil {
			return nil, fmt.Errorf("unable to initialize the ONS client: %#v", err)
		}

		onsconn.AppendUserAgent(Terraform, TerraformVersion)
		onsconn.AppendUserAgent(Provider, ProviderVersion)
		onsconn.Domain = endpoint

		onsconn.AppendUserAgent(Module, client.Config.ConfigurationSource)
		onsconn.SetHTTPSInsecure(client.Config.Insecure)
		if client.Config.Proxy != "" {
			onsconn.SetHttpProxy(client.Config.Proxy)
		}
		client.onsconn = onsconn
	}

	return do(client.onsconn)
}

func (client *ApsaraStackClient) WithLogClient(do func(*sls.Client) (interface{}, error)) (interface{}, error) {
	goSdkMutex.Lock()
	defer goSdkMutex.Unlock()

	// Initialize the LOG client if necessary
	if client.logconn == nil {
		endpoint := client.Config.LogEndpoint
		if endpoint == "" {
			return nil, fmt.Errorf("unable to initialize the log client: endpoint or domain is not provided for log service")
		}
		if strings.HasPrefix(endpoint, "http") {
			endpoint = strings.TrimPrefix(strings.TrimPrefix(endpoint, "http://"), "https://")
		}
		if client.Config.Proxy != "" {
			os.Setenv("http_proxy", client.Config.Proxy)
		}
		client.logconn = &sls.Client{
			AccessKeyID:     client.Config.OrganizationAccessKey,
			AccessKeySecret: client.Config.OrganizationSecretKey,
			Endpoint:        client.Config.SLSOpenAPIEndpoint,
			SecurityToken:   client.Config.SecurityToken,
			UserAgent:       client.getUserAgent(),
		}
	}

	return do(client.logconn)
}
func (client *ApsaraStackClient) WithLogPopClient(do func(*slsPop.Client) (interface{}, error)) (interface{}, error) {
	// Initialize the HBase client if necessary
	if client.logpopconn == nil {
		endpoint := client.Config.LogEndpoint
		if endpoint == "" {
			return nil, fmt.Errorf("unable to initialize the lopgpop client: endpoint or domain is not provided for logpop service")
		}
		if endpoint != "" {
			endpoint = fmt.Sprintf("%s."+endpoint, client.Config.RegionId)
		}
		logpopconn, err := slsPop.NewClientWithOptions(client.Config.RegionId, client.getSdkConfig(), client.Config.getAuthCredential(true))

		if err != nil {
			return nil, fmt.Errorf("unable to initialize the sls client: %#v", err)
		}

		logpopconn.AppendUserAgent(Terraform, TerraformVersion)
		logpopconn.AppendUserAgent(Provider, ProviderVersion)
		logpopconn.AppendUserAgent(Module, client.Config.ConfigurationSource)
		client.logpopconn = logpopconn
	}

	return do(client.logpopconn)
}

func (client *ApsaraStackClient) WithEdasClient(do func(*edas.Client) (interface{}, error)) (interface{}, error) {
	// Initialize the edas client if necessary
	if client.edasconn == nil {
		endpoint := client.Config.EdasEndpoint
		if endpoint == "" {
			endpoint = loadEndpoint(client.Config.RegionId, EDASCode)
		}
		if endpoint != "" {
			endpoints.AddEndpointMapping(client.Config.RegionId, string(EDASCode), endpoint)
		}
		edasconn, err := edas.NewClientWithOptions(client.Config.RegionId, client.getSdkConfig().WithTimeout(time.Duration(60)*time.Second), client.Config.getAuthCredential(true))
		if err != nil {
			return nil, fmt.Errorf("unable to initialize the ALIKAFKA client: %#v", err)
		}
		edasconn.SetReadTimeout(time.Duration(client.Config.ClientReadTimeout) * time.Millisecond)
		edasconn.SetConnectTimeout(time.Duration(client.Config.ClientConnectTimeout) * time.Millisecond)
		edasconn.SourceIp = client.Config.SourceIp
		edasconn.SecureTransport = client.Config.SecureTransport
		edasconn.AppendUserAgent(Terraform, TerraformVersion)
		edasconn.AppendUserAgent(Provider, ProviderVersion)
		edasconn.AppendUserAgent(Module, client.Config.ConfigurationSource)
		client.edasconn = edasconn
	}

	return do(client.edasconn)
}

func (client *ApsaraStackClient) WithCrEEClient(do func(*cr_ee.Client) (interface{}, error)) (interface{}, error) {
	// Initialize the CR EE client if necessary
	if client.creeconn == nil {
		endpoint := client.Config.CrEndpoint
		if endpoint == "" {
			return nil, fmt.Errorf("unable to initialize the CRee client: endpoint or domain is not provided for CR service")
		}
		if endpoint != "" {
			endpoints.AddEndpointMapping(client.Config.RegionId, string(CRCode), endpoint)
		}
		creeconn, err := cr_ee.NewClientWithOptions(client.Config.RegionId, client.getSdkConfig(), client.Config.getAuthCredential(true))
		if err != nil {
			return nil, fmt.Errorf("unable to initialize the CR EE client: %#v", err)
		}
		creeconn.AppendUserAgent(Terraform, TerraformVersion)
		creeconn.AppendUserAgent(Provider, ProviderVersion)
		creeconn.AppendUserAgent(Module, client.Config.ConfigurationSource)
		if client.Config.Proxy != "" {
			creeconn.SetHttpProxy(client.Config.Proxy)
		}
		client.creeconn = creeconn
	}

	return do(client.creeconn)
}

func (client *ApsaraStackClient) WithCrClient(do func(*cr.Client) (interface{}, error)) (interface{}, error) {
	// Initialize the CR client if necessary
	if client.crconn == nil {
		endpoint := client.Config.CrEndpoint

		if endpoint != "" {
			endpoints.AddEndpointMapping(client.Config.RegionId, string(CRCode), endpoint)
		}

		if strings.HasPrefix(endpoint, "http") {
			endpoint = strings.TrimPrefix(strings.TrimPrefix(endpoint, "http://"), "https://")
		}
		crconn, err := cr.NewClientWithOptions(client.Config.RegionId, client.getSdkConfig(), client.Config.getAuthCredential(true))
		if err != nil {
			return nil, fmt.Errorf("unable to initialize the CR client: %#v", err)
		}
		crconn.Domain = endpoint
		if client.Config.Proxy != "" {
			crconn.SetHttpProxy(client.Config.Proxy)
		}
		crconn.AppendUserAgent(Terraform, TerraformVersion)
		crconn.AppendUserAgent(Provider, ProviderVersion)
		crconn.AppendUserAgent(Module, client.Config.ConfigurationSource)
		client.crconn = crconn
	}

	return do(client.crconn)
}
func (client *ApsaraStackClient) WithDnsClient(do func(*alidns.Client) (interface{}, error)) (interface{}, error) {
	// Initialize the DNS client if necessary
	if client.dnsconn == nil {
		endpoint := client.Config.DnsEndpoint
		if endpoint != "" {
			endpoints.AddEndpointMapping(client.Config.RegionId, string(DNSCode), endpoint)
		}

		dnsconn, err := alidns.NewClientWithOptions(client.Config.RegionId, client.getSdkConfig(), client.Config.getAuthCredential(true))
		if err != nil {
			return nil, fmt.Errorf("unable to initialize the DNS client: %#v", err)
		}
		dnsconn.AppendUserAgent(Terraform, TerraformVersion)
		dnsconn.AppendUserAgent(Provider, ProviderVersion)
		dnsconn.AppendUserAgent(Module, client.Config.ConfigurationSource)
		dnsconn.Domain = endpoint
		if client.Config.Proxy != "" {
			dnsconn.SetHttpProxy(client.Config.Proxy)
		}
		client.dnsconn = dnsconn
	}

	return do(client.dnsconn)
}
func (client *ApsaraStackClient) WithCmsClient(do func(*cms.Client) (interface{}, error)) (interface{}, error) {
	// Initialize the CMS client if necessary
	if client.cmsconn == nil {
		endpoint := client.Config.CmsEndpoint
		if endpoint == "" {
			endpoint = loadEndpoint(client.Config.RegionId, CMSCode)
		}
		if endpoint != "" {
			endpoints.AddEndpointMapping(client.Config.RegionId, string(CMSCode), endpoint)
		}
		cmsconn, err := cms.NewClientWithOptions(client.Config.RegionId, client.getSdkConfig(), client.Config.getAuthCredential(true))
		if err != nil {
			return nil, fmt.Errorf("unable to initialize the CMS client: %#v", err)
		}

		cmsconn.Domain = endpoint
		cmsconn.AppendUserAgent(Terraform, TerraformVersion)
		cmsconn.AppendUserAgent(Provider, ProviderVersion)
		cmsconn.AppendUserAgent(Module, client.Config.ConfigurationSource)
		client.cmsconn = cmsconn
		if client.Config.Proxy != "" {
			cmsconn.SetHttpProxy(client.Config.Proxy)
		}
	}

	return do(client.cmsconn)
}
func (client *ApsaraStackClient) WithMaxComputeClient(do func(*maxcompute.Client) (interface{}, error)) (interface{}, error) {
	goSdkMutex.Lock()
	defer goSdkMutex.Unlock()

	// Initialize the MaxCompute client if necessary
	if client.maxcomputeconn == nil {
		endpoint := client.Config.MaxComputeEndpoint
		if endpoint == "" {
			endpoint = loadEndpoint(client.Config.RegionId, MAXCOMPUTECode)
		}
		if strings.HasPrefix(endpoint, "http") {
			endpoint = fmt.Sprintf("https://%s", strings.TrimPrefix(endpoint, "http://"))
		}
		if endpoint == "" {
			endpoint = "maxcompute.aliyuncs.com"
		}

		if endpoint != "" {
			endpoints.AddEndpointMapping(client.Config.RegionId, string(MAXCOMPUTECode), endpoint)
		}
		maxcomputeconn, err := maxcompute.NewClientWithOptions(client.Config.RegionId, client.getSdkConfig(), client.Config.getAuthCredential(false))
		if err != nil {
			return nil, fmt.Errorf("unable to initialize the MaxCompute client: %#v", err)
		}

		maxcomputeconn.AppendUserAgent(Terraform, TerraformVersion)
		maxcomputeconn.AppendUserAgent(Provider, ProviderVersion)
		maxcomputeconn.AppendUserAgent(Module, client.Config.ConfigurationSource)
		client.maxcomputeconn = maxcomputeconn
	}

	return do(client.maxcomputeconn)
}
func (client *ApsaraStackClient) NewEcsClient() (*rpc.Client, error) {
	productCode := "ecs"
	endpoint := client.Config.EcsEndpoint
	if v, ok := client.Config.Endpoints[productCode]; !ok || v.(string) == "" {
		if err := client.loadEndpoint(productCode); err != nil {
			return nil, err
		}
	}
	if v, ok := client.Config.Endpoints[productCode]; ok && v.(string) != "" {
		endpoint = v.(string)
	}
	if endpoint == "" {
		return nil, fmt.Errorf("[ERROR] missing the product %s endpoint.", productCode)
	}

	sdkConfig := client.teaSdkConfig
	sdkConfig.SetEndpoint(endpoint).SetReadTimeout(60000)

	conn, err := rpc.NewClient(&sdkConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize the %s client: %#v", productCode, err)
	}

	return conn, nil
}
func (client *ApsaraStackClient) NewRosClient() (*rpc.Client, error) {
	productCode := "ros"
	endpoint := client.Config.RosEndpoint
	if v, ok := client.Config.Endpoints[productCode]; !ok || v.(string) == "" {
		if err := client.loadEndpoint(productCode); err != nil {
			return nil, err
		}
	}
	if v, ok := client.Config.Endpoints[productCode]; ok && v.(string) != "" {
		endpoint = v.(string)
	}
	if endpoint == "" {
		return nil, fmt.Errorf("[ERROR] missing the product %s endpoint.", productCode)
	}
	sdkConfig := client.teaSdkConfig
	sdkConfig.SetEndpoint(endpoint)
	conn, err := rpc.NewClient(&sdkConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize the %s client: %#v", productCode, err)
	}
	return conn, nil
}

func (client *ApsaraStackClient) NewDmsenterpriseClient() (*rpc.Client, error) {
	productCode := "dmsenterprise"
	endpoint := client.Config.DmsEnterpriseEndpoint
	if v, ok := client.Config.Endpoints[productCode]; !ok || v.(string) == "" {
		if err := client.loadEndpoint(productCode); err != nil {
			endpoint = "dms-enterprise.aliyuncs.com"
			client.Config.Endpoints[productCode] = endpoint
			log.Printf("[ERROR] loading %s endpoint got an error: %#v. Using the central endpoint %s instead.", productCode, err, endpoint)
		}
	}
	if v, ok := client.Config.Endpoints[productCode]; ok && v.(string) != "" {
		endpoint = v.(string)
	}
	if endpoint == "" {
		return nil, fmt.Errorf("[ERROR] missing the product %s endpoint.", productCode)
	}
	sdkConfig := client.teaSdkConfig
	sdkConfig.SetEndpoint(endpoint)
	conn, err := rpc.NewClient(&sdkConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize the %s client: %#v", productCode, err)
	}
	return conn, nil
}
func (client *ApsaraStackClient) NewQuickbiClient() (*rpc.Client, error) {
	productCode := "quickbi"
	endpoint := client.Config.QuickbiEndpoint
	//endpoint := "quickbi-public.inter.env202.shuguang.com"
	if v, ok := client.Config.Endpoints[productCode]; !ok || v.(string) == "" {
		if err := client.loadEndpoint(productCode); err != nil {
			endpoint = fmt.Sprintf("quickbi.%s.aliyuncs.com", client.Config.RegionId)
			client.Config.Endpoints[productCode] = endpoint
			log.Printf("[ERROR] loading %s endpoint got an error: %#v. Using the endpoint %s instead.", productCode, err, endpoint)
		}
	}
	if v, ok := client.Config.Endpoints[productCode]; ok && v.(string) != "" {
		endpoint = v.(string)
	}
	if endpoint == "" {
		return nil, fmt.Errorf("[ERROR] missing the product %s endpoint.", productCode)
	}
	sdkConfig := client.teaSdkConfig
	sdkConfig.SetEndpoint(endpoint)
	conn, err := rpc.NewClient(&sdkConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize the %s client: %#v", productCode, err)
	}
	return conn, nil
}
func (client *ApsaraStackClient) NewAscmClient() (*rpc.Client, error) {
	productCode := "ascm"
	endpoint := client.Config.AscmEndpoint
	if endpoint == "" {
		if v, ok := client.Config.Endpoints[productCode]; !ok || v.(string) == "" {
			if err := client.loadEndpoint(productCode); err != nil {
				endpoint = fmt.Sprintf("eds-user.%s.aliyuncs.com", client.Config.RegionId)
				client.Config.Endpoints[productCode] = endpoint
				log.Printf("[ERROR] loading %s endpoint got an error: %#v. Using the endpoint %s instead.", productCode, err, endpoint)
			}
		}
		if v, ok := client.Config.Endpoints[productCode]; ok && v.(string) != "" {
			endpoint = v.(string)
		}
		if endpoint == "" {
			return nil, fmt.Errorf("[ERROR] missing the product %s endpoint.", productCode)
		}
	}
	sdkConfig := client.teaSdkConfig
	sdkConfig.SetEndpoint(endpoint)
	conn, err := rpc.NewClient(&sdkConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize the %s client: %#v", productCode, err)
	}
	return conn, nil
}

func (client *ApsaraStackClient) NewOdpsClient() (*rpc.Client, error) {
	productCode := "odps"
	endpoint := client.Config.MaxComputeEndpoint
	if endpoint == "" {
		if v, ok := client.Config.Endpoints[productCode]; !ok || v.(string) == "" {
			if err := client.loadEndpoint(productCode); err != nil {
				return nil, err
			}
		}
		if v, ok := client.Config.Endpoints[productCode]; ok && v.(string) != "" {
			endpoint = v.(string)
		}
	}
	if endpoint == "" {
		return nil, fmt.Errorf("[ERROR] missing the product %s endpoint.", productCode)
	}
	sdkConfig := client.teaSdkConfig
	sdkConfig.SetEndpoint(endpoint)
	conn, err := rpc.NewClient(&sdkConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize the %s client: %#v", productCode, err)
	}
	return conn, nil
}
