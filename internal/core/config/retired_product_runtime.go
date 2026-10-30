package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

var retiredProductRuntimeYAMLKeys = map[string]string{
	"processor":                          "processor",
	"worker":                             "worker",
	"listingcontrolplane":                "listingControlPlane",
	"rabbitmq":                           "rabbitmq",
	"updater":                            "updater",
	"logging":                            "logging",
	"redis":                              "redis",
	"prompts":                            "prompts",
	"featureflags":                       "featureFlags",
	"observability":                      "observability",
	"watermark":                          "watermark",
	"debug.save_publish_json":            "debug.save_publish_json",
	"listingkit.sheinsubmitdebugdumpdir": "listingkit.sheinSubmitDebugDumpDir",
	"listingkit.generationusageledgerenabled":           "listingkit.generationUsageLedgerEnabled",
	"listingkit.generationusageledgertenantids":         "listingkit.generationUsageLedgerTenantIDs",
	"aicapability.studioimageroutingmode":               "aiCapability.studioImageRoutingMode",
	"debug.productenrichmockllm":                        "debug.productEnrichMockLLM",
	"aicapability.productimagesceneenabled":             "aiCapability.productImageSceneEnabled",
	"aicapability.productimagesceneallowedtenantids":    "aiCapability.productImageSceneAllowedTenantIDs",
	"aicapability.productenrichtextenabled":             "aiCapability.productEnrichTextEnabled",
	"aicapability.productenrichtextallowedtenantids":    "aiCapability.productEnrichTextAllowedTenantIDs",
	"aicapability.productenrichvisionenabled":           "aiCapability.productEnrichVisionEnabled",
	"aicapability.productenrichvisionallowedtenantids":  "aiCapability.productEnrichVisionAllowedTenantIDs",
	"aicapability.productenrichlistingenabled":          "aiCapability.productEnrichListingEnabled",
	"aicapability.productenrichlistingallowedtenantids": "aiCapability.productEnrichListingAllowedTenantIDs",
}

var retiredProductRuntimeEnvironmentVariables = map[string]struct{}{
	"LISTINGKIT_ZITADEL_ALLOWED_TENANT_IDS":                                  {},
	"LISTINGKIT_ZITADEL_ALLOWED_USER_IDS":                                    {},
	"LISTINGKIT_ZITADEL_ALLOWED_ROLES":                                       {},
	"TASK_PROCESSOR_AI_CAPABILITY_STUDIO_IMAGE_ROUTING_MODE":                 {},
	"TASK_PROCESSOR_AICAPABILITY_STUDIOIMAGEROUTINGMODE":                     {},
	"TASK_PROCESSOR_DEBUG_PRODUCTENRICHMOCKLLM":                              {},
	"TASK_PROCESSOR_DEBUG_PRODUCTENRICH_MOCK_LLM":                            {},
	"TASK_PROCESSOR_AI_CAPABILITY_PRODUCT_IMAGE_SCENE_ENABLED":               {},
	"TASK_PROCESSOR_AI_CAPABILITY_PRODUCT_IMAGE_SCENE_ALLOWED_TENANT_IDS":    {},
	"TASK_PROCESSOR_AI_CAPABILITY_PRODUCT_ENRICH_TEXT_ENABLED":               {},
	"TASK_PROCESSOR_AI_CAPABILITY_PRODUCT_ENRICH_TEXT_ALLOWED_TENANT_IDS":    {},
	"TASK_PROCESSOR_AI_CAPABILITY_PRODUCT_ENRICH_VISION_ENABLED":             {},
	"TASK_PROCESSOR_AI_CAPABILITY_PRODUCT_ENRICH_VISION_ALLOWED_TENANT_IDS":  {},
	"TASK_PROCESSOR_AI_CAPABILITY_PRODUCT_ENRICH_LISTING_ENABLED":            {},
	"TASK_PROCESSOR_AI_CAPABILITY_PRODUCT_ENRICH_LISTING_ALLOWED_TENANT_IDS": {},
	"TASK_PROCESSOR_AICAPABILITY_PRODUCTIMAGESCENEENABLED":                   {},
	"TASK_PROCESSOR_AICAPABILITY_PRODUCTIMAGESCENEALLOWEDTENANTIDS":          {},
	"TASK_PROCESSOR_AICAPABILITY_PRODUCTENRICHTEXTENABLED":                   {},
	"TASK_PROCESSOR_AICAPABILITY_PRODUCTENRICHTEXTALLOWEDTENANTIDS":          {},
	"TASK_PROCESSOR_AICAPABILITY_PRODUCTENRICHVISIONENABLED":                 {},
	"TASK_PROCESSOR_AICAPABILITY_PRODUCTENRICHVISIONALLOWEDTENANTIDS":        {},
	"TASK_PROCESSOR_AICAPABILITY_PRODUCTENRICHLISTINGENABLED":                {},
	"TASK_PROCESSOR_AICAPABILITY_PRODUCTENRICHLISTINGALLOWEDTENANTIDS":       {},
	"TASK_PROCESSOR_AMAZON_CRAWLTIMEOUT":                                     {},
	"TASK_PROCESSOR_AMAZON_ENABLED":                                          {},
	"TASK_PROCESSOR_AMAZON_FAILUREARTIFACTS_CAPTUREHTML":                     {},
	"TASK_PROCESSOR_AMAZON_FAILUREARTIFACTS_DIRECTORY":                       {},
	"TASK_PROCESSOR_AMAZON_FAILUREARTIFACTS_ENABLED":                         {},
	"TASK_PROCESSOR_AMAZON_FAILUREARTIFACTS_MAXHTMLBYTES":                    {},
	"TASK_PROCESSOR_AMAZON_PROXYPOOL_ENABLED":                                {},
	"TASK_PROCESSOR_AMAZON_PROXYPOOL_FAILURECOOLDOWNSECONDS":                 {},
	"TASK_PROCESSOR_AMAZON_PROXYPOOL_PROXIES":                                {},
	"TASK_PROCESSOR_AMAZON_PROXYPOOL_PROXYFILE":                              {},
	"TASK_PROCESSOR_AMAZON_PROXYPOOL_STRATEGY":                               {},
	"TASK_PROCESSOR_AMAZON_QUALITYCONTROL_RETRYONVALIDATIONFAILURE":          {},
	"TASK_PROCESSOR_AMAZON_QUALITYCONTROL_VALIDATIONRETRYMAXATTEMPTS":        {},
	"TASK_PROCESSOR_AMAZON_RISKCONTROL_AUTHENTICATIONRECREATETHRESHOLD":      {},
	"TASK_PROCESSOR_AMAZON_RISKCONTROL_BROWSERCRASHRECREATETHRESHOLD":        {},
	"TASK_PROCESSOR_AMAZON_RISKCONTROL_CAPTCHARECREATETHRESHOLD":             {},
	"TASK_PROCESSOR_AMAZON_RISKCONTROL_NETWORKRECREATETHRESHOLD":             {},
	"TASK_PROCESSOR_AMAZON_RISKCONTROL_SERVERERRORRECREATETHRESHOLD":         {},
	"TASK_PROCESSOR_AMAZON_RISKCONTROL_TIMEOUTRECREATETHRESHOLD":             {},
	"TASK_PROCESSOR_AMAZON_ZIPCODES":                                         {},
	"TASK_PROCESSOR_BROWSER_BROWSERPATH":                                     {},
	"TASK_PROCESSOR_BROWSER_ENABLED":                                         {},
	"TASK_PROCESSOR_BROWSER_HEADLESS":                                        {},
	"TASK_PROCESSOR_BROWSER_PATH":                                            {},
	"TASK_PROCESSOR_BROWSER_POOLSIZE":                                        {},
	"TASK_PROCESSOR_BROWSER_POOL_SIZE":                                       {},
	"TASK_PROCESSOR_BROWSER_PROXYSERVER":                                     {},
	"TASK_PROCESSOR_BROWSER_RANDOMCONFIG_ENABLED":                            {},
	"TASK_PROCESSOR_BROWSER_RANDOMCONFIG_FINGERPRINTSTRATEGY":                {},
	"TASK_PROCESSOR_BROWSER_RANDOMCONFIG_HEALTHCHECKENABLED":                 {},
	"TASK_PROCESSOR_BROWSER_RANDOMCONFIG_MAXRETRIES":                         {},
	"TASK_PROCESSOR_BROWSER_RANDOMCONFIG_MAXUSESPERINSTANCE":                 {},
	"TASK_PROCESSOR_BROWSER_RANDOMCONFIG_PRESETNAME":                         {},
	"TASK_PROCESSOR_BROWSER_RANDOMCONFIG_STRATEGY":                           {},
	"TASK_PROCESSOR_BROWSER_USERDATADIR":                                     {},
	"TASK_PROCESSOR_BROWSER_USER_DATA_DIR":                                   {},
	"TASK_PROCESSOR_BROWSER_VIEWPORTHEIGHT":                                  {},
	"TASK_PROCESSOR_BROWSER_VIEWPORTWIDTH":                                   {},
	"TASK_PROCESSOR_PLATFORMS_ALIBABA1688_ENABLED":                           {},
	"TASK_PROCESSOR_PLATFORMS_ALIBABA1688_POOLSIZE":                          {},
	"TASK_PROCESSOR_PLATFORMS_ALIBABA1688_PROFILEROOTDIR":                    {},
	"TASK_PROCESSOR_PLATFORMS_ALIBABA1688_TIMEOUT":                           {},
	"TASK_PROCESSOR_PLATFORM_1688_ENABLED":                                   {},
}

var retiredProductRuntimeEnvironmentPrefixes = []string{
	"TASK_PROCESSOR_PRODUCTIMAGE_",
	"TASK_PROCESSOR_PRODUCT_IMAGE_",
	"PRODUCTIMAGE_",
	"PRODUCT_IMAGE_",
	"TASK_PROCESSOR_PRODUCTENRICH_",
	"TASK_PROCESSOR_PRODUCT_ENRICH_",
	"PRODUCTENRICH_",
	"PRODUCT_ENRICH_",
}

func rejectRetiredProductRuntimeConfig(v *viper.Viper) error {
	if v != nil {
		for _, root := range []string{"browser", "amazon", "platforms"} {
			if v.InConfig(root) {
				return fmt.Errorf("retired configuration key %s is not supported", root)
			}
		}
		if v.InConfig("productimage") {
			return fmt.Errorf("retired configuration key productimage is not supported; use owner-specific imageagent and listingkit settings")
		}
		for key, canonical := range retiredProductRuntimeYAMLKeys {
			if v.InConfig(key) {
				return fmt.Errorf("retired configuration key %s is not supported", canonical)
			}
		}
		for _, key := range v.AllKeys() {
			lowerKey := strings.ToLower(key)
			if v.InConfig(key) && strings.HasPrefix(lowerKey, "productimage.") {
				return fmt.Errorf("retired configuration key %s is not supported; use owner-specific imageagent and listingkit settings", key)
			}
			if canonical, retired := retiredProductRuntimeYAMLKeys[lowerKey]; retired && v.InConfig(key) {
				return fmt.Errorf("retired configuration key %s is not supported", canonical)
			}
		}
	}

	for _, assignment := range os.Environ() {
		name, _, _ := strings.Cut(assignment, "=")
		upperName := strings.ToUpper(strings.TrimSpace(name))
		if _, retired := retiredProductRuntimeEnvironmentVariables[upperName]; retired {
			return fmt.Errorf("retired environment variable %s is not supported", name)
		}
		for _, prefix := range retiredProductRuntimeEnvironmentPrefixes {
			if strings.HasPrefix(upperName, prefix) {
				return fmt.Errorf("retired environment variable %s is not supported", name)
			}
		}
	}
	return nil
}
