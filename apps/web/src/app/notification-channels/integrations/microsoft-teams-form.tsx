import { z } from "zod";
import { ChannelInputField } from "./channel-fields";

export const schema = z.object({
  type: z.literal("microsoft_teams"),
  webhook_url: z.string().url(),
  title: z.string().optional(),
});

export const defaultValues = {
  type: "microsoft_teams" as const,
  webhook_url: "",
  title: "",
};

export const displayName = "Microsoft Teams";

export default function MicrosoftTeamsForm() {
  return (
    <>
      <ChannelInputField
        name="webhook_url"
        label="notifications.form.microsoft_teams.webhook_url_label"
        description="notifications.form.microsoft_teams.webhook_url_description"
        placeholder="https://..."
      />
      <ChannelInputField
        name="title"
        label="notifications.form.microsoft_teams.title_label"
        description="notifications.form.microsoft_teams.title_description"
        placeholder="Pulsewatch Alert"
      />
    </>
  );
}
