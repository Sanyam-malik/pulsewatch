import { useEffect } from "react";
import { Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import General from "../shared/general";
import Intervals from "../shared/intervals";
import Notifications from "../shared/notifications";
import Tags from "../shared/tags";
import { useMonitorFormContext } from "../../context/monitor-form-context";
import { deserialize, serialize, steamDefaultValues, type SteamForm } from "./schema";

const SteamFormComponent = () => {
  const {
    form, setNotifierSheetOpen, isPending, mode, createMonitorMutation,
    editMonitorMutation, monitorId, monitor,
  } = useMonitorFormContext();

  useEffect(() => {
    if (mode === "create" && form.getValues("expected_app_id") === undefined) {
      const name = form.getValues("name");
      form.reset({ ...steamDefaultValues, name: name || steamDefaultValues.name });
    } else if (monitor?.data) {
      form.reset(deserialize(monitor.data));
    }
  }, [form, mode, monitor]);

  const onSubmit = (data: SteamForm) => {
    const body = { ...serialize(data), active: mode === "create" ? true : monitor?.data?.active };
    if (mode === "create") {
      createMonitorMutation.mutate({ body });
    } else {
      editMonitorMutation.mutate({ path: { id: monitorId! }, body });
    }
  };

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(data => onSubmit(data as SteamForm))} className="space-y-6 max-w-[600px]">
        <Card><CardContent className="space-y-4"><General /></CardContent></Card>
        <Card>
          <CardContent className="space-y-4">
            <h4 className="text-lg font-semibold">Steam server query</h4>
            <p className="text-sm text-muted-foreground">Checks a Steam server using the Source A2S_INFO UDP query protocol.</p>
            <FormField control={form.control} name="host" render={({ field }) => (
              <FormItem><FormLabel>Host</FormLabel><FormControl><Input placeholder="game.example.com" {...field} /></FormControl><FormMessage /></FormItem>
            )} />
            <FormField control={form.control} name="port" render={({ field }) => (
              <FormItem><FormLabel>Query port</FormLabel><FormControl><Input type="number" {...field} onChange={event => field.onChange(event.target.value)} /></FormControl><FormMessage /></FormItem>
            )} />
            <FormField control={form.control} name="expected_app_id" render={({ field }) => (
              <FormItem><FormLabel>Expected Steam app ID (optional)</FormLabel><FormControl><Input type="number" placeholder="e.g. 730" {...field} onChange={event => field.onChange(event.target.value)} /></FormControl><FormMessage /></FormItem>
            )} />
            <FormField control={form.control} name="expected_name" render={({ field }) => (
              <FormItem><FormLabel>Expected server name (optional)</FormLabel><FormControl><Input {...field} /></FormControl><FormMessage /></FormItem>
            )} />
          </CardContent>
        </Card>
        <Card><CardContent className="space-y-4"><Notifications onNewNotifier={() => setNotifierSheetOpen(true)} /></CardContent></Card>
        <Card><CardContent className="space-y-4"><Tags /></CardContent></Card>
        <Card><CardContent className="space-y-4"><Intervals /></CardContent></Card>
        <Button type="submit">{isPending && <Loader2 className="animate-spin" />}{mode === "create" ? "Create monitor" : "Update monitor"}</Button>
      </form>
    </Form>
  );
};

export default SteamFormComponent;
