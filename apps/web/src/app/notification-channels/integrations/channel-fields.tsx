import { Input } from "@/components/ui/input";
import { PasswordInput } from "@/components/ui/password-input";
import { Textarea } from "@/components/ui/textarea";
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { useLocalizedTranslation } from "@/hooks/useTranslation";
import { useFormContext } from "react-hook-form";
import type { FieldValues, Path } from "react-hook-form";

type ChannelFieldProps = {
  name: string;
  label: string;
  description?: string;
  placeholder?: string;
  secret?: boolean;
};

export function ChannelInputField({
  name,
  label,
  description,
  placeholder,
  secret = false,
}: ChannelFieldProps) {
  const form = useFormContext<FieldValues>();
  const { t } = useLocalizedTranslation();
  return (
    <FormField
      control={form.control}
      name={name as Path<FieldValues>}
      render={({ field }) => (
        <FormItem>
          <FormLabel>{t(label)}</FormLabel>
          <FormControl>
            {secret ? (
              <PasswordInput placeholder={placeholder} {...field} />
            ) : (
              <Input placeholder={placeholder} {...field} />
            )}
          </FormControl>
          {description && <FormDescription>{t(description)}</FormDescription>}
          <FormMessage />
        </FormItem>
      )}
    />
  );
}

export function ChannelTextareaField({
  name,
  label,
  description,
  placeholder,
}: ChannelFieldProps) {
  const form = useFormContext<FieldValues>();
  const { t } = useLocalizedTranslation();
  return (
    <FormField
      control={form.control}
      name={name as Path<FieldValues>}
      render={({ field }) => (
        <FormItem>
          <FormLabel>{t(label)}</FormLabel>
          <FormControl>
            <Textarea
              placeholder={placeholder}
              className="min-h-[100px]"
              {...field}
            />
          </FormControl>
          {description && <FormDescription>{t(description)}</FormDescription>}
          <FormMessage />
        </FormItem>
      )}
    />
  );
}
