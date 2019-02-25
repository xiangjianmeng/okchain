package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/ok-chain/okchain/accounts/abi"
	//"math/big"
	"math/big"
	"reflect"
	"strconv"
)

func main() {
	const definition = `[{"constant":false,"inputs":[{"name":"a","type":"uint8[2]"},{"name":"b","type":"uint80"}],"name":"withdraw","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"}]`
	myabi, err := abi.JSON(strings.NewReader(definition))
	if err != nil {
		log.Fatalln(err)
	}

	for name, method := range myabi.Methods {
		fmt.Printf("func name :%s\n", name)
		for _, typ := range method.Inputs {
			fmt.Printf("    %s\n", typ.Type.String())
		}
	}

	var args []interface{}
	var (
		v1 uint64 = 12
		v2 uint64 = 13
		v3 uint64 = 14
		v4 uint64 = 15
		v5 uint64 = 16
		v6 uint64 = 17
	)
	arr := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(v1)), 0, 0)
	arr = reflect.Append(arr, reflect.ValueOf(v1))
	arr = reflect.Append(arr, reflect.ValueOf(v2))
	arr = reflect.Append(arr, reflect.ValueOf(v3))

	//arr11 := arr.Interface()
	arr1 := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(v4)), 0, 0)
	arr1 = reflect.Append(arr1, reflect.ValueOf(v4))
	arr1 = reflect.Append(arr1, reflect.ValueOf(v5))
	arr1 = reflect.Append(arr1, reflect.ValueOf(v6))
	fmt.Println(reflect.TypeOf(arr1.Interface()))

	arrarr := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(arr1.Interface())), 0, 10)
	arrarr = reflect.Append(arrarr, reflect.ValueOf(arr.Interface()))
	arrarr = reflect.Append(arrarr, reflect.ValueOf(arr1.Interface()))
	arrarr1 := arrarr.Interface()
	fmt.Println(arrarr1)

	n := 3
	value := uint8(0)
	var arg interface{}
	arg = value
	for i := 0; i < n; i++ {
		arr := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(arg)), 0, 0)
		arg = arr.Interface()
	}

	fmt.Println(reflect.TypeOf(arg))

	inter, err := PareseUint8Arr("", "[1 2]")
	fmt.Println(inter)

	args = append(args, inter, big.NewInt(845))
	out, err := myabi.Pack("withdraw", args...)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Printf("%x\n%x\n", out, myabi.Methods["withdraw"].Id())

}

func PareseUint8Arr(argType, arg string) (interface{}, error) {
	var (
		argSlice []interface{}
		strVal   string
		convert  = false
	)
	for i := len(arg) - 1; i >= 0; i-- {
		val := arg[i]
		fmt.Println(string(val))
		if string(val) == "]" {
			argSlice = append(argSlice, string(val))
		} else if string(val) == " " || (string(val) == "[" && convert) {
			if strVal != "" {
				strVal = reverseString(strVal)
				element, err := strconv.ParseUint(strVal, 10, 8)
				if err != nil {
					return nil, err
				}
				value := uint8(element)
				//arr := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(value)), 0, 0)
				//arr = reflect.Append(arr, reflect.ValueOf(value))
				argSlice = append(argSlice, value)
				strVal = ""
				fmt.Println(argSlice)
			}
			if string(val) == "[" && convert {
				i++
				convert = false
			}
		} else if string(val) == "[" {
			arr := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(argSlice[len(argSlice)-1])), 0, 0)
			for {
				pop := argSlice[len(argSlice)-1]
				argSlice = argSlice[:len(argSlice)-1]
				fmt.Println(pop, argSlice)
				str1, ok := pop.(string)
				fmt.Println(str1)
				if ok && str1 == "]" {
					break
				}

				arr = reflect.Append(arr, reflect.ValueOf(pop))
			}
			argSlice = append(argSlice, arr.Interface())
			fmt.Println(argSlice)
		} else {
			strVal = strVal + string(val)
			if i >= 1 && string(arg[i-1]) == "[" {
				convert = true
			}
		}
	}
	return argSlice[0], nil
}

func reverseString(s string) string {
	runes := []rune(s)

	for from, to := 0, len(runes)-1; from < to; from, to = from+1, to-1 {
		runes[from], runes[to] = runes[to], runes[from]
	}

	return string(runes)
}
