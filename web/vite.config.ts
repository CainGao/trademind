import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// Vite 配置
// 开发：前端跑 5173，后端跑 7789，通过 proxy 转发 /api
// 构建：产物在 dist/，供 Go 后端 go:embed
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      "/api": { target: "http://localhost:7789", changeOrigin: true },
      "/health": { target: "http://localhost:7789", changeOrigin: true },
    },
  },
  build: {
    outDir: "dist",
    sourcemap: false,
  },
});
