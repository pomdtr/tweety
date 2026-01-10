import { defineConfig } from 'wxt';

// See https://wxt.dev/api/config.html
export default defineConfig({
    outDir: "dist",
    manifest: ({ browser }) => ({
        key: browser == "chrome" ? "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAlT6IZrhsX6Y4GNar3eVY7gDmH1mujxci6QeeaiFMvz4TrJ1VSCy4eCBC3XLBnmN4Evi//pgG8XeW9Ock4aZhLr/zQboQ8uuqv2V/MvHYRZqirghYnVIPZ8FaMiYIwCPpG/dB+PsYlpsxtb0vDEfa0RYt7uUAERBhOCIX/j47TdiuUpvARKZaoPSFZCUdgq7n4XcEv0sZtjhuXR2tD7rgqmZgu6FGlO4CvshdWcXHMmiZWssfYcHUGeJP/Zbcs0tqwk7LstT80zGtVSUu1ey7CQxKZTAaNZVglyye2rSECR52UzTIeHI92gZjsFl7tENs3Hs+lY3ReVJGhhF3ksTn0QIDAQAB" : undefined,
        name: "Tweety",
        description: "An integrated terminal for your browser",
        version: process.env.MANIFEST_VERSION || "0.0.0",
        action: {},
        permissions: [
            "tabs",
            "nativeMessaging",
            "contextMenus",
            "notifications",
            "bookmarks",
            "history",
            "scripting",
            "storage"
        ],
        host_permissions: [
            "<all_urls>"
        ],
        commands: {
            openInNewTab: {
                description: "Open in new tab"
            },
            openInNewWindow: {
                description: "Open in new window",
            }
        },
        browser_specific_settings: browser == "firefox" ? {
            gecko: {
                id: "tweety@pomdtr.me"
            }
        } : undefined,
    })
});
