package main

// Ripped from https://github.com/kelseyhightower/confd/blob/34a6ce889/resource/template/template_funcs.go
// since newFuncMap isn't exported. Also includes IsFileExist() from
// https://github.com/kelseyhightower/confd/blob/34a6ce889/util/util.go so we
// don't have to pull in the entire module.
// Also removes the reliance on github.com/kelseyhightower/memkv since it only used the KVPair struct.

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

func newFuncMap() map[string]interface{} {
	m := make(map[string]interface{})
	m["base"] = path.Base
	m["split"] = strings.Split
	m["json"] = unmarshalJSONObject
	m["jsonArray"] = unmarshalJSONArray
	m["dir"] = path.Dir
	m["map"] = createMap
	m["getenv"] = getenv
	m["join"] = strings.Join
	m["datetime"] = time.Now
	m["toUpper"] = strings.ToUpper
	m["toLower"] = strings.ToLower
	m["contains"] = strings.Contains
	m["replace"] = strings.Replace
	m["trimSuffix"] = strings.TrimSuffix
	m["lookupIP"] = lookupIP
	m["lookupIPV4"] = lookupIPV4
	m["lookupIPV6"] = lookupIPV6
	m["lookupSRV"] = lookupSRV
	m["fileExists"] = isFileExist
	m["base64Encode"] = base64Encode
	m["base64Decode"] = base64Decode
	m["parseBool"] = strconv.ParseBool
	m["reverse"] = reverse
	m["sortByLength"] = sortByLength
	m["sortKVByLength"] = sortKVByLength
	m["add"] = func(a, b int) int { return a + b }
	m["sub"] = func(a, b int) int { return a - b }
	m["div"] = func(a, b int) int { return a / b }
	m["mod"] = func(a, b int) int { return a % b }
	m["mul"] = func(a, b int) int { return a * b }
	m["seq"] = seq
	m["atoi"] = strconv.Atoi
	return m
}

func addFuncs(out, in map[string]interface{}) {
	for name, fn := range in {
		out[name] = fn
	}
}

// Seq creates a sequence of integers. It's named and used as GNU's seq.
// Seq takes the first and the last element as arguments. So Seq(3, 5) will generate [3,4,5]
func seq(first, last int) []int {
	var arr []int
	for i := first; i <= last; i++ {
		arr = append(arr, i)
	}
	return arr
}

type byLengthKV []kvPair

func (s byLengthKV) Len() int {
	return len(s)
}

func (s byLengthKV) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byLengthKV) Less(i, j int) bool {
	return len(s[i].Key) < len(s[j].Key)
}

func sortKVByLength(values []kvPair) []kvPair {
	sort.Sort(byLengthKV(values))
	return values
}

type byLength []string

func (s byLength) Len() int {
	return len(s)
}
func (s byLength) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s byLength) Less(i, j int) bool {
	return len(s[i]) < len(s[j])
}

func sortByLength(values []string) []string {
	sort.Sort(byLength(values))
	return values
}

//Reverse returns the array in reversed order
//works with []string and []kvPair
func reverse(values interface{}) interface{} {
	switch values.(type) {
	case []string:
		v := values.([]string)
		for left, right := 0, len(v)-1; left < right; left, right = left+1, right-1 {
			v[left], v[right] = v[right], v[left]
		}
	case []kvPair:
		v := values.([]kvPair)
		for left, right := 0, len(v)-1; left < right; left, right = left+1, right-1 {
			v[left], v[right] = v[right], v[left]
		}
	}
	return values
}

// Getenv retrieves the value of the environment variable named by the key.
// It returns the value, which will the default value if the variable is not present.
// If no default value was given - returns "".
func getenv(key string, v ...string) string {
	defaultValue := ""
	if len(v) > 0 {
		defaultValue = v[0]
	}

	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// CreateMap creates a key-value map of string -> interface{}
// The i'th is the key and the i+1 is the value
func createMap(values ...interface{}) (map[string]interface{}, error) {
	if len(values)%2 != 0 {
		return nil, errors.New("invalid map call")
	}
	dict := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, errors.New("map keys must be strings")
		}
		dict[key] = values[i+1]
	}
	return dict, nil
}

func unmarshalJSONObject(data string) (map[string]interface{}, error) {
	var ret map[string]interface{}
	err := json.Unmarshal([]byte(data), &ret)
	return ret, err
}

func unmarshalJSONArray(data string) ([]interface{}, error) {
	var ret []interface{}
	err := json.Unmarshal([]byte(data), &ret)
	return ret, err
}

func lookupIP(data string) []string {
	ips, err := net.LookupIP(data)
	if err != nil {
		return nil
	}
	// "Cast" IPs into strings and sort the array
	ipStrings := make([]string, len(ips))

	for i, ip := range ips {
		ipStrings[i] = ip.String()
	}
	sort.Strings(ipStrings)
	return ipStrings
}

func lookupIPV6(data string) []string {
	var addresses []string
	for _, ip := range lookupIP(data) {
		if strings.Contains(ip, ":") {
			addresses = append(addresses, ip)
		}
	}
	return addresses
}

func lookupIPV4(data string) []string {
	var addresses []string
	for _, ip := range lookupIP(data) {
		if strings.Contains(ip, ".") {
			addresses = append(addresses, ip)
		}
	}
	return addresses
}

type sortSRV []*net.SRV

func (s sortSRV) Len() int {
	return len(s)
}

func (s sortSRV) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s sortSRV) Less(i, j int) bool {
	str1 := fmt.Sprintf("%s%d%d%d", s[i].Target, s[i].Port, s[i].Priority, s[i].Weight)
	str2 := fmt.Sprintf("%s%d%d%d", s[j].Target, s[j].Port, s[j].Priority, s[j].Weight)
	return str1 < str2
}

func lookupSRV(service, proto, name string) []*net.SRV {
	_, addrs, err := net.LookupSRV(service, proto, name)
	if err != nil {
		return []*net.SRV{}
	}
	sort.Sort(sortSRV(addrs))
	return addrs
}

func base64Encode(data string) string {
	return base64.StdEncoding.EncodeToString([]byte(data))
}

func base64Decode(data string) (string, error) {
	s, err := base64.StdEncoding.DecodeString(data)
	return string(s), err
}

// From https://github.com/kelseyhightower/confd/blob/34a6ce889/util/util.go so we don't have to pull in the entire module.
func isFileExist(fpath string) bool {
	if _, err := os.Stat(fpath); os.IsNotExist(err) {
		return false
	}
	return true
}
