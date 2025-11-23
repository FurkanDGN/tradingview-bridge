package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	jsoniter "github.com/json-iterator/go"
)

var Json = jsoniter.ConfigCompatibleWithStandardLibrary

type FlexString string

func (fs *FlexString) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*fs = FlexString(s)
		return nil
	}

	var n float64
	if err := json.Unmarshal(data, &n); err == nil {
		*fs = FlexString(fmt.Sprintf("%f", n))
		return nil
	}

	var i int64
	if err := json.Unmarshal(data, &i); err == nil {
		*fs = FlexString(fmt.Sprintf("%d", i))
		return nil
	}

	return fmt.Errorf("cannot unmarshal to FlexString")
}

func (fs *FlexString) String() string {
	return string(*fs)
}

func (fs *FlexString) Float64() float64 {
	f, err := strconv.ParseFloat(string(*fs), 64)
	if err != nil {
		return 0
	}

	return f
}

func (fs *FlexString) Int64() int64 {
	i, err := strconv.ParseInt(string(*fs), 10, 64)
	if err != nil {
		return 0
	}

	return i
}
