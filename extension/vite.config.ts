import { defineConfig } from "vite";
import webExtension from "@samrum/vite-plugin-web-extension";
import path from "path";

// https://vitejs.dev/config/
export default defineConfig(() => {
  return {
    plugins: [
      webExtension({
        additionalInputs: {
          html: ["tty.html"],
        },
        manifest: {
          author: {
            email: "contact@pomdtr.me",
          },
          name: "tweety",
          version: "1.0.0",
          manifest_version: 3,
          action: {
            default_icon: {
              16: "icons/16.png",
              19: "icons/19.png",
              32: "icons/32.png",
              38: "icons/38.png",
            }
          },
          background: {
            service_worker: "src/worker.ts",
          },
          permissions: [
            "nativeMessaging",
            "tabs",
          ],
          host_permissions: ["*://*/*"],
          icons: {
            16: "icons/16.png",
            19: "icons/19.png",
            32: "icons/32.png",
            38: "icons/38.png",
            48: "icons/48.png",
            64: "icons/64.png",
            96: "icons/96.png",
            128: "icons/128.png",
            256: "icons/256.png",
            512: "icons/512.png",
          },
        }
      }),
    ],
    resolve: {
      alias: {
        "~": path.resolve(__dirname, "./src"),
      },
    },
  };
});
