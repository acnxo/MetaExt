// Package util 提供无状态的通用工具函数。
//
// 工具函数应当纯函数化、不持有全局状态，也不引用 model、sdk 与 infra，
// 避免工具包反过来成为依赖汇聚点。
package util
