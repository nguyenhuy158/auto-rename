import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwind from "@tailwindcss/vite";

export default defineConfig({
  plugins: [react(), tailwind()],
  server: { proxy: { "/api": process.env.AUTO_RENAME_DEV_API || "http://127.0.0.1:8080", "/openapi.yaml": process.env.AUTO_RENAME_DEV_API || "http://127.0.0.1:8080" } },
});
