import { z } from "zod";
import type { MonitorCreateUpdateDto, MonitorMonitorResponseDto } from "@/api";
import { generalDefaultValues, generalSchema } from "../shared/general";
import { intervalsDefaultValues, intervalsSchema } from "../shared/intervals";
import { notificationsDefaultValues, notificationsSchema } from "../shared/notifications";
import { tagsDefaultValues, tagsSchema } from "../shared/tags";

export const steamSchema = z.object({
  type: z.literal("steam"),
  host: z.string().min(1, "Host is required"),
  port: z.coerce.number().int().min(1).max(65535),
  expected_app_id: z.coerce.number().int().min(0).max(65535).optional(),
  expected_name: z.string().optional(),
}).merge(generalSchema)
  .merge(intervalsSchema)
  .merge(notificationsSchema)
  .merge(tagsSchema);

export type SteamForm = z.infer<typeof steamSchema>;

interface SteamConfig {
  host: string;
  port: number;
  expected_app_id?: number;
  expected_name?: string;
}

export const steamDefaultValues: SteamForm = {
  type: "steam",
  host: "example.com",
  port: 27015,
  expected_app_id: 0,
  expected_name: "",
  ...generalDefaultValues,
  ...intervalsDefaultValues,
  ...notificationsDefaultValues,
  ...tagsDefaultValues,
};

export const deserialize = (data: MonitorMonitorResponseDto): SteamForm => {
  let config: Partial<SteamConfig> = {};
  try {
    config = data.config ? JSON.parse(data.config) : {};
  } catch (error) {
    console.error("Failed to parse Steam monitor config:", error);
  }

  return {
    ...steamDefaultValues,
    name: data.name || steamDefaultValues.name,
    interval: data.interval || steamDefaultValues.interval,
    timeout: data.timeout || steamDefaultValues.timeout,
    max_retries: data.max_retries ?? steamDefaultValues.max_retries,
    retry_interval: data.retry_interval || steamDefaultValues.retry_interval,
    resend_interval: data.resend_interval ?? steamDefaultValues.resend_interval,
    notification_ids: data.notification_ids || [],
    tag_ids: data.tag_ids || [],
    host: config.host || steamDefaultValues.host,
    port: config.port ?? steamDefaultValues.port,
    expected_app_id: config.expected_app_id ?? 0,
    expected_name: config.expected_name || "",
  };
};

export const serialize = (form: SteamForm): MonitorCreateUpdateDto => ({
  type: "steam",
  name: form.name,
  interval: form.interval,
  timeout: form.timeout,
  max_retries: form.max_retries,
  retry_interval: form.retry_interval,
  resend_interval: form.resend_interval,
  notification_ids: form.notification_ids,
  tag_ids: form.tag_ids,
  config: JSON.stringify({
    host: form.host,
    port: form.port,
    ...(form.expected_app_id ? { expected_app_id: form.expected_app_id } : {}),
    ...(form.expected_name ? { expected_name: form.expected_name } : {}),
  } satisfies SteamConfig),
});
