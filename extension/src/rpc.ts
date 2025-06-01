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
    command?: string;
    args?: string[];
    cols?: number;
    rows?: number;
}>

export type RequestGetXtermConfig = JSONRPCRequestBase<"config.get">


type JSONRPCResponseBase<T extends Record<string, any> = Record<string, any>> = {
    jsonrpc: "2.0";
    id: string;
    result: T;
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




