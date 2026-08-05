package pagerduty

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccPagerDutyUserContactMethod_Basic(t *testing.T) {
	username := fmt.Sprintf("tf-%s", acctest.RandString(5))
	email := fmt.Sprintf("%s@foo.test", username)
	address := fmt.Sprintf("%s@foo.test", acctest.RandString(6))
	addressUpdated := fmt.Sprintf("%s@foo.test", acctest.RandString(6))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPagerDutyUserContactMethodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyUserContactMethodConfig(username, email, address, "Work"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyUserContactMethodExists("pagerduty_user_contact_method.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_user_contact_method.foo", "address", address),
					resource.TestCheckResourceAttr(
						"pagerduty_user_contact_method.foo", "label", "Work"),
				),
			},
			{
				Config: testAccCheckPagerDutyUserContactMethodConfig(username, email, addressUpdated, "Home"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyUserContactMethodExists("pagerduty_user_contact_method.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_user_contact_method.foo", "address", addressUpdated),
					resource.TestCheckResourceAttr(
						"pagerduty_user_contact_method.foo", "label", "Home"),
				),
			},
		},
	})
}

// TestAccPagerDutyUserContactMethod_RecreateAfterOutOfBandDelete covers
// TFPROVDEV-146: a contact method deleted directly in the PagerDuty web
// interface must be re-created by the next plan/apply, with no manual
// `terraform state rm` in between.
func TestAccPagerDutyUserContactMethod_RecreateAfterOutOfBandDelete(t *testing.T) {
	username := fmt.Sprintf("tf-%s", acctest.RandString(5))
	email := fmt.Sprintf("%s@foo.test", username)
	address := fmt.Sprintf("%s@foo.test", acctest.RandString(6))
	config := testAccCheckPagerDutyUserContactMethodConfig(username, email, address, "Work")

	var userID, deletedID string

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPagerDutyUserContactMethodDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyUserContactMethodExists("pagerduty_user_contact_method.foo"),
					testAccCapturePagerDutyUserContactMethodIDs("pagerduty_user_contact_method.foo", &userID, &deletedID),
				),
			},
			{
				// Stands in for the customer deleting the contact method in the
				// web UI while it stays in Terraform's state and configuration.
				PreConfig: func() {
					client, err := testAccProvider.Meta().(*Config).Client()
					if err != nil {
						t.Fatalf("err: %s", err)
					}
					if _, err := client.Users.DeleteContactMethod(userID, deletedID); err != nil {
						t.Fatalf("could not delete contact method %s out of band: %s", deletedID, err)
					}
				},
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyUserContactMethodExists("pagerduty_user_contact_method.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_user_contact_method.foo", "address", address),
					// A new ID proves the object was re-created rather than the
					// deleted one being reported back out of stale state.
					testAccCheckPagerDutyUserContactMethodRecreated("pagerduty_user_contact_method.foo", &deletedID),
				),
			},
		},
	})
}

func testAccCheckPagerDutyUserContactMethodDestroy(s *terraform.State) error {
	client, _ := testAccProvider.Meta().(*Config).Client()
	for _, r := range s.RootModule().Resources {
		if r.Type != "pagerduty_user_contact_method" {
			continue
		}

		if _, _, err := client.Users.GetContactMethod(r.Primary.Attributes["user_id"], r.Primary.ID); err == nil {
			return fmt.Errorf("User contact method still exists")
		}
	}
	return nil
}

func testAccCheckPagerDutyUserContactMethodExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No user contact method ID is set")
		}

		client, _ := testAccProvider.Meta().(*Config).Client()

		found, _, err := client.Users.GetContactMethod(rs.Primary.Attributes["user_id"], rs.Primary.ID)
		if err != nil {
			return err
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("User contact method not found: %v - %v", rs.Primary.ID, found)
		}

		return nil
	}
}

func testAccCapturePagerDutyUserContactMethodIDs(n string, userID, id *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		*userID = rs.Primary.Attributes["user_id"]
		*id = rs.Primary.ID

		return nil
	}
}

func testAccCheckPagerDutyUserContactMethodRecreated(n string, deletedID *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == *deletedID {
			return fmt.Errorf("Expected a re-created contact method, but state still holds the deleted ID %s", *deletedID)
		}

		return nil
	}
}

func testAccCheckPagerDutyUserContactMethodConfig(username, email, address, label string) string {
	return fmt.Sprintf(`
resource "pagerduty_user" "foo" {
  name  = "%[1]s"
  email = "%[2]s"
}

resource "pagerduty_user_contact_method" "foo" {
  user_id = pagerduty_user.foo.id
  type    = "email_contact_method"
  address = "%[3]s"
  label   = "%[4]s"
}
`, username, email, address, label)
}
