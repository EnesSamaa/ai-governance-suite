import { BrowserPage, PlaywrightMcpDriver, toolSchemas } from "./index.js";

class FakePage implements BrowserPage {
  calls: string[] = [];
  async goto(url: string): Promise<void> { this.calls.push(`goto:${url}`); }
  async click(selector: string): Promise<void> { this.calls.push(`click:${selector}`); }
  async fill(selector: string, value: string): Promise<void> { this.calls.push(`fill:${selector}:${value}`); }
  async screenshot(): Promise<Uint8Array> { return new Uint8Array([1, 2, 3]); }
}
function expect(value: unknown, message: string): void { if (!value) throw new Error(message); }
export async function runUnitTests(): Promise<void> {
  const page = new FakePage(); const driver = new PlaywrightMcpDriver(page);
  await driver.invoke("navigate", { url: "https://example.test" });
  await driver.invoke("click", { selector: "button" });
  await driver.invoke("type", { selector: "input", text: "hello" });
  expect((await driver.invoke("screenshot")).pngBase64 === "AQID", "screenshot encoding failed");
  expect(page.calls.length === 3 && toolSchemas.length === 4, "tool dispatch failed");
  let rejected = false; try { await driver.invoke("navigate", { url: "http://example.test" }); } catch { rejected = true; }
  expect(rejected, "insecure navigation was accepted");
}
void runUnitTests();
