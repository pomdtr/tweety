export type Config = {
    theme?: string
    themeDark?: string
    env: Record<string, string>
    defaultProfile: string
    profiles: Record<string, {
        command: string
        args: string[]
        env: Record<string, string>
    }>
}
