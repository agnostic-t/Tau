module github.com/agnostic-t/neutrino-vpn

go 1.26.3

require (
	github.com/agnostic-t/neutrino-core v0.0.0-20260527171504-f517b40a5d18
	github.com/agnostic-t/neutrino-lproxies v0.0.0-20260527151951-66dce49712c8
	github.com/agnostic-t/neutrino-obfs v0.0.0-20260527163634-6db777a5ae47
	github.com/agnostic-t/neutrino-transport v0.0.0-20260527165202-2eb436509aa7
)

replace github.com/agnostic-t/neutrino-core => ../neutrino-core

replace github.com/agnostic-t/neutrino-obfs => ../neutrino-obfs
