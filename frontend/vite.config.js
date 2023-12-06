/** @type {import('vite').UserConfig} */
export default {
    define: {
        __TWEETY_ORIGIN__: process.env.TWEETY_ORIGIN ? JSON.stringify(process.env.TWEETY_ORIGIN) : undefined,
    }
}
