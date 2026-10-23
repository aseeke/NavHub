//go:build !windows

package main

func openBrowser(string) {}

func runAsService() bool { return false }
