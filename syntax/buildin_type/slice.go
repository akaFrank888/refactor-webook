package main

import "fmt"

func CRUD() {

	// 初始化
	s2 := make([]int, 3, 4)                                         //直接初始化了三个元素，容量为 4 的切片
	fmt.Printf("s2: %v, len: %d, cap: %d \n", s2, len(s2), cap(s2)) // s2: [0 0 0], len: 3, cap: 4

	// 增
	s2 = append(s2, 7)                                              // 追加一个元素，没有扩容
	fmt.Printf("s2: %v, len: %d, cap: %d \n", s2, len(s2), cap(s2)) // s2: [0 0 0 7], len: 4, cap: 4

	s2 = append(s2, 8)                                              // 再追加一个元素，扩容了
	fmt.Printf("s2: %v, len: %d, cap: %d \n", s2, len(s2), cap(s2)) // s2: [0 0 0 7 8], len: 5, cap: 8

	// 删 （利用子切片的方法）
	const deleteNumCnt = 3
	s2 = s2[deleteNumCnt:]         // 删除前deleteNumCnt个元素
	s2 = s2[:len(s2)-deleteNumCnt] // 删除后deleteNumCnt个元素

	// 查
	fmt.Printf("s3[2]: %d", s2[2]) // 按照下标索引
	for i, val := range s2 {
		println(i, val)
	}

}

func ShareSlice() {
	s1 := []int{1, 2, 3, 4}
	s2 := s1[2:]
	fmt.Printf("share slice s1: %v len: %d, cap: %d \n", s1, len(s1), cap(s1))
	fmt.Printf("share slice s2: %v len: %d, cap: %d \n", s2, len(s2), cap(s2))

	s2[0] = 99

	fmt.Printf("s2[0]=99 share slice s1: %v len: %d, cap: %d \n", s1, len(s1), cap(s1))
	fmt.Printf("s2[0]=99 share slice s2: %v len: %d, cap: %d \n", s2, len(s2), cap(s2))

	s2 = append(s2, 199)
	fmt.Printf("append s2 share slice s1: %v len: %d, cap: %d \n", s1, len(s1), cap(s1))
	fmt.Printf("append s2 share slice s2: %v len: %d, cap: %d \n", s2, len(s2), cap(s2))

	s2[0] = 1999
	fmt.Printf("s2[0] = 1999 share slice s1: %v len: %d, cap: %d \n", s1, len(s1), cap(s1))
	fmt.Printf("s2[0] = 1999 share slice s2: %v len: %d, cap: %d \n", s2, len(s2), cap(s2))

	//share slice s1: [1 2 3 4] len: 4, cap: 4
	//share slice s2: [3 4] len: 2, cap: 2
	//s2[0]=99 share slice s1: [1 2 99 4] len: 4, cap: 4
	//s2[0]=99 share slice s2: [99 4] len: 2, cap: 2
	//append s2 share slice s1: [1 2 99 4] len: 4, cap: 4
	//append s2 share slice s2: [99 4 199] len: 3, cap: 4
	//s2[0] = 1999 share slice s1: [1 2 99 4] len: 4, cap: 4
	//s2[0] = 1999 share slice s2: [1999 4 199] len: 3, cap: 4
}
