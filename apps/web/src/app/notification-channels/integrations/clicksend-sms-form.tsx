import { z } from "zod";
import {
  ChannelInputField,
  ChannelTextareaField,
} from "./channel-fields";

export const schema = z.object({
  type: z.literal("clicksend_sms"),
  username: z.string().min(1),
  api_key: z.string().min(1),
  recipient: z.string().min(1),
  sender_id: z.string().optional(),
  api_url: z.string().url().or(z.literal("")).optional(),
  custom_message: z.string().optional(),
});

export const defaultValues = {
  type: "clicksend_sms" as const,
  username: "",
  api_key: "",
  recipient: "",
  sender_id: "",
  api_url: "",
  custom_message: "",
};

export const displayName = "ClickSend SMS";

export default function ClickSendSMSForm() {
  return (
    <>
      <ChannelInputField
        name="username"
        label="notifications.form.clicksend_sms.username_label"
        description="notifications.form.clicksend_sms.username_description"
      />
      <ChannelInputField
        name="api_key"
        label="notifications.form.clicksend_sms.api_key_label"
        secret
      />
      <ChannelInputField
        name="recipient"
        label="notifications.form.clicksend_sms.recipient_label"
        description="notifications.form.clicksend_sms.recipient_description"
        placeholder="+15551234567"
      />
      <ChannelInputField
        name="sender_id"
        label="notifications.form.clicksend_sms.sender_id_label"
        description="notifications.form.clicksend_sms.sender_id_description"
      />
      <ChannelTextareaField
        name="custom_message"
        label="notifications.form.clicksend_sms.custom_message_label"
        description="notifications.form.clicksend_sms.custom_message_description"
        placeholder="{{ monitor.name }}: {{ status }} - {{ msg }}"
      />
      <ChannelInputField
        name="api_url"
        label="notifications.form.clicksend_sms.api_url_label"
        description="notifications.form.clicksend_sms.api_url_description"
        placeholder="Use the ClickSend API default"
      />
    </>
  );
}
