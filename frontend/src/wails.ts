export type Status = {
  state: "idle" | "starting" | "ready" | "failed" | "stopped";
  url?: string;
  message: string;
  detail?: string;
  version: string;
  dshVersion: string;
};

export type Settings = {
  proxyMode: "inherit" | "custom" | "disabled";
  proxyUrl?: string;
};

type AppBindings = {
  GetStatus(): Promise<Status>;
  Retry(): Promise<Status>;
  GetSettings(): Promise<Settings>;
  SaveSettings(settings: Settings): Promise<Status>;
  OpenLogs(): Promise<void>;
  OpenInBrowser(): Promise<void>;
};

type WailsRuntime = {
  EventsOn(name: string, callback: (...args: unknown[]) => void): () => void;
};

declare global {
  interface Window {
    go: { main: { App: AppBindings } };
    runtime: WailsRuntime;
  }
}

export const backend = (): AppBindings => window.go.main.App;

export const onStatusEvent = (
  name: "dsh:ready" | "dsh:failed" | "dsh:stopped",
  callback: (status: Status) => void,
): (() => void) =>
  window.runtime.EventsOn(name, (...args: unknown[]) => callback(args[0] as Status));

export const onCommandEvent = (
  name: "shell:open-settings" | "shell:open-help",
  callback: () => void,
): (() => void) => window.runtime.EventsOn(name, callback);
