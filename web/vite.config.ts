import { defineConfig } from "npm:vite@^6";
import deno from "npm:@deno/vite-plugin";

export default defineConfig({
  plugins: [deno()],
  root: ".",
  server: {
    port: 5173,
    strictPort: true,
    host: "0.0.0.0",
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
    target: "esnext",
    minify: "esbuild",
  },
});
