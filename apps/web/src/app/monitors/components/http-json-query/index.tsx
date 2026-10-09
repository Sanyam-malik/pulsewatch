import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import Advanced from "../http/advanced";
import Authentication from "../http/authentication";
import HttpOptions from "../http/options";
import { Separator } from "@/components/ui/separator";
import { Card, CardContent } from "@/components/ui/card";
import Notifications from "../shared/notifications";
import Proxies from "../shared/proxies";
import Intervals from "../shared/intervals";
import General from "../shared/general";
import Tags from "../shared/tags";
import { useMonitorFormContext } from "../../context/monitor-form-context";
import { Button } from "@/components/ui/button";
import { Loader2 } from "lucide-react";
import type { HttpJsonQueryForm } from "./schema";
import { deserialize, serialize } from "./schema";
import { useEffect } from "react";
import { useLocalizedTranslation } from "@/hooks/useTranslation";
import { Plus, Trash2 } from "lucide-react";
import { useFieldArray, useFormContext } from "react-hook-form";

const HttpJsonQuery = () => {
  const { t } = useLocalizedTranslation();
  const conditionForm = useFormContext<HttpJsonQueryForm>();
  const { fields, append, remove } = useFieldArray({
    control: conditionForm.control,
    name: "conditions",
  });
  const {
    form,
    setNotifierSheetOpen,
    setProxySheetOpen,
    isPending,
    mode,
    createMonitorMutation,
    editMonitorMutation,
    monitorId,
    monitor,
  } = useMonitorFormContext();

  const onSubmit = (data: HttpJsonQueryForm) => {
    const payload = serialize(data);

    if (mode === "create") {
      createMonitorMutation.mutate({
        body: {
          ...payload,
          active: true,
        },
      });
    } else {
      editMonitorMutation.mutate({
        path: {
          id: monitorId!,
        },
        body: {
          ...payload,
          active: monitor?.data?.active,
        },
      });
    }
  };

  // Reset form with monitor data in edit mode
  useEffect(() => {
    if (mode === "edit" && monitor?.data) {
      const parsedConfig = deserialize(monitor.data);
      form.reset(parsedConfig)
    }
  }, [form, monitor, mode]);

  return (
    <Form {...form}>
      <form
        onSubmit={form.handleSubmit((data) => onSubmit(data as HttpJsonQueryForm))}
        className="space-y-6 max-w-[600px]"
      >
        <Card>
          <CardContent className="space-y-4">
            <General />
          </CardContent>
        </Card>

        <Card>
          <CardContent className="space-y-4">
            <FormField
              control={form.control}
              name="url"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>URL</FormLabel>
                  <FormControl>
                    <Input placeholder="https://" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </CardContent>
        </Card>

        <Card>
          <CardContent className="space-y-4">
            <h4 className="text-lg font-semibold">{t("monitors.form.http_json_query.title")}</h4>
            <div className="text-sm text-muted-foreground mb-4">
              Add one or more response rules. Rules are combined with the selected AND/OR operator. JSON rules use GJSON paths; an empty JSON path is allowed for whole-response ==/!= comparisons. When rules are present, they replace the legacy single JSON query check.
              <br /><br />
              <a href="https://github.com/tidwall/gjson/blob/master/SYNTAX.md" target="_blank" rel="noopener noreferrer" className="underline">GJSON syntax documentation</a>
            </div>

            <FormField
              control={conditionForm.control}
              name="condition_operator"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Combine conditions</FormLabel>
                  <FormControl>
                    <Select value={field.value} onValueChange={field.onChange}>
                      <SelectTrigger><SelectValue /></SelectTrigger>
                      <SelectContent>
                        <SelectItem value="and">All conditions must pass (AND)</SelectItem>
                        <SelectItem value="or">At least one condition must pass (OR)</SelectItem>
                      </SelectContent>
                    </Select>
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            {fields.map((condition, index) => {
              const conditionType = conditionForm.watch(`conditions.${index}.type` as const) || "status";
              const conditionOperator = conditionForm.watch(`conditions.${index}.operator` as const);
              const isJSON = conditionType === "json";
              const needsValue = conditionOperator !== "exists" && conditionOperator !== "not_exists";
              return (
                <div key={condition.id} className="space-y-4 rounded-lg border p-4">
                  <div className="flex items-start gap-3">
                    <FormField
                      control={conditionForm.control}
                      name={`conditions.${index}.type` as const}
                      render={({ field }) => (
                        <FormItem className="flex-1">
                          <FormLabel>Check</FormLabel>
                          <Select
                            value={field.value}
                            onValueChange={value => {
                              field.onChange(value);
                              conditionForm.setValue(`conditions.${index}.json_query`, "", { shouldValidate: true });
                              if (value !== "json" && (conditionOperator === "exists" || conditionOperator === "not_exists")) {
                                conditionForm.setValue(`conditions.${index}.operator`, "==", { shouldValidate: true });
                              }
                            }}
                          >
                            <FormControl>
                              <SelectTrigger><SelectValue /></SelectTrigger>
                            </FormControl>
                            <SelectContent>
                              <SelectItem value="status">HTTP status code</SelectItem>
                              <SelectItem value="response_time">Response time (ms)</SelectItem>
                              <SelectItem value="json">JSON response</SelectItem>
                            </SelectContent>
                          </Select>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                    {fields.length > 1 && (
                      <Button type="button" variant="ghost" size="icon" className="mt-7" onClick={() => remove(index)} aria-label="Remove condition">
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    )}
                  </div>

                  {isJSON && (
                    <FormField
                      control={conditionForm.control}
                      name={`conditions.${index}.json_query` as const}
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>GJSON path (blank compares the complete JSON value)</FormLabel>
                          <FormControl><Input placeholder="e.g. data.status or items.0.id" {...field} /></FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  )}

                  <div className="grid grid-cols-2 gap-4">
                    <FormField
                      control={conditionForm.control}
                      name={`conditions.${index}.operator` as const}
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>Operator</FormLabel>
                          <Select value={field.value} onValueChange={field.onChange}>
                            <FormControl><SelectTrigger><SelectValue /></SelectTrigger></FormControl>
                            <SelectContent>
                              {(isJSON ? ["==", "!=", ">", "<", ">=", "<=", "exists", "not_exists"] : ["==", "!=", ">", "<", ">=", "<="]).map(operator => (
                                <SelectItem key={operator} value={operator}>{operator}</SelectItem>
                              ))}
                            </SelectContent>
                          </Select>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                    {needsValue && (
                      <FormField
                        control={conditionForm.control}
                        name={`conditions.${index}.expected_value` as const}
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>Expected value</FormLabel>
                            <FormControl>
                              <Input
                                placeholder={conditionType === "status" ? "200" : conditionType === "response_time" ? "500" : "value"}
                                {...field}
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                    )}
                  </div>
                </div>
              );
            })}

            <Button
              type="button"
              variant="outline"
              onClick={() => append({ type: "status", operator: "==", expected_value: "200" })}
            >
              <Plus className="mr-2 h-4 w-4" /> Add condition
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="space-y-4">
            <Notifications onNewNotifier={() => setNotifierSheetOpen(true)} />
          </CardContent>
        </Card>

        <Card>
          <CardContent className="space-y-4">
            <Tags />
          </CardContent>
        </Card>

        <Card>
          <CardContent className="space-y-4">
            <Proxies onNewProxy={() => setProxySheetOpen(true)} />
          </CardContent>
        </Card>

        <Card>
          <CardContent className="space-y-4">
            <Intervals />
          </CardContent>
        </Card>

        <Card>
          <CardContent className="space-y-4">
            <Advanced />
            <Separator className="my-8" />
            <Authentication />
            <Separator className="my-8" />
            <HttpOptions />
          </CardContent>
        </Card>

        <Button type="submit">
          {isPending && <Loader2 className="animate-spin" />}
          {mode === "create" ? t("monitors.form.buttons.create") : t("monitors.form.buttons.update")}
        </Button>
      </form>
    </Form>
  );
};

export default HttpJsonQuery;
