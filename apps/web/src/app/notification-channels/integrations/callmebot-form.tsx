import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useLocalizedTranslation } from "@/hooks/useTranslation";
import { useFormContext } from "react-hook-form";
import { z } from "zod";
import { ChannelInputField } from "./channel-fields";

export const schema = z.object({
  type: z.literal("callmebot"),
  service: z.enum(["whatsapp", "telegram", "facebook"]),
  recipient: z.string().optional(),
  api_key: z.string().min(1),
  language: z.string().optional(),
  api_url: z.string().url().or(z.literal("")).optional(),
});

export const defaultValues = {
  type: "callmebot" as const,
  service: "whatsapp" as const,
  recipient: "",
  api_key: "",
  language: "",
  api_url: "",
};

export const displayName = "CallMeBot";

export default function CallMeBotForm() {
  const form = useFormContext();
  const { t } = useLocalizedTranslation();
  return (
    <>
      <FormField
        control={form.control}
        name="service"
        render={({ field }) => (
          <FormItem>
            <FormLabel>{t("notifications.form.callmebot.service_label")}</FormLabel>
            <Select onValueChange={field.onChange} value={field.value}>
              <FormControl>
                <SelectTrigger>
                  <SelectValue placeholder={t("notifications.form.callmebot.service_label")} />
                </SelectTrigger>
              </FormControl>
              <SelectContent>
                <SelectItem value="whatsapp">WhatsApp</SelectItem>
                <SelectItem value="telegram">Telegram</SelectItem>
                <SelectItem value="facebook">Facebook Messenger</SelectItem>
              </SelectContent>
            </Select>
            <FormDescription>{t("notifications.form.callmebot.service_description")}</FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />
      <ChannelInputField
        name="recipient"
        label="notifications.form.callmebot.recipient_label"
        description="notifications.form.callmebot.recipient_description"
        placeholder="+15551234567"
      />
      <ChannelInputField
        name="api_key"
        label="notifications.form.callmebot.api_key_label"
        description="notifications.form.callmebot.api_key_description"
        secret
      />
      <ChannelInputField
        name="language"
        label="notifications.form.callmebot.language_label"
        description="notifications.form.callmebot.language_description"
        placeholder="en-US"
      />
      <ChannelInputField
        name="api_url"
        label="notifications.form.callmebot.api_url_label"
        description="notifications.form.callmebot.api_url_description"
        placeholder="Leave empty to use the service default"
      />
    </>
  );
}
