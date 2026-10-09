import { z } from "zod";
import {
  ChannelInputField,
  ChannelTextareaField,
} from "./channel-fields";

export const schema = z.object({
  type: z.literal("aliyun_sms"),
  access_key_id: z.string().min(1),
  access_key_secret: z.string().min(1),
  region_id: z.string().optional(),
  sign_name: z.string().min(1),
  template_code: z.string().min(1),
  phone_numbers: z.string().min(1),
  template_param: z.string().refine((value) => {
    if (!value) return true;
    try {
      JSON.parse(value);
      return true;
    } catch {
      return false;
    }
  }, "Template parameters must be valid JSON").optional(),
  endpoint: z.string().url().or(z.literal("")).optional(),
});

export const defaultValues = {
  type: "aliyun_sms" as const,
  access_key_id: "",
  access_key_secret: "",
  region_id: "cn-hangzhou",
  sign_name: "",
  template_code: "",
  phone_numbers: "",
  template_param: "{}",
  endpoint: "",
};

export const displayName = "Aliyun SMS";

export default function AliyunSMSForm() {
  return (
    <>
      <ChannelInputField
        name="access_key_id"
        label="notifications.form.aliyun_sms.access_key_id_label"
      />
      <ChannelInputField
        name="access_key_secret"
        label="notifications.form.aliyun_sms.access_key_secret_label"
        secret
      />
      <ChannelInputField
        name="region_id"
        label="notifications.form.aliyun_sms.region_id_label"
        description="notifications.form.aliyun_sms.region_id_description"
        placeholder="cn-hangzhou"
      />
      <ChannelInputField
        name="sign_name"
        label="notifications.form.aliyun_sms.sign_name_label"
      />
      <ChannelInputField
        name="template_code"
        label="notifications.form.aliyun_sms.template_code_label"
      />
      <ChannelInputField
        name="phone_numbers"
        label="notifications.form.aliyun_sms.phone_numbers_label"
        description="notifications.form.aliyun_sms.phone_numbers_description"
        placeholder="+15551234567"
      />
      <ChannelTextareaField
        name="template_param"
        label="notifications.form.aliyun_sms.template_param_label"
        description="notifications.form.aliyun_sms.template_param_description"
        placeholder='{"code":"1234"}'
      />
      <ChannelInputField
        name="endpoint"
        label="notifications.form.aliyun_sms.endpoint_label"
        description="notifications.form.aliyun_sms.endpoint_description"
        placeholder="Use the default Aliyun endpoint"
      />
    </>
  );
}
