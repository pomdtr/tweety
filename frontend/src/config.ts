export type Config = {
    theme?: string
    themeDark?: string
    env: Record<string, string>
    defaultProfile: string
    profiles: Record<string, Profile>
}

export type Profile = {
    command: string
    args?: string[]
    env?: Record<string, string>
    favicon?: string
}
