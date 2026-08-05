export type ToolName = "navigate" | "click" | "type" | "screenshot";

export interface BrowserPage {
  goto(url: string): Promise<void>;
  click(selector: string): Promise<void>;
  fill(selector: string, value: string): Promise<void>;
  screenshot(): Promise<Uint8Array>;
}

export interface ToolSchema { name: ToolName; description: string; required: string[] }
export const toolSchemas: ToolSchema[] = [
  { name: "navigate", description: "Navigate the browser to an HTTPS URL.", required: ["url"] },
  { name: "click", description: "Click an accessible CSS selector.", required: ["selector"] },
  { name: "type", description: "Replace text in an input selector.", required: ["selector", "text"] },
  { name: "screenshot", description: "Capture the current page as base64 PNG.", required: [] },
];

export class PlaywrightMcpDriver {
  constructor(private readonly page: BrowserPage) {}

  async invoke(name: ToolName, args: Record<string, string> = {}): Promise<Record<string, string>> {
    switch (name) {
      case "navigate":
        this.require(args, "url");
        if (!args.url.startsWith("https://")) throw new Error("only HTTPS navigation is allowed");
        await this.page.goto(args.url); return { status: "navigated", url: args.url };
      case "click": this.require(args, "selector"); await this.page.click(args.selector); return { status: "clicked" };
      case "type": this.require(args, "selector"); this.require(args, "text"); await this.page.fill(args.selector, args.text); return { status: "typed" };
      case "screenshot": return { pngBase64: base64(await this.page.screenshot()) };
      default: throw new Error(`unknown tool: ${name satisfies never}`);
    }
  }
  private require(args: Record<string, string>, key: string): void { if (!args[key]?.trim()) throw new Error(`${key} is required`); }
}

function base64(bytes: Uint8Array): string {
  let binary = "";
  for (const value of bytes) binary += String.fromCharCode(value);
  return btoa(binary);
}
