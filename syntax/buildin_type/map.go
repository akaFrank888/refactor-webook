package main

func Map() {

	// 初始化的两种方法
	m1 := make(map[string]string, 8) // 8是cap

	m1 = map[string]string{
		"key1": "value1",
	}

	// 增
	m1["key_add"] = "value_add"

	// 删  需要用内置函数
	delete(m1, "key1")
	clear(m1)

	// 改
	m1["key_add"] = "value_add_update"

	// 查
	val, ok := m1["key1"]
	if ok {
		println("第一步:", val)
	}
	val = m1["key2"]
	println("第二步:", val) // 返回零值，所以才要判断ok
}
