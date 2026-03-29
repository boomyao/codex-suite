//go:build !android

package main

/*
#include <stdlib.h>
*/
import "C"

//export CodexMobileSetAndroidDefaultRouteInterface
func CodexMobileSetAndroidDefaultRouteInterface(_ *C.char) {}

//export CodexMobileSetAndroidInterfaceSnapshot
func CodexMobileSetAndroidInterfaceSnapshot(_ *C.char) {}
