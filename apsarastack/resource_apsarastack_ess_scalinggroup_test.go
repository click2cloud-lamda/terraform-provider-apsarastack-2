package apsarastack

import (
	"fmt"
	"log"
	"testing"

	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ess"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/slb"
	"github.com/apsara-stack/terraform-provider-apsarastack/apsarastack/connectivity"
	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func init() {
	resource.AddTestSweepers("apsarastack_ess_scalinggroup", &resource.Sweeper{
		Name: "apsarastack_ess_scalinggroup",
		F:    testSweepEssGroups,
	})
}

func testSweepEssGroups(region string) error {
	rawClient, err := sharedClientForRegion(region)
	if err != nil {
		return fmt.Errorf("error getting Apsarastack client: %s", err)
	}
	client := rawClient.(*connectivity.ApsaraStackClient)

	prefixes := []string{
		"tf-testAcc",
		"tf_testAcc",
	}

	var groups []ess.ScalingGroup
	req := ess.CreateDescribeScalingGroupsRequest()

	req.RegionId = client.RegionId
	req.PageSize = requests.NewInteger(PageSizeLarge)
	req.PageNumber = requests.NewInteger(1)
	for {
		raw, err := client.WithEssClient(func(essClient *ess.Client) (interface{}, error) {
			return essClient.DescribeScalingGroups(req)
		})
		if err != nil {
			return fmt.Errorf("Error retrieving Scaling groups: %s", err)
		}
		resp, _ := raw.(*ess.DescribeScalingGroupsResponse)
		if resp == nil || len(resp.ScalingGroups.ScalingGroup) < 1 {
			break
		}
		groups = append(groups, resp.ScalingGroups.ScalingGroup...)

		if len(resp.ScalingGroups.ScalingGroup) < PageSizeLarge {
			break
		}

		page, err := getNextpageNumber(req.PageNumber)
		if err != nil {
			return err
		}
		req.PageNumber = page
	}

	sweeped := false
	for _, v := range groups {
		name := v.ScalingGroupName
		id := v.ScalingGroupId
		skip := true
		for _, prefix := range prefixes {
			if strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)) {
				skip = false
				break
			}
		}
		if skip {
			log.Printf("[INFO] Skipping Scaling Group: %s (%s)", name, id)
			continue
		}
		sweeped = true
		log.Printf("[INFO] Deleting Scaling Group: %s (%s)", name, id)
		req := ess.CreateDeleteScalingGroupRequest()
		req.ScalingGroupId = id
		req.ForceDelete = requests.NewBoolean(true)
		_, err := client.WithEssClient(func(essClient *ess.Client) (interface{}, error) {
			return essClient.DeleteScalingGroup(req)
		})
		if err != nil {
			log.Printf("[ERROR] Failed to delete Scaling Group (%s (%s)): %s", name, id, err)
		}
	}
	if sweeped {
		time.Sleep(2 * time.Minute)
	}
	return nil
}

func TestAccApsaraStackEssScalingGroup_basic(t *testing.T) {
	rand := acctest.RandIntRange(10000, 999999)
	var v ess.ScalingGroup
	resourceId := "apsarastack_ess_scaling_group.default"

	basicMap := map[string]string{
		"min_size":           "1",
		"max_size":           "4",
		"default_cooldown":   "20",
		"scaling_group_name": fmt.Sprintf("tf-testAccEssScalingGroup-%d", rand),
		"vswitch_ids.#":      "2",
		"removal_policies.#": "2",
	}

	ra := resourceAttrInit(resourceId, basicMap)
	rc := resourceCheckInit(resourceId, &v, func() interface{} {
		return &EssService{testAccProvider.Meta().(*connectivity.ApsaraStackClient)}
	})
	rac := resourceAttrCheckInit(rc, ra)

	testAccCheck := rac.resourceAttrMapUpdateSet()
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: resourceId,

		Providers:    testAccProviders,
		CheckDestroy: testAccCheckEssScalingGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEssScalingGroup(EcsInstanceCommonTestCase, rand),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(nil),
				),
			},
			{
				ResourceName:      resourceId,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccEssScalingGroupUpdateMaxSize(EcsInstanceCommonTestCase, rand),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"max_size": "5",
					}),
				),
			},

			{
				Config: testAccEssScalingGroupUpdateScalingGroupName(EcsInstanceCommonTestCase, rand),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"scaling_group_name": fmt.Sprintf("tf-testAccEssScalingGroupUpdate-%d", rand),
					}),
				),
			},
			{
				Config: testAccEssScalingGroupUpdateRemovalPolicies(EcsInstanceCommonTestCase, rand),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"removal_policies.#": "1",
					}),
				),
			},
			{
				Config: testAccEssScalingGroupUpdateDefaultCooldown(EcsInstanceCommonTestCase, rand),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"default_cooldown": "200",
					}),
				),
			},
			{
				Config: testAccEssScalingGroupUpdateMinSize(EcsInstanceCommonTestCase, rand),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"min_size": "2",
					}),
				),
			},
			{
				Config: testAccEssScalingGroupModifyVSwitchIds(EcsInstanceCommonTestCase, rand),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"vswitch_ids.#": "1",
					}),
				),
			},
			{
				Config: testAccEssScalingGroup(EcsInstanceCommonTestCase, rand),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(basicMap),
				),
			},
		},
	})

}

func TestAccApsaraStackdEssScalingGroup_vpc(t *testing.T) {
	rand := acctest.RandIntRange(10000, 999999)
	var v ess.ScalingGroup
	resourceId := "apsarastack_ess_scaling_group.default"

	basicMap := map[string]string{
		"min_size":           "1",
		"max_size":           "1",
		"default_cooldown":   "20",
		"scaling_group_name": fmt.Sprintf("tf-testAccEssScalingGroup_vpc-%d", rand),
		"vswitch_ids.#":      "2",
		"removal_policies.#": "2",
	}

	ra := resourceAttrInit(resourceId, basicMap)
	rc := resourceCheckInit(resourceId, &v, func() interface{} {
		return &EssService{testAccProvider.Meta().(*connectivity.ApsaraStackClient)}
	})
	rac := resourceAttrCheckInit(rc, ra)

	testAccCheck := rac.resourceAttrMapUpdateSet()
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: resourceId,

		Providers:    testAccProviders,
		CheckDestroy: testAccCheckEssScalingGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEssScalingGroupVpc(EcsInstanceCommonTestCase, rand),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(nil),
				),
				//ExpectNonEmptyPlan: true,
			},
			{
				ResourceName:      resourceId,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccEssScalingGroupVpcUpdateMaxSize(EcsInstanceCommonTestCase, rand),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"max_size": "2",
					}),
				),
			},
			{
				Config: testAccEssScalingGroupVpcUpdateScalingGroupName(EcsInstanceCommonTestCase, rand),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"scaling_group_name": fmt.Sprintf("tf-testAccEssScalingGroupUpdate-%d", rand),
					}),
				),
			},
			{
				Config: testAccEssScalingGroupVpcUpdateRemovalPolicies(EcsInstanceCommonTestCase, rand),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"removal_policies.#": "1",
					}),
				),
			},
			{
				Config: testAccEssScalingGroupVpcUpdateDefaultCooldown(EcsInstanceCommonTestCase, rand),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"default_cooldown": "200",
					}),
				),
			},
			{
				Config: testAccEssScalingGroupVpcUpdateMinSize(EcsInstanceCommonTestCase, rand),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"min_size": "2",
					}),
				),
			},
			{
				Config: testAccEssScalingGroupVpc(EcsInstanceCommonTestCase, rand),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(basicMap),
				),
			},
		},
	})

}

func TestAccApsaraStackEssScalingGroup_slb(t *testing.T) {
	var v ess.ScalingGroup
	var slb *slb.DescribeLoadBalancerAttributeResponse
	rand := acctest.RandIntRange(10000, 999999)
	resourceId := "apsarastack_ess_scaling_group.default"

	basicMap := map[string]string{
		"min_size":           "1",
		"max_size":           "1",
		"default_cooldown":   "300",
		"scaling_group_name": fmt.Sprintf("tf-testAccEssScalingGroup_slb-%d", rand),
		"vswitch_ids.#":      "1",
		"removal_policies.#": "2",
		"loadbalancer_ids.#": "0",
	}

	ra := resourceAttrInit(resourceId, basicMap)
	rc := resourceCheckInit(resourceId, &v, func() interface{} {
		return &EssService{testAccProvider.Meta().(*connectivity.ApsaraStackClient)}
	})
	rcSlb0 := resourceCheckInit("apsarastack_slb.default.0", &slb, func() interface{} {
		return &SlbService{testAccProvider.Meta().(*connectivity.ApsaraStackClient)}
	})
	rcSlb1 := resourceCheckInit("apsarastack_slb.default.1", &slb, func() interface{} {
		return &SlbService{testAccProvider.Meta().(*connectivity.ApsaraStackClient)}
	})
	rac := resourceAttrCheckInit(rc, ra)
	testAccCheck := rac.resourceAttrMapUpdateSet()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		// module name
		IDRefreshName: resourceId,

		Providers:    testAccProviders,
		CheckDestroy: testAccCheckEssScalingGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEssScalingGroupSlbempty(EcsInstanceCommonTestCase, rand),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(nil),
				),
				//ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccEssScalingGroupSlb(EcsInstanceCommonTestCase, rand),
				Check: resource.ComposeTestCheckFunc(
					rcSlb0.checkResourceExists(),
					rcSlb1.checkResourceExists(),
					testAccCheck(map[string]string{
						"loadbalancer_ids.#": "2",
					}),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccEssScalingGroupSlbDetach(EcsInstanceCommonTestCase, rand),
				Check: resource.ComposeTestCheckFunc(
					rcSlb0.checkResourceExists(),
					testAccCheck(map[string]string{
						"loadbalancer_ids.#": "1",
					}),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccEssScalingGroupSlbUpdateMaxSize(EcsInstanceCommonTestCase, rand),
				Check: resource.ComposeTestCheckFunc(
					rcSlb0.checkResourceExists(),
					rcSlb1.checkResourceExists(),
					testAccCheck(map[string]string{
						"max_size":           "2",
						"loadbalancer_ids.#": "2",
					}),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccEssScalingGroupSlbUpdateScalingGroupName(EcsInstanceCommonTestCase, rand),
				Check: resource.ComposeTestCheckFunc(
					rcSlb0.checkResourceExists(),
					rcSlb1.checkResourceExists(),
					testAccCheck(map[string]string{
						"scaling_group_name": fmt.Sprintf("tf-testAccEssScalingGroupUpdate-%d", rand),
					}),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccEssScalingGroupSlbUpdateRemovalPolicies(EcsInstanceCommonTestCase, rand),
				Check: resource.ComposeTestCheckFunc(
					rcSlb0.checkResourceExists(),
					rcSlb1.checkResourceExists(),
					testAccCheck(map[string]string{
						"removal_policies.#": "1",
					}),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccEssScalingGroupSlbUpdateDefaultCooldown(EcsInstanceCommonTestCase, rand),
				Check: resource.ComposeTestCheckFunc(
					rcSlb0.checkResourceExists(),
					rcSlb1.checkResourceExists(),
					testAccCheck(map[string]string{
						"default_cooldown": "200",
					}),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccEssScalingGroupSlbUpdateMinSize(EcsInstanceCommonTestCase, rand),
				Check: resource.ComposeTestCheckFunc(
					rcSlb0.checkResourceExists(),
					rcSlb1.checkResourceExists(),
					testAccCheck(map[string]string{
						"min_size": "2",
					}),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccEssScalingGroupSlbempty(EcsInstanceCommonTestCase, rand),
				Check: resource.ComposeTestCheckFunc(
					testAccCheck(map[string]string{
						"loadbalancer_ids.#": "0",
						"min_size":           "1",
						"max_size":           "1",
						"default_cooldown":   "300",
						"removal_policies.#": "2",
						"scaling_group_name": fmt.Sprintf("tf-testAccEssScalingGroup_slb-%d", rand),
					}),
				),
				//ExpectNonEmptyPlan: true,
			},
		},
	})

}

func testAccCheckEssScalingGroupDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*connectivity.ApsaraStackClient)
	essService := EssService{client}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "apsarastack_ess_scaling_group" {
			continue
		}

		if _, err := essService.DescribeEssScalingGroup(rs.Primary.ID); err != nil {
			if NotFoundError(err) {
				continue
			}
			return WrapError(err)
		}
		return WrapError(fmt.Errorf("Scaling group %s still exists.", rs.Primary.ID))
	}

	return nil
}

func testAccEssScalingGroup(common string, rand int) string {
	return fmt.Sprintf(`
	%s
	variable "name" {
		default = "tf-testAccEssScalingGroup-%d"
	}
	
	resource "apsarastack_vswitch" "default2" {
		  vpc_id = "${apsarastack_vpc.default.id}"
		  cidr_block = "172.16.1.0/24"
		  availability_zone = "${data.apsarastack_zones.default.zones.0.id}"
		  name = "${var.name}-bar"
	}
	
	resource "apsarastack_ess_scaling_group" "default" {
		min_size = 1
		max_size = 4
		scaling_group_name = "${var.name}"
		default_cooldown = 20
		vswitch_ids = ["${apsarastack_vswitch.default.id}", "${apsarastack_vswitch.default2.id}"]
		removal_policies = ["OldestInstance", "NewestInstance"]
	}`, common, rand)
}

func testAccEssScalingGroupUpdateMaxSize(common string, rand int) string {
	return fmt.Sprintf(`
	%s
	variable "name" {
		default = "tf-testAccEssScalingGroup-%d"
	}
	
	resource "apsarastack_vswitch" "default2" {
		  vpc_id = "${apsarastack_vpc.default.id}"
		  cidr_block = "172.16.1.0/24"
		  availability_zone = "${data.apsarastack_zones.default.zones.0.id}"
		  name = "${var.name}-bar"
	}
	
	resource "apsarastack_ess_scaling_group" "default" {
		min_size = 1
		max_size = 5
		scaling_group_name = "${var.name}"
		default_cooldown = 20
		vswitch_ids = ["${apsarastack_vswitch.default.id}", "${apsarastack_vswitch.default2.id}"]
		removal_policies = ["OldestInstance", "NewestInstance"]
	}`, common, rand)
}

func testAccEssScalingGroupUpdateScalingGroupName(common string, rand int) string {
	return fmt.Sprintf(`
	%s
	variable "name" {
		default = "tf-testAccEssScalingGroupUpdate-%d"
	}
	
	resource "apsarastack_vswitch" "default2" {
		  vpc_id = "${apsarastack_vpc.default.id}"
		  cidr_block = "172.16.1.0/24"
		  availability_zone = "${data.apsarastack_zones.default.zones.0.id}"
		  name = "${var.name}-bar"
	}
	
	resource "apsarastack_ess_scaling_group" "default" {
		min_size = 1
		max_size = 5
		scaling_group_name = "${var.name}"
		default_cooldown = 20
		vswitch_ids = ["${apsarastack_vswitch.default.id}", "${apsarastack_vswitch.default2.id}"]
		removal_policies = ["OldestInstance", "NewestInstance"]
	}`, common, rand)
}

func testAccEssScalingGroupUpdateRemovalPolicies(common string, rand int) string {
	return fmt.Sprintf(`
	%s
	variable "name" {
		default = "tf-testAccEssScalingGroupUpdate-%d"
	}
	
	resource "apsarastack_vswitch" "default2" {
		  vpc_id = "${apsarastack_vpc.default.id}"
		  cidr_block = "172.16.1.0/24"
		  availability_zone = "${data.apsarastack_zones.default.zones.0.id}"
		  name = "${var.name}-bar"
	}
	
	resource "apsarastack_ess_scaling_group" "default" {
		min_size = 1
		max_size = 5
		scaling_group_name = "${var.name}"
		default_cooldown = 20
		vswitch_ids = ["${apsarastack_vswitch.default.id}", "${apsarastack_vswitch.default2.id}"]
		removal_policies = ["OldestInstance"]
	}`, common, rand)
}

func testAccEssScalingGroupUpdateDefaultCooldown(common string, rand int) string {
	return fmt.Sprintf(`
	%s
	variable "name" {
		default = "tf-testAccEssScalingGroupUpdate-%d"
	}
	
	resource "apsarastack_vswitch" "default2" {
		  vpc_id = "${apsarastack_vpc.default.id}"
		  cidr_block = "172.16.1.0/24"
		  availability_zone = "${data.apsarastack_zones.default.zones.0.id}"
		  name = "${var.name}-bar"
	}
	
	resource "apsarastack_ess_scaling_group" "default" {
		min_size = 1
		max_size = 5
		scaling_group_name = "${var.name}"
		default_cooldown = 200
		vswitch_ids = ["${apsarastack_vswitch.default.id}", "${apsarastack_vswitch.default2.id}"]
		removal_policies = ["OldestInstance"]
	}`, common, rand)
}

func testAccEssScalingGroupUpdateMinSize(common string, rand int) string {
	return fmt.Sprintf(`
	%s
	variable "name" {
		default = "tf-testAccEssScalingGroupUpdate-%d"
	}
	
	resource "apsarastack_vswitch" "default2" {
		  vpc_id = "${apsarastack_vpc.default.id}"
		  cidr_block = "172.16.1.0/24"
		  availability_zone = "${data.apsarastack_zones.default.zones.0.id}"
		  name = "${var.name}-bar"
	}
	
	resource "apsarastack_ess_scaling_group" "default" {
		min_size = 2
		max_size = 5
		scaling_group_name = "${var.name}"
		default_cooldown = 200
		vswitch_ids = ["${apsarastack_vswitch.default.id}", "${apsarastack_vswitch.default2.id}"]
		removal_policies = ["OldestInstance"]
	}`, common, rand)
}
func testAccEssScalingGroupVpc(common string, rand int) string {
	return fmt.Sprintf(`
	%s
	variable "name" {
		default = "tf-testAccEssScalingGroup_vpc-%d"
	}
	
	resource "apsarastack_vswitch" "default2" {
		  vpc_id = "${apsarastack_vpc.default.id}"
		  cidr_block = "172.16.1.0/24"
		  availability_zone = "${data.apsarastack_zones.default.zones.0.id}"
		  name = "${var.name}-bar"
	}
	
	resource "apsarastack_ess_scaling_group" "default" {
		min_size = 1
		max_size = 1
		scaling_group_name = "${var.name}"
		default_cooldown = 20
		vswitch_ids = ["${apsarastack_vswitch.default.id}", "${apsarastack_vswitch.default2.id}"]
		removal_policies = ["OldestInstance", "NewestInstance"]
	}`, common, rand)
}

func testAccEssScalingGroupVpcUpdateMaxSize(common string, rand int) string {
	return fmt.Sprintf(`
	%s
	variable "name" {
		default = "tf-testAccEssScalingGroup_vpc-%d"
	}
	
	resource "apsarastack_vswitch" "default2" {
		  vpc_id = "${apsarastack_vpc.default.id}"
		  cidr_block = "172.16.1.0/24"
		  availability_zone = "${data.apsarastack_zones.default.zones.0.id}"
		  name = "${var.name}-bar"
	}
	
	resource "apsarastack_ess_scaling_group" "default" {
		min_size = 1
		max_size = 2
		scaling_group_name = "${var.name}"
		default_cooldown = 20
		vswitch_ids = ["${apsarastack_vswitch.default.id}", "${apsarastack_vswitch.default2.id}"]
		removal_policies = ["OldestInstance", "NewestInstance"]
	}`, common, rand)
}

func testAccEssScalingGroupVpcUpdateScalingGroupName(common string, rand int) string {
	return fmt.Sprintf(`
	%s
	variable "name" {
		default = "tf-testAccEssScalingGroupUpdate-%d"
	}
	
	resource "apsarastack_vswitch" "default2" {
		  vpc_id = "${apsarastack_vpc.default.id}"
		  cidr_block = "172.16.1.0/24"
		  availability_zone = "${data.apsarastack_zones.default.zones.0.id}"
		  name = "${var.name}-bar"
	}
	
	resource "apsarastack_ess_scaling_group" "default" {
		min_size = 1
		max_size = 2
		scaling_group_name = "${var.name}"
		default_cooldown = 20
		vswitch_ids = ["${apsarastack_vswitch.default.id}", "${apsarastack_vswitch.default2.id}"]
		removal_policies = ["OldestInstance", "NewestInstance"]
	}`, common, rand)
}

func testAccEssScalingGroupVpcUpdateRemovalPolicies(common string, rand int) string {
	return fmt.Sprintf(`
	%s
	variable "name" {
		default = "tf-testAccEssScalingGroupUpdate-%d"
	}
	
	resource "apsarastack_vswitch" "default2" {
		  vpc_id = "${apsarastack_vpc.default.id}"
		  cidr_block = "172.16.1.0/24"
		  availability_zone = "${data.apsarastack_zones.default.zones.0.id}"
		  name = "${var.name}-bar"
	}
	
	resource "apsarastack_ess_scaling_group" "default" {
		min_size = 1
		max_size = 2
		scaling_group_name = "${var.name}"
		default_cooldown = 20
		vswitch_ids = ["${apsarastack_vswitch.default.id}", "${apsarastack_vswitch.default2.id}"]
		removal_policies = ["OldestInstance"]
	}`, common, rand)
}

func testAccEssScalingGroupVpcUpdateDefaultCooldown(common string, rand int) string {
	return fmt.Sprintf(`
	%s
	variable "name" {
		default = "tf-testAccEssScalingGroupUpdate-%d"
	}
	
	resource "apsarastack_vswitch" "default2" {
		  vpc_id = "${apsarastack_vpc.default.id}"
		  cidr_block = "172.16.1.0/24"
		  availability_zone = "${data.apsarastack_zones.default.zones.0.id}"
		  name = "${var.name}-bar"
	}
	
	resource "apsarastack_ess_scaling_group" "default" {
		min_size = 1
		max_size = 2
		scaling_group_name = "${var.name}"
		default_cooldown = 200
		vswitch_ids = ["${apsarastack_vswitch.default.id}", "${apsarastack_vswitch.default2.id}"]
		removal_policies = ["OldestInstance"]
	}`, common, rand)
}

func testAccEssScalingGroupVpcUpdateMinSize(common string, rand int) string {
	return fmt.Sprintf(`
	%s
	variable "name" {
		default = "tf-testAccEssScalingGroupUpdate-%d"
	}
	
	resource "apsarastack_vswitch" "default2" {
		  vpc_id = "${apsarastack_vpc.default.id}"
		  cidr_block = "172.16.1.0/24"
		  availability_zone = "${data.apsarastack_zones.default.zones.0.id}"
		  name = "${var.name}-bar"
	}
	
	resource "apsarastack_ess_scaling_group" "default" {
		min_size = 2
		max_size = 2
		scaling_group_name = "${var.name}"
		default_cooldown = 200
		vswitch_ids = ["${apsarastack_vswitch.default.id}", "${apsarastack_vswitch.default2.id}"]
		removal_policies = ["OldestInstance"]
	}`, common, rand)
}

func testAccEssScalingGroupSlb(common string, rand int) string {
	return fmt.Sprintf(`
	%s
	variable "name" {
		default = "tf-testAccEssScalingGroup_slb-%d"
	}
	
	resource "apsarastack_ess_scaling_group" "default" {
	  min_size = "1"
	  max_size = "1"
	  scaling_group_name = "${var.name}"
	  removal_policies = ["OldestInstance", "NewestInstance"]
	  vswitch_ids = ["${apsarastack_vswitch.default.id}"]
	  loadbalancer_ids = ["${apsarastack_slb.default.0.id}","${apsarastack_slb.default.1.id}"]
	  depends_on = ["apsarastack_slb_listener.default"]
	}

	resource "apsarastack_slb" "default" {
	  count=2
	  name = "${var.name}"
	  vswitch_id = "${apsarastack_vswitch.default.id}"
	}

	resource "apsarastack_slb_listener" "default" {
	  count = 2
	  load_balancer_id = "${element(apsarastack_slb.default.*.id, count.index)}"
	  backend_port = "22"
	  frontend_port = "22"
	  protocol = "http"
	  bandwidth = "10"
	  health_check_type = "http"
      health_check ="off"
	  sticky_session ="off"
	}
	`, common, rand)
}

func testAccEssScalingGroupSlbDetach(common string, rand int) string {
	return fmt.Sprintf(`
	%s
	variable "name" {
		default = "tf-testAccEssScalingGroup_slb-%d"
	}
	
	resource "apsarastack_ess_scaling_group" "default" {
	  min_size = "1"
	  max_size = "1"
	  scaling_group_name = "${var.name}"
	  removal_policies = ["OldestInstance", "NewestInstance"]
	  vswitch_ids = ["${apsarastack_vswitch.default.id}"]
	  loadbalancer_ids = ["${apsarastack_slb.default.0.id}"]
	  depends_on = ["apsarastack_slb_listener.default"]
	}

	resource "apsarastack_slb" "default" {
	  count=2
	  name = "${var.name}"
	  vswitch_id = "${apsarastack_vswitch.default.id}"
	}

	resource "apsarastack_slb_listener" "default" {
	  count = 2
	  load_balancer_id = "${element(apsarastack_slb.default.*.id, count.index)}"
	  backend_port = "22"
	  frontend_port = "22"
	  protocol = "http"
	  bandwidth = "10"
	  health_check_type = "http"
      health_check ="off"
	  sticky_session ="off"
	}
	`, common, rand)
}

func testAccEssScalingGroupSlbempty(common string, rand int) string {
	return fmt.Sprintf(`
	%s
	variable "name" {
		default = "tf-testAccEssScalingGroup_slb-%d"
	}
	
	resource "apsarastack_ess_scaling_group" "default" {
	  min_size = "1"
	  max_size = "1"
	  scaling_group_name = "${var.name}"
	  removal_policies = ["OldestInstance", "NewestInstance"]
	  vswitch_ids = ["${apsarastack_vswitch.default.id}"]
	  loadbalancer_ids = []
	}`, common, rand)
}

func testAccEssScalingGroupSlbUpdateMaxSize(common string, rand int) string {
	return fmt.Sprintf(`
	%s
	variable "name" {
		default = "tf-testAccEssScalingGroup_slb-%d"
	}
	
	resource "apsarastack_ess_scaling_group" "default" {
	  min_size = "1"
	  max_size = "2"
	  scaling_group_name = "${var.name}"
	  removal_policies = ["OldestInstance", "NewestInstance"]
	  vswitch_ids = ["${apsarastack_vswitch.default.id}"]
	  loadbalancer_ids = ["${apsarastack_slb.default.0.id}","${apsarastack_slb.default.1.id}"]
	  depends_on = ["apsarastack_slb_listener.default"]
	}

	resource "apsarastack_slb" "default" {
	  count=2
	  name = "${var.name}"
	  vswitch_id = "${apsarastack_vswitch.default.id}"
	}

	resource "apsarastack_slb_listener" "default" {
	  count = 2
	  load_balancer_id = "${element(apsarastack_slb.default.*.id, count.index)}"
	  backend_port = "22"
	  frontend_port = "22"
	  protocol = "http"
	  bandwidth = "10"
	  health_check_type = "http"
      health_check ="off"
	  sticky_session ="off"
	}
	`, common, rand)
}

func testAccEssScalingGroupSlbUpdateScalingGroupName(common string, rand int) string {
	return fmt.Sprintf(`
	%s
	variable "name" {
		default = "tf-testAccEssScalingGroupUpdate-%d"
	}
	
	resource "apsarastack_ess_scaling_group" "default" {
	  min_size = "1"
	  max_size = "2"
	  scaling_group_name = "${var.name}"
	  removal_policies = ["OldestInstance", "NewestInstance"]
	  vswitch_ids = ["${apsarastack_vswitch.default.id}"]
	  loadbalancer_ids = ["${apsarastack_slb.default.0.id}","${apsarastack_slb.default.1.id}"]
	  depends_on = ["apsarastack_slb_listener.default"]
	}

	resource "apsarastack_slb" "default" {
	  count=2
	  name = "${var.name}"
	  vswitch_id = "${apsarastack_vswitch.default.id}"
	}

	resource "apsarastack_slb_listener" "default" {
	  count = 2
	  load_balancer_id = "${element(apsarastack_slb.default.*.id, count.index)}"
	  backend_port = "22"
	  frontend_port = "22"
	  protocol = "http"
	  bandwidth = "10"
	  health_check_type = "http"
       sticky_session="off"
	  health_check= "off"
	}
	`, common, rand)
}

func testAccEssScalingGroupSlbUpdateRemovalPolicies(common string, rand int) string {
	return fmt.Sprintf(`
	%s
	variable "name" {
		default = "tf-testAccEssScalingGroupUpdate-%d"
	}
	
	resource "apsarastack_ess_scaling_group" "default" {
	  min_size = "1"
	  max_size = "2"
	  scaling_group_name = "${var.name}"
	  removal_policies = ["OldestInstance"]
	  vswitch_ids = ["${apsarastack_vswitch.default.id}"]
	  loadbalancer_ids = ["${apsarastack_slb.default.0.id}","${apsarastack_slb.default.1.id}"]
	  depends_on = ["apsarastack_slb_listener.default"]
	}

	resource "apsarastack_slb" "default" {
	  count=2
	  name = "${var.name}"
	  vswitch_id = "${apsarastack_vswitch.default.id}"
	}

	resource "apsarastack_slb_listener" "default" {
	  count = 2
	  load_balancer_id = "${element(apsarastack_slb.default.*.id, count.index)}"
	  backend_port = "22"
	  frontend_port = "22"
	  protocol = "http"
	  bandwidth = "10"
	  health_check_type = "http"
      health_check ="off"
	  sticky_session ="off"
	}
	`, common, rand)
}

func testAccEssScalingGroupSlbUpdateDefaultCooldown(common string, rand int) string {
	return fmt.Sprintf(`
	%s
	variable "name" {
		default = "tf-testAccEssScalingGroupUpdate-%d"
	}
	
	resource "apsarastack_ess_scaling_group" "default" {
	  min_size = "1"
	  max_size = "2"
      default_cooldown = 200
	  scaling_group_name = "${var.name}"
	  removal_policies = ["OldestInstance"]
	  vswitch_ids = ["${apsarastack_vswitch.default.id}"]
	  loadbalancer_ids = ["${apsarastack_slb.default.0.id}","${apsarastack_slb.default.1.id}"]
	  depends_on = ["apsarastack_slb_listener.default"]
	}

	resource "apsarastack_slb" "default" {
	  count=2
	  name = "${var.name}"
	  vswitch_id = "${apsarastack_vswitch.default.id}"
	}

	resource "apsarastack_slb_listener" "default" {
	  count = 2
	  load_balancer_id = "${element(apsarastack_slb.default.*.id, count.index)}"
	  backend_port = "22"
	  frontend_port = "22"
	  protocol = "http"
	  bandwidth = "10"
	  health_check_type = "http"
      health_check ="off"
	  sticky_session ="off"
	}
	`, common, rand)
}

func testAccEssScalingGroupSlbUpdateMinSize(common string, rand int) string {
	return fmt.Sprintf(`
	%s
	variable "name" {
		default = "tf-testAccEssScalingGroupUpdate-%d"
	}
	
	resource "apsarastack_ess_scaling_group" "default" {
	  min_size = "2"
	  max_size = "2"
      default_cooldown = 200
	  scaling_group_name = "${var.name}"
	  removal_policies = ["OldestInstance"]
	  vswitch_ids = ["${apsarastack_vswitch.default.id}"]
	  loadbalancer_ids = ["${apsarastack_slb.default.0.id}","${apsarastack_slb.default.1.id}"]
	  depends_on = ["apsarastack_slb_listener.default"]
	}

	resource "apsarastack_slb" "default" {
	  count=2
	  name = "${var.name}"
	  vswitch_id = "${apsarastack_vswitch.default.id}"
	}

	resource "apsarastack_slb_listener" "default" {
	  count = 2
	  load_balancer_id = "${element(apsarastack_slb.default.*.id, count.index)}"
	  backend_port = "22"
	  frontend_port = "22"
	  protocol = "http"
	  bandwidth = "10"
	  health_check_type = "http"
      health_check ="off"
	  sticky_session ="off"
	}
	`, common, rand)
}

func testAccEssScalingGroupModifyVSwitchIds(common string, rand int) string {
	return fmt.Sprintf(`
	%s
	variable "name" {
		default = "tf-testAccEssScalingGroupUpdate-%d"
	}
	
	resource "apsarastack_vswitch" "default2" {
		  vpc_id = "${apsarastack_vpc.default.id}"
		  cidr_block = "172.16.1.0/24"
		  availability_zone = "${data.apsarastack_zones.default.zones.0.id}"
		  name = "${var.name}-bar"
	}

	resource "apsarastack_ess_scaling_group" "default" {
		min_size = 2
		max_size = 5
		scaling_group_name = "${var.name}"
		default_cooldown = 200
		vswitch_ids = ["${apsarastack_vswitch.default2.id}"]
		removal_policies = ["OldestInstance"]
	}`, common, rand)
}
