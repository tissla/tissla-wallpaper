package main

import "flag"

var pathPtr = flag.String("path", "", "path to resource")
var outputPtr = flag.String("output", "", "output name")
var modePtr = flag.String("mode", "fill", "scaling mode")
