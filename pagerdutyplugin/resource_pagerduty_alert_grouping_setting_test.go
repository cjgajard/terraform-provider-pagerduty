package pagerduty

import (
	"context"
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func init() {
	resource.AddTestSweepers("pagerduty_alert_grouping_setting", &resource.Sweeper{
		Name: "pagerduty_alert_grouping_setting",
		F:    testSweepAlertGroupingSetting,
	})
}

func testSweepAlertGroupingSetting(_ string) error {
	ctx := context.Background()

	resp, err := testAccProvider.client.ListAlertGroupingSettings(ctx, pagerduty.ListAlertGroupingSettingsOptions{
		Limit: 100,
	})
	if err != nil {
		return err
	}

	for _, setting := range resp.AlertGroupingSettings {
		if strings.HasPrefix(setting.Name, "test") || strings.HasPrefix(setting.Name, "tf-") {
			log.Printf("Destroying alert grouping setting %s (%s)", setting.Name, setting.ID)
			if err := testAccProvider.client.DeleteAlertGroupingSetting(ctx, setting.ID); err != nil {
				return err
			}
		}
	}

	return nil
}

func TestAccPagerDutyAlertGroupingSetting_Basic(t *testing.T) {
	ref := fmt.Sprint("tf-", acctest.RandString(5))
	name := ref
	nameUpdated := fmt.Sprint("tf-", acctest.RandString(5))

	configType := string(pagerduty.AlertGroupingSettingContentBasedType)
	config := pagerduty.AlertGroupingSettingConfigContentBased{
		TimeWindow: 0,
		Aggregate:  "all",
		Fields:     []string{"summary"},
	}
	configTypeUpdated := string(pagerduty.AlertGroupingSettingTimeType)
	configUpdated := pagerduty.AlertGroupingSettingConfigTime{
		Timeout: 60,
	}

	service0 := fmt.Sprint("tf-", acctest.RandString(5))
	service1 := fmt.Sprint("tf-", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccCheckPagerDutyAlertGroupingSettingDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyAlertGroupingSettingConfig(name, configType, config, service0),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyAlertGroupingSettingExists("pagerduty_alert_grouping_setting."+ref),
					resource.TestCheckResourceAttr("pagerduty_alert_grouping_setting."+ref, "name", name),
					resource.TestCheckResourceAttrSet("pagerduty_alert_grouping_setting."+ref, "description"),
					resource.TestCheckResourceAttr("pagerduty_alert_grouping_setting."+ref, "type", configType),
					resource.TestCheckResourceAttr("pagerduty_alert_grouping_setting."+ref, "config.time", fmt.Sprint(config.TimeWindow)),
					resource.TestCheckResourceAttr("pagerduty_alert_grouping_setting."+ref, "config.aggregate", config.Aggregate),
					resource.TestCheckResourceAttr("pagerduty_alert_grouping_setting."+ref, "config.fields.0", config.Fields[0]),
					resource.TestCheckResourceAttrSet("pagerduty_alert_grouping_setting."+ref, "services.0"),
				),
			},
			{
				Config: testAccCheckPagerDutyAlertGroupingSettingConfig(nameUpdated, configTypeUpdated, configUpdated, service0, service1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyAlertGroupingSettingExists("pagerduty_alert_grouping_setting."+ref),
					resource.TestCheckResourceAttr("pagerduty_alert_grouping_setting."+ref, "name", nameUpdated),
					resource.TestCheckResourceAttrSet("pagerduty_alert_grouping_setting."+ref, "description"),
					resource.TestCheckResourceAttr("pagerduty_alert_grouping_setting."+ref, "type", configTypeUpdated),
					resource.TestCheckResourceAttr("pagerduty_alert_grouping_setting."+ref, "config.time", fmt.Sprint(configUpdated.Timeout)),
					resource.TestCheckResourceAttrSet("pagerduty_alert_grouping_setting."+ref, "services.0"),
					resource.TestCheckResourceAttrSet("pagerduty_alert_grouping_setting."+ref, "services.1"),
				),
			},
		},
	})
}

func TestAccPagerDutyAlertGroupingSetting_ContentBased_WithTimeWindow(t *testing.T) {
	ref := fmt.Sprint("tf-", acctest.RandString(5))
	name := ref

	configType := string(pagerduty.AlertGroupingSettingContentBasedType)
	config := pagerduty.AlertGroupingSettingConfigContentBased{
		TimeWindow: 600,
		Aggregate:  "all",
		Fields:     []string{"summary"},
	}

	service0 := fmt.Sprint("tf-", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccCheckPagerDutyAlertGroupingSettingDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyAlertGroupingSettingConfig(name, configType, config, service0),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyAlertGroupingSettingExists("pagerduty_alert_grouping_setting."+ref),
					resource.TestCheckResourceAttr("pagerduty_alert_grouping_setting."+ref, "name", name),
					resource.TestCheckResourceAttrSet("pagerduty_alert_grouping_setting."+ref, "description"),
					resource.TestCheckResourceAttr("pagerduty_alert_grouping_setting."+ref, "type", configType),
					resource.TestCheckResourceAttr("pagerduty_alert_grouping_setting."+ref, "config.time", fmt.Sprint(config.TimeWindow)),
					resource.TestCheckResourceAttr("pagerduty_alert_grouping_setting."+ref, "config.aggregate", config.Aggregate),
					resource.TestCheckResourceAttr("pagerduty_alert_grouping_setting."+ref, "config.fields.0", config.Fields[0]),
					resource.TestCheckResourceAttrSet("pagerduty_alert_grouping_setting."+ref, "services.0"),
				),
			},
		},
	})
}

func TestAccPagerDutyAlertGroupingSetting_Time_WithTimeoutZero(t *testing.T) {
	ref := fmt.Sprint("tf-", acctest.RandString(5))
	name := ref

	configType := string(pagerduty.AlertGroupingSettingContentBasedType)
	config := pagerduty.AlertGroupingSettingConfigTime{Timeout: 0}

	service0 := fmt.Sprint("tf-", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccCheckPagerDutyAlertGroupingSettingDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyAlertGroupingSettingConfig(name, configType, config, service0),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyAlertGroupingSettingExists("pagerduty_alert_grouping_setting."+ref),
					resource.TestCheckResourceAttr("pagerduty_alert_grouping_setting."+ref, "name", name),
					resource.TestCheckResourceAttrSet("pagerduty_alert_grouping_setting."+ref, "description"),
					resource.TestCheckResourceAttr("pagerduty_alert_grouping_setting."+ref, "type", configType),
					resource.TestCheckResourceAttr("pagerduty_alert_grouping_setting."+ref, "config.time", fmt.Sprint(config.Timeout)),
					resource.TestCheckResourceAttrSet("pagerduty_alert_grouping_setting."+ref, "services.0"),
				),
			},
		},
	})
}

func testAccCheckPagerDutyAlertGroupingSettingDestroy(s *terraform.State) error {
	for _, r := range s.RootModule().Resources {
		if r.Type != "pagerduty_alert_grouping_setting" {
			continue
		}

		ctx := context.Background()

		if _, err := testAccProvider.client.GetAlertGroupingSetting(ctx, r.Primary.ID); err == nil {
			return fmt.Errorf("Add-on still exists")
		}

	}
	return nil
}

func testAccCheckPagerDutyAlertGroupingSettingExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ctx := context.Background()

		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No alert grouping setting ID is set")
		}

		found, err := testAccProvider.client.GetAlertGroupingSetting(ctx, rs.Primary.ID)
		if err != nil {
			return err
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("Add-on not found: %v - %v", rs.Primary.ID, found)
		}

		return nil
	}
}

func helperConfigPagerDutyAlertGroupingSettingConfig(config interface{}) string {
	switch c := config.(type) {
	case pagerduty.AlertGroupingSettingConfigContentBased:
		return fmt.Sprintf(`{
			time = "%d"
			aggregate = "%s"
			fields = ["%s"]
		}`, c.TimeWindow, c.Aggregate, strings.Join(c.Fields, `","`))
	case pagerduty.AlertGroupingSettingConfigTime:
		return fmt.Sprintf(`{
			time = "%d"
		}`, c.Timeout)
	}
	return "{}"
}

func testAccCheckPagerDutyAlertGroupingSettingConfig(name string, cfgType string, cfg interface{}, serviceNames ...string) string {
	s := `
data "pagerduty_escalation_policy" "default" {
	name = "Default"
}`

	services := []string{}
	serviceIDs := []string{}
	for _, n := range serviceNames {
		s += fmt.Sprintf(`
resource "pagerduty_service" "%s" {
	name = "%s"
	escalation_policy = data.pagerduty_escalation_policy.default.id
}`, n, n)
		services = append(services, fmt.Sprintf("pagerduty_service.%s", n))
		serviceIDs = append(serviceIDs, fmt.Sprintf("pagerduty_service.%s.id", n))
	}

	config := helperConfigPagerDutyAlertGroupingSettingConfig(cfg)
	s += fmt.Sprintf(`
resource "pagerduty_alert_grouping_setting" "%s" {
  name = "%s"
  type = "%s"
  config = %s
  services = [%s]
  depends_on = [%s]
}`, name, name, cfgType, config, strings.Join(serviceIDs, `,`), strings.Join(services, `,`))
	return s
}
