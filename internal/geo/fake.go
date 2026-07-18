package geo

import "context"

// FakeLocator is a hand-rolled Locator for tests.
type FakeLocator struct {
	Loc Location
	Err error
}

// Detect returns the configured Location or error.
func (f FakeLocator) Detect(_ context.Context) (Location, error) {
	return f.Loc, f.Err
}

// FakeGeocoder is a hand-rolled Geocoder for tests.
type FakeGeocoder struct {
	Loc Location
	Err error
}

// Geocode returns the configured Location or error.
func (f FakeGeocoder) Geocode(_ context.Context, _ string) (Location, error) {
	return f.Loc, f.Err
}

// Compile-time interface checks.
var (
	_ Locator  = (*FakeLocator)(nil)
	_ Geocoder = (*FakeGeocoder)(nil)
)
