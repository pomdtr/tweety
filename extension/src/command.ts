type Arg = {
  name: string;
};

type Options = {
  [key: string]: Flag;
};

type Flag = {
  type: "string" | "boolean";
  default?: string | boolean;
  description?: string;
};

class Command {
  argv: string[];
  static args: Arg[];
  static options: Options;
  static commands: Command[];

  constructor(argv: string[]) {
    this.argv = argv;
  }

  run() {}

  static run(argv: string[]) {
    const cmd = new this(argv);
    cmd.run();
  }
}
