export type JSONRPCRequest = {
    jsonrpc: string;
    id?: string;
    method: string;
    params: Record<string, any> | any[] | undefined;
}

type JSONRPCRequestBase<M extends string, P extends Record<string, any> | undefined = undefined> = {
    jsonrpc: "2.0";
    id?: string
    method: M;
    params?: P
}

export type RequestResizeTTY = JSONRPCRequestBase<"tty.resize", {
    tty: string;
    cols: number;
    rows: number;
}>

export type RequestCreateTTY = JSONRPCRequestBase<"tty.create", {
    mode: "app";
    app: string;
    args: string[];
    cwd?: string;
}>

export type RequestGetXtermConfig = JSONRPCRequestBase<"xterm.getConfig", {
    variant?: "light" | "dark";
}>;


type JSONRPCResponseBase<T extends Record<string, any> = Record<string, any>> = {
    jsonrpc: "2.0";
    id: string;
    result: T;
} | {
    jsonrpc: "2.0";
    id: string;
    error: {
        code: number;
        message: string;
        data?: Record<string, any>;
    }
}

export type JSONRPCResponse = {
    jsonrpc: "2.0";
    id: string;
    result?: Record<string, any>;
    error?: {
        code: number;
        message: string;
        data?: Record<string, any>;
    };
}

export type ResponseGetXtermConfig = JSONRPCResponseBase

export type ResponseCreateTTY = JSONRPCResponseBase<{
    id: string;
    url: string;
}>




