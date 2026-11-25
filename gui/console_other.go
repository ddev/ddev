//go:build !windows

package main

// shareHiddenConsole does nothing: only Windows opens a console window for
// a console program that a GUI app starts.
func shareHiddenConsole() {}
