import createClient from "openapi-fetch";
import type { paths } from "./generated/api";

export const api = createClient<paths>({ baseUrl: window.location.origin });

export async function unwrap<T>(request: Promise<{ data?: T; error?: unknown; response: Response }>): Promise<T> {
  const { data, error, response } = await request;
  if (error || !data) {
    if (typeof error === "string") throw new Error(error);
    throw new Error(`Yêu cầu thất bại (${response.status})`);
  }
  return data;
}
