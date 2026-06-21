# Lingua Buddy 前端

Vue 3 + TypeScript + Vite 单页应用。完整说明见[根 README](../README.md)。

## 技术栈
- Vue 3（`<script setup>`）+ TypeScript
- Vite 6（开发服务器 + 构建）
- Pinia（状态管理）· Vue Router（hash 路由 + 守卫）· Axios（HTTP）

## 命令

```powershell
npm install
npm run dev          # 开发服务器 http://localhost:5173，已代理 /api -> http://127.0.0.1:8080
npm run build        # vue-tsc 类型检查 + 生产构建到 dist/
npm run preview      # 预览生产构建
npm run type-check   # 仅类型检查
```

## 目录

```text
src/
├── api/client.ts    # Axios 实例：注入 JWT、解包统一响应、ApiError 携带后端错误码
├── stores/auth.ts   # 登录态与用户信息
├── router/index.ts  # 路由表与登录守卫
├── layouts/         # 主布局（侧栏导航）
└── pages/           # 19 个页面（登录/首页/查词/单词学习/翻译/语音/语法/外刊/对话/作文/训练/历史…）
```

## 约定
- 所有业务请求经 `@/api/client` 的 `api.get/post/patch/delete`，自动带上 `Authorization`。
- 后端非 `OK` 的业务码（如 `NO_DUE_WORDS`、`QUESTION_TOKEN_EXPIRED`）通过 `ApiError.code` 区分处理。
- 401 由拦截器统一清除 token 并跳转登录。
