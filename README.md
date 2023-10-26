# gofuzz-shim

`gofuzz-shim` is heavily inspired by [go-118-fuzz-build](https://github.com/AdamKorcz/go-118-fuzz-build), 
but is rewritten from scratch. It does not have the same algorithm for conversion of 
fuzzing-input to test-inputs, hence corpus is not reusable across these engines. 

The data-conversion algorithm in gofuzz-shim if focused on allowing libfuzzer to 
have as much control as possible over the input, and making full use of the libfuzzer instrumentation
data. 

## Status

Very much work in progress. 