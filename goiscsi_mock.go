package goiscsi

import (
	"errors"
	"fmt"
	"strconv"
)

const (

	// MockNumberOfInitiators controls the number of initiators found in mock mode
	MockNumberOfInitiators = "numberOfInitiators"
	// MockNumberOfTargets controls the number of targets found in mock mode
	MockNumberOfTargets = "numberOfTargets"
)

var (
	// GOISCSIMock is a struct controlling induced errors
	GOISCSIMock struct {
		InduceDiscoveryError bool
		InduceInitiatorError bool
		InduceLoginError     bool
		InduceLogoutError    bool
		InduceRescanError    bool
	}
)

// MockISCSI provides a mock implementation of an iscsi client
type MockISCSI struct {
	ISCSIType
}

// NewMockISCSI returns an mock ISCSI client
func NewMockISCSI(opts map[string]string) *MockISCSI {
	var iscsi MockISCSI
	iscsi = MockISCSI{
		ISCSIType: ISCSIType{
			mock:    true,
			options: opts,
		},
	}

	return &iscsi
}

func getOptionAsInt(opts map[string]string, key string) int64 {
	v, _ := strconv.ParseInt(opts[key], 10, 64)
	return v
}

func (iscsi *MockISCSI) discoverTargets(address string, login bool) ([]ISCSITarget, error) {
	if GOISCSIMock.InduceDiscoveryError {
		return []ISCSITarget{}, errors.New("discoverTargets induced error")
	}
	mockedTargets := make([]ISCSITarget, 0)
	count := getOptionAsInt(iscsi.options, MockNumberOfTargets)
	if count == 0 {
		count = 1
	}

	for idx := 0; idx < int(count); idx++ {
		tgt := fmt.Sprintf("%05d", idx)
		mockedTargets = append(mockedTargets,
			ISCSITarget{
				Portal:   address + ":3260",
				GroupTag: "0",
				Target:   "iqn.1992-04.com.mock:600009700bcbb70e32870174000" + tgt,
			})
	}

	// send back a slice of targets
	return mockedTargets, nil
}

func (iscsi *MockISCSI) getInitiators(filename string) ([]string, error) {

	if GOISCSIMock.InduceInitiatorError {
		return []string{}, errors.New("getInitiators induced error")
	}

	mockedInitiators := make([]string, 0)
	count := getOptionAsInt(iscsi.options, MockNumberOfInitiators)
	if count == 0 {
		count = 1
	}

	for idx := 0; idx < int(count); idx++ {
		init := fmt.Sprintf("%05d", idx)
		mockedInitiators = append(mockedInitiators,
			"iqn.1993-08.com.mock:01:00000000"+init)
	}
	return mockedInitiators, nil
}

func (iscsi *MockISCSI) performLogin(target ISCSITarget) error {

	if GOISCSIMock.InduceLoginError {
		return errors.New("iSCSI Login induced error")
	}

	return nil
}

func (iscsi *MockISCSI) performLogout(target ISCSITarget) error {

	if GOISCSIMock.InduceLogoutError {
		return errors.New("iSCSI Logout induced error")
	}

	return nil
}

func (iscsi *MockISCSI) performRescan() error {

	if GOISCSIMock.InduceRescanError {
		return errors.New("iSCSI Rescan induced error")
	}

	return nil
}

// ====================================================================
// Architecture agnostic code for the mock implementation

// DiscoverTargets runs an iSCSI discovery and returns a list of targets.
func (iscsi *MockISCSI) DiscoverTargets(address string, login bool) ([]ISCSITarget, error) {
	return iscsi.discoverTargets(address, login)
}

// GetInitiators returns a list of initiators on the local system.
func (iscsi *MockISCSI) GetInitiators(filename string) ([]string, error) {
	return iscsi.getInitiators(filename)
}

// PerformLogin will attempt to log into an iSCSI target
func (iscsi *MockISCSI) PerformLogin(target ISCSITarget) error {
	return iscsi.performLogin(target)
}

// PerformLogout will attempt to log out of an iSCSI target
func (iscsi *MockISCSI) PerformLogout(target ISCSITarget) error {
	return iscsi.performLogout(target)
}

// PerformRescan will will rescan targets known to current sessions
func (iscsi *MockISCSI) PerformRescan() error {
	return iscsi.performRescan()
}
