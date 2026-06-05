package handler

import "unified-backend/internal/domain"

// PermToStringSliceExported exposes the internal permToStringSlice for black-box tests
// in the test/ package.
func PermToStringSliceExported(p domain.RolePermissions) []string {
	return permToStringSlice(p)
}
