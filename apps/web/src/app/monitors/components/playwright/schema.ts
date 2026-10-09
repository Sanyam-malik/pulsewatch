import { z } from "zod";
import type { MonitorCreateUpdateDto, MonitorMonitorResponseDto } from "@/api";
import { generalDefaultValues, generalSchema } from "../shared/general";
import { intervalsDefaultValues, intervalsSchema } from "../shared/intervals";
import { notificationsDefaultValues, notificationsSchema } from "../shared/notifications";
import { tagsDefaultValues, tagsSchema } from "../shared/tags";

export const playwrightSchema = z.object({
  type: z.literal("playwright"),
  browser_ws_endpoint: z.string().url().refine(value => value.startsWith("ws://") || value.startsWith("wss://"), "Use a Chromium DevTools WebSocket URL"),
  url: z.string().url(),
  selector: z.string().optional(),
  keyword: z.string().optional(),
}).merge(generalSchema)
  .merge(intervalsSchema)
  .merge(notificationsSchema)
  .merge(tagsSchema);

export type PlaywrightForm = z.infer<typeof playwrightSchema>;

interface PlaywrightConfig {
  browser_ws_endpoint: string;
  url: string;
  selector?: string;
  keyword?: string;
}

export const playwrightDefaultValues: PlaywrightForm = {
  type: "playwright",
  browser_ws_endpoint: "",
  url: "https://example.com",
  selector: "body",
  keyword: "",
  ...generalDefaultValues,
  ...intervalsDefaultValues,
  ...notificationsDefaultValues,
  ...tagsDefaultValues,
};

export const deserialize = (data: MonitorMonitorResponseDto): PlaywrightForm => {
  let config: Partial<PlaywrightConfig> = {};
  try {
    config = data.config ? JSON.parse(data.config) : {};
  } catch (error) {
    console.error("Failed to parse Playwright monitor config:", error);
  }

  return {
    ...playwrightDefaultValues,
    name: data.name || playwrightDefaultValues.name,
    interval: data.interval || playwrightDefaultValues.interval,
    timeout: data.timeout || playwrightDefaultValues.timeout,
    max_retries: data.max_retries ?? playwrightDefaultValues.max_retries,
    retry_interval: data.retry_interval || playwrightDefaultValues.retry_interval,
    resend_interval: data.resend_interval ?? playwrightDefaultValues.resend_interval,
    notification_ids: data.notification_ids || [],
    tag_ids: data.tag_ids || [],
    browser_ws_endpoint: config.browser_ws_endpoint || "",
    url: config.url || playwrightDefaultValues.url,
    selector: config.selector || "",
    keyword: config.keyword || "",
  };
};

export const serialize = (form: PlaywrightForm): MonitorCreateUpdateDto => ({
  type: "playwright",
  name: form.name,
  interval: form.interval,
  timeout: form.timeout,
  max_retries: form.max_retries,
  retry_interval: form.retry_interval,
  resend_interval: form.resend_interval,
  notification_ids: form.notification_ids,
  tag_ids: form.tag_ids,
  config: JSON.stringify({
    browser_ws_endpoint: form.browser_ws_endpoint,
    url: form.url,
    ...(form.selector ? { selector: form.selector } : {}),
    ...(form.keyword ? { keyword: form.keyword } : {}),
  } satisfies PlaywrightConfig),
});
