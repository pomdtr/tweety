/** @type {import('vite').UserConfig} */
export default {
    base: process.env.CI ? '/tweety/' : '/',
    define: {
        __TWEETY_ORIGIN__: process.env.CI ? JSON.stringify('http://localhost:9999') : 'window.location.origin',
    }
}
