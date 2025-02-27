export type Config = {
  theme?: string;
  themeDark?: string;
  xterm: Record<string, unknown>;
  env: Record<string, string>;
};

export type Profile = {
  command: string;
  args?: string[];
  env?: Record<string, string>;
  favicon?: string;
};
