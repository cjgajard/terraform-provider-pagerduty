package pagerduty

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccPagerDutyUserContactMethod_import(t *testing.T) {
	username := fmt.Sprintf("tf-%s", acctest.RandString(5))
	email := fmt.Sprintf("%s@foo.test", username)
	address := fmt.Sprintf("%s@foo.test", acctest.RandString(6))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPagerDutyUserContactMethodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyUserContactMethodConfig(username, email, address, "Work"),
			},
			{
				ResourceName:      "pagerduty_user_contact_method.foo",
				ImportStateIdFunc: testAccCheckPagerDutyUserContactMethodID,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckPagerDutyUserContactMethodID(s *terraform.State) (string, error) {
	cm := s.RootModule().Resources["pagerduty_user_contact_method.foo"].Primary
	return fmt.Sprintf("%v:%v", cm.Attributes["user_id"], cm.ID), nil
}
