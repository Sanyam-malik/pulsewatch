import { z } from "zod";
import { ChannelInputField } from "./channel-fields";

export const schema = z.object({
  type: z.literal("whatsapp_whapi"),
  server_url: z.string().url(),
  api_key: z.string().min(1),
  recipient: z.string().min(1),
});

export const defaultValues = {
  type: "whatsapp_whapi" as const,
  server_url: "https://gate.whapi.cloud",
  api_key: "",
  recipient: "",
};

export const displayName = "WhatsApp (Whapi)";

export default function WhapiForm() {
  return (
    <>
      <ChannelInputField
        name="server_url"
        label="notifications.form.whapi.server_url_label"
        description="notifications.form.whapi.server_url_description"
        placeholder="https://gate.whapi.cloud"
      />
      <ChannelInputField
        name="api_key"
        label="notifications.form.whapi.api_key_label"
        description="notifications.form.whapi.api_key_description"
        secret
      />
      <ChannelInputField
        name="recipient"
        label="notifications.form.whapi.recipient_label"
        description="notifications.form.whapi.recipient_description"
        placeholder="15551234567"
      />
    </>
  );
}
