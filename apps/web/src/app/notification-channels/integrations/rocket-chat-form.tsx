import { z } from "zod";
import { ChannelInputField } from "./channel-fields";

export const schema = z.object({
  type: z.literal("rocket_chat"),
  webhook_url: z.string().url(),
  username: z.string().optional(),
  channel: z.string().optional(),
  icon_emoji: z.string().optional(),
});

export const defaultValues = {
  type: "rocket_chat" as const,
  webhook_url: "",
  username: "",
  channel: "",
  icon_emoji: "",
};

export const displayName = "Rocket.Chat";

export default function RocketChatForm() {
  return (
    <>
      <ChannelInputField
        name="webhook_url"
        label="notifications.form.rocket_chat.webhook_url_label"
        description="notifications.form.rocket_chat.webhook_url_description"
        placeholder="https://chat.example.com/hooks/..."
      />
      <ChannelInputField
        name="username"
        label="notifications.form.rocket_chat.username_label"
      />
      <ChannelInputField
        name="channel"
        label="notifications.form.rocket_chat.channel_label"
        description="notifications.form.rocket_chat.channel_description"
        placeholder="#alerts"
      />
      <ChannelInputField
        name="icon_emoji"
        label="notifications.form.rocket_chat.icon_emoji_label"
        description="notifications.form.rocket_chat.icon_emoji_description"
        placeholder=":satellite:"
      />
    </>
  );
}
