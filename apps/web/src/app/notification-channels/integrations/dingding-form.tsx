import { z } from "zod";
import { ChannelInputField } from "./channel-fields";

export const schema = z.object({
  type: z.literal("dingding"),
  webhook_url: z.string().url(),
  secret: z.string().optional(),
  title: z.string().optional(),
});

export const defaultValues = {
  type: "dingding" as const,
  webhook_url: "",
  secret: "",
  title: "",
};

export const displayName = "DingDing (钉钉)";

export default function DingDingForm() {
  return (
    <>
      <ChannelInputField
        name="webhook_url"
        label="notifications.form.dingding.webhook_url_label"
        description="notifications.form.dingding.webhook_url_description"
        placeholder="https://oapi.dingtalk.com/robot/send?access_token=..."
      />
      <ChannelInputField
        name="secret"
        label="notifications.form.dingding.secret_label"
        description="notifications.form.dingding.secret_description"
        secret
      />
      <ChannelInputField
        name="title"
        label="notifications.form.dingding.title_label"
        description="notifications.form.dingding.title_description"
        placeholder="Pulsewatch Alert"
      />
    </>
  );
}
