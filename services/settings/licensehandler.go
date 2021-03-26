package settings

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"time"

	"github.com/untangle/golang-shared/services/logger"
)

// LicenseSub type, the main engine for licenses
type LicenseSub struct {
	enabledServices map[string]bool
	licenseFilename string
	uidFile         string
	product         string
}

// LicenseInfo represents the json returned from license server
type LicenseInfo struct {
	JavaClass string    `json:"javaClass"`
	List      []License `json:"list"`
}

// License is the struct representing each individual license
type License struct {
	UID         string `json:"UID"`
	Type        string `json:"type"`
	End         int    `json:"end"`
	Start       int    `json:"start"`
	Seats       int    `json:"seats" default:"-1"`
	DisplayName string `json:"displayName"`
	Key         string `json:"key"`
	KeyVersion  int    `json:"keyVersion"`
	Name        string `json:"name"`
	JavaClass   string `json:"javaClass"`
	Valid       bool   `json:"valid,omitempty" default:"false"`
}

const (
	licenseReadRetries = 5 //Number of retries to try on reading the license file
)

// GetLicenseDefaults gets the default enabled services file, where everything is disabled
func GetLicenseDefaults(product string) map[string]bool {
	var defaults map[string]bool
	switch product {
	case "WAF":
		defaults = map[string]bool{
			"loadBalancing":    false,
			"sslCertUpload":    false,
			"advancedLogging":  false,
			"manualRuleConfig": false,
			"ruleException":    false,
		}
	}

	return defaults
}

// NewLicenseSub creates new license
func NewLicenseSub(licenseFilename string, uidFile string, product string) *LicenseSub {
	logger.Info("Starting license sub...\n")

	l := new(LicenseSub)
	l.enabledServices = GetLicenseDefaults(product)
	l.licenseFilename = licenseFilename
	l.uidFile = uidFile
	l.product = product

	return l
}

// CleanUp cleans up the contexts of the licenseSub
func (l *LicenseSub) CleanUp() {
	logger.Info("Shutting down license sub...\n")
}

// GetLicenses gets the enabled services
func (l *LicenseSub) GetLicenses() (map[string]bool, error) {
	l.enabledServices = GetLicenseDefaults(l.product)
	// read license file
	retries := licenseReadRetries
	var fileBytes []byte
	for retries > 0 {
		var readErr error
		fileBytes, readErr = ioutil.ReadFile(l.licenseFilename)
		if readErr != nil {
			retries = retries - 1
			// sleep one second, perhaps file is being written
			time.Sleep(time.Second)
		} else {
			retries = 1
			break
		}
	}

	if retries <= 0 {
		return l.enabledServices, errors.New("Failed to read license file")
	}

	// unmarshal license inforamtion
	var licenseInfo LicenseInfo
	jsonErr := json.Unmarshal(fileBytes, &licenseInfo)
	if jsonErr != nil {
		return l.enabledServices, jsonErr
	}

	l.determineEnabledServices(licenseInfo.List)

	return l.enabledServices, nil
}

func (l *LicenseSub) determineEnabledServices(licenses []License) {
	for _, license := range licenses {
		_, ok := l.enabledServices[license.Name]
		if ok {
			l.enabledServices[license.Name] = license.Valid
		} else {
			logger.Warn("Saw a license name that's unknown: %s\n", license.Name)
		}
	}
}
