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
    // 分包策略（2026-08-16）：原先单 chunk 2.58MB，vite 每次警告。
    // 拆成三大 vendor chunk：echarts/antd/react 系，库代码变化少可长期缓存，
    // 业务代码单独成 chunk，迭代更新只需重新下载小块。
    rollupOptions: {
      output: {
        manualChunks(id: string) {
          if (id.includes("node_modules")) {
            if (id.includes("echarts") || id.includes("zrender")) return "echarts";
            if (
              id.includes("antd") ||
              id.includes("@ant-design") ||
              id.includes("rc-") ||
              id.includes("dayjs")
            )
              return "antd";
            if (
              id.includes("react") ||
              id.includes("scheduler") ||
              id.includes("axios") ||
              id.includes("zustand")
            )
              return "react-vendor";
          }
          return undefined;
        },
      },
    },
  },
});
