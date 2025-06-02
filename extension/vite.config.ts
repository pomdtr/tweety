import { defineConfig } from "vite";
import webExtension from "@samrum/vite-plugin-web-extension";
import path from "path";
import manifest from "./manifest"

// https://vitejs.dev/config/
export default defineConfig(() => {
  return {
    build: {
      outDir: "dist/chrome",
    },
    plugins: [
      webExtension({
        manifest
      }),
    ],
    resolve: {
      alias: {
        "~": path.resolve(__dirname, "./src"),
      },
    },
  };
});
