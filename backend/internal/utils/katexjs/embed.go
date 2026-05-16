// Package katexjs 内嵌 KaTeX 的 JavaScript 主文件，供后端服务端渲染使用。
//
// 当前版本：0.16.47（来自 npm registry 的 katex@0.16.47/dist/katex.min.js）。
// 升级流程：
//
//	curl -sL https://registry.npmjs.org/katex/-/katex-<版本>.tgz | tar -xz -C /tmp
//	cp /tmp/package/dist/katex.min.js backend/internal/utils/katexjs/katex.min.js
//	cp /tmp/package/LICENSE          backend/internal/utils/katexjs/LICENSE
//
// 升级后请同步：
//   - frontend/package.json 里 katex 的版本号
//   - 各内置主题模板里 katex CSS link 的版本号
//   - backend/internal/utils/markdown_katex.go 里的 katexVersion 常量
package katexjs

import _ "embed"

// Script 是 KaTeX 的完整 JS（已 minify），在 QuickJS 里 eval 后即可调用 katex.renderToString。
//
//go:embed katex.min.js
var Script string

// Version 是当前内嵌的 KaTeX 版本。
const Version = "0.16.47"
