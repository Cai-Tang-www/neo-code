package main

import (
	"fmt"
	"os"
	"strconv"
)

func main() {
	if len(os.Args) != 4 {
		printUsage()
		return
	}

	num1, err1 := strconv.ParseFloat(os.Args[1], 64)
	op := os.Args[2]
	num2, err2 := strconv.ParseFloat(os.Args[3], 64)

	if err1 != nil || err2 != nil {
		fmt.Println("错误: 请输入有效的数字")
		return
	}

	result, err := calculate(num1, op, num2)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}

	fmt.Printf("结果: %.2f %s %.2f = %.2f\n", num1, op, num2, result)
}

func calculate(num1 float64, op string, num2 float64) (float64, error) {
	switch op {
	case "+":
		return num1 + num2, nil
	case "-":
		return num1 - num2, nil
	case "*":
		return num1 * num2, nil
	case "/":
		if num2 == 0 {
			return 0, fmt.Errorf("除数不能为零")
		}
		return num1 / num2, nil
	default:
		return 0, fmt.Errorf("不支持的操作符: %s (支持: +, -, *, /)", op)
	}
}

func printUsage() {
	fmt.Println("Golang 四则运算器")
	fmt.Println("用法: go run calculator.go <数字1> <操作符> <数字2>")
	fmt.Println("操作符: +, -, *, /")
	fmt.Println("示例: go run calculator.go 10 + 5")
}