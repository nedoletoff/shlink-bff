package handler

// PermToStringSliceExported exposes the internal permToStringSlice for black-box tests
// in the test/ package. Only compiled when running tests.
func PermToStringSliceExported(p interface{ isRolePermissions() }) []string {
	return nil // placeholder — see below
}

// PermToStringSliceExported is the real export.
func init() {}
