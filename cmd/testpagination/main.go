package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func main() {
	// 登录获取token
	token := getToken()
	if token == "" {
		fmt.Println("获取token失败")
		return
	}

	// 测试查询业务数据API
	testBusinessDataQuery(token)
}

func getToken() string {
	url := "http://localhost:8080/api/auth/login"
	payload := map[string]interface{}{
		"username": "admin",
		"password": "liangminchuan",
	}

	jsonData, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("登录失败: %v\n", err)
		return ""
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("登录失败: %v\n", result)
		return ""
	}

	token, ok := result["token"].(string)
	if !ok {
		fmt.Println("获取token失败")
		return ""
	}

	return token
}

func testBusinessDataQuery(token string) {
	// 测试查询业务数据
	url := "http://localhost:8080/api/business/module/test_upload"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("查询失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	fmt.Printf("状态码: %d\n", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("查询失败: %v\n", result)
		return
	}

	// 检查响应结构
	if data, ok := result["data"]; ok {
		fmt.Printf("数据列表: %v\n", data)
	} else {
		fmt.Println("响应中没有data字段")
	}

	if total, ok := result["total"]; ok {
		fmt.Printf("数据总量: %v\n", total)
	} else {
		fmt.Println("响应中没有total字段")
	}

	// 测试带分页参数的查询
	testPagination(token)
}

func testPagination(token string) {
	// 测试带分页参数的查询
	url := "http://localhost:8080/api/business/module/test_upload?page=1&pageSize=5"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("分页查询失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	fmt.Printf("\n分页查询测试:\n")
	fmt.Printf("状态码: %d\n", resp.StatusCode)
	if resp.StatusCode == http.StatusOK {
		if data, ok := result["data"]; ok {
			dataList, ok := data.([]interface{})
			if ok {
				fmt.Printf("返回数据条数: %d\n", len(dataList))
			}
		}
		if total, ok := result["total"]; ok {
			fmt.Printf("数据总量: %v\n", total)
		}
		if page, ok := result["page"]; ok {
			fmt.Printf("当前页码: %v\n", page)
		}
		if pageSize, ok := result["pageSize"]; ok {
			fmt.Printf("每页大小: %v\n", pageSize)
		}
	}
}
