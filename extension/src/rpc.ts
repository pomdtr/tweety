export type JSONRPCRequest = {
    jsonrpc: string;
    id?: string;
    method: string;
    params: Record<string, any> | undefined;
}

type JSONRPCRequestBase<M extends string, P extends Record<string, any> | undefined = undefined> = {
    jsonrpc: "2.0";
    id?: string
    method: M;
    params?: P
}

export type RequestResizeTTY = JSONRPCRequestBase<"resize_tty", {
    tty: string;
    cols: number;
    rows: number;
}>

export type RequestCreateTTY = JSONRPCRequestBase<"create_tty", {
    command?: string;
    args?: string[];
    cols?: number;
    rows?: number;
}>

export type RequestGetXtermConfig = JSONRPCRequestBase<"get_xterm_config">


type JSONRPCResponseBase<T extends Record<string, any> = Record<string, any>> = {
    jsonrpc: "2.0";
    id: string;
    result: T;
}

export type JSONRPCError = {
    jsonrpc: "2.0";
    id: string;
    error: {
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




