package testdata

import (
	"testing"
)

func TestMocks_Instantiate(t *testing.T) {
	mocks := []interface{}{
		&MockTimeEntryRepo{},
		&MockUserRepo{},
		&MockOrgRepo{},
		&MockContractRepo{},
		&MockCustomerRepo{},
		&MockActivityRepo{},
		&MockUnitRepo{},
		&MockWorkingGroupRepo{},
		&MockInvitationRepo{},
		&MockPasswordResetRepo{},
		&MockExportRepo{},
		&MockRefreshTokenRepo{},
		&MockTokenService{},
		&MockPasswordHasher{},
	}
	for i, m := range mocks {
		if m == nil {
			t.Errorf("mock %d is nil", i)
		}
	}
}
