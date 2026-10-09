import { z } from "zod";
import type { MonitorCreateUpdateDto, MonitorMonitorResponseDto } from "@/api";
import { generalDefaultValues, generalSchema } from "../shared/general";
import { intervalsDefaultValues, intervalsSchema } from "../shared/intervals";
import { notificationsDefaultValues, notificationsSchema } from "../shared/notifications";
import { tagsDefaultValues, tagsSchema } from "../shared/tags";

export const gameDigSchema = z.object({
  type: z.literal("gamedig"),
  game: z.string().min(1, "Game is required"),
  host: z.string().min(1, "Host is required"),
  port: z.coerce.number().int().min(1).max(65535),
  expected_name: z.string().optional(),
  expected_map: z.string().optional(),
  min_players: z.coerce.number().int().min(0).optional(),
  max_players: z.coerce.number().int().min(0).optional(),
}).merge(generalSchema)
  .merge(intervalsSchema)
  .merge(notificationsSchema)
  .merge(tagsSchema);

export type GameDigForm = z.infer<typeof gameDigSchema>;

interface GameDigConfig {
  game: string;
  host: string;
  port: number;
  expected_name?: string;
  expected_map?: string;
  min_players?: number;
  max_players?: number;
}

export const gameDigDefaultValues: GameDigForm = {
  type: "gamedig",
  game: "source",
  host: "example.com",
  port: 27015,
  expected_name: "",
  expected_map: "",
  min_players: 0,
  max_players: 0,
  ...generalDefaultValues,
  ...intervalsDefaultValues,
  ...notificationsDefaultValues,
  ...tagsDefaultValues,
};

export const deserialize = (data: MonitorMonitorResponseDto): GameDigForm => {
  let config: Partial<GameDigConfig> = {};
  try {
    config = data.config ? JSON.parse(data.config) : {};
  } catch (error) {
    console.error("Failed to parse GameDig monitor config:", error);
  }

  return {
    ...gameDigDefaultValues,
    name: data.name || gameDigDefaultValues.name,
    interval: data.interval || gameDigDefaultValues.interval,
    timeout: data.timeout || gameDigDefaultValues.timeout,
    max_retries: data.max_retries ?? gameDigDefaultValues.max_retries,
    retry_interval: data.retry_interval || gameDigDefaultValues.retry_interval,
    resend_interval: data.resend_interval ?? gameDigDefaultValues.resend_interval,
    notification_ids: data.notification_ids || [],
    tag_ids: data.tag_ids || [],
    game: config.game || gameDigDefaultValues.game,
    host: config.host || gameDigDefaultValues.host,
    port: config.port ?? gameDigDefaultValues.port,
    expected_name: config.expected_name || "",
    expected_map: config.expected_map || "",
    min_players: config.min_players ?? 0,
    max_players: config.max_players ?? 0,
  };
};

export const serialize = (form: GameDigForm): MonitorCreateUpdateDto => ({
  type: "gamedig",
  name: form.name,
  interval: form.interval,
  timeout: form.timeout,
  max_retries: form.max_retries,
  retry_interval: form.retry_interval,
  resend_interval: form.resend_interval,
  notification_ids: form.notification_ids,
  tag_ids: form.tag_ids,
  config: JSON.stringify({
    game: form.game,
    host: form.host,
    port: form.port,
    ...(form.expected_name ? { expected_name: form.expected_name } : {}),
    ...(form.expected_map ? { expected_map: form.expected_map } : {}),
    ...(form.min_players ? { min_players: form.min_players } : {}),
    ...(form.max_players ? { max_players: form.max_players } : {}),
  } satisfies GameDigConfig),
});
