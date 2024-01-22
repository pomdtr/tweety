export type Config = {
  key?: string;
  theme?: string;
  themeDark?: string;
  xterm: Record<string, unknown>;
  env: Record<string, string>;
  defaultProfile: string;
  profiles: Record<string, Profile>;
};

export type Profile = {
  command: string;
  args?: string[];
  env?: Record<string, string>;
  favicon?: string;
};
