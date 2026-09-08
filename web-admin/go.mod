// go.mod —— 前端目录隔离模块（占位声明）。
// 背景：web-admin/node_modules 内含第三方 npm 包自带的 Go 样例源码
//（如 flatted/golang），若不加隔离，后端在根目录执行 `go list/test ./...`
// 会把这些文件扫进 meteorx 模块。
// 借助 Go 模块规则（含独立 go.mod 的子目录构成嵌套 module，不被父模块遍历），
// 此处声明一个占位 module 即可把整个前端目录从后端 Go 模块范围中隔离。
// 本文件不影响 npm/TypeScript/Vite 等前端工具链。
module webadmin

go 1.25.0
