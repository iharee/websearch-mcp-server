package cdp

import "github.com/go-rod/rod"

// BrowserSource acquires and releases a rod Browser.
type BrowserSource interface {
	Acquire() (*rod.Browser, error)
	Release()
}
