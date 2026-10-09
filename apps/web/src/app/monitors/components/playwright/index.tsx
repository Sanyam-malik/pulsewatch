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
import { deserialize, playwrightDefaultValues, serialize, type PlaywrightForm } from "./schema";

const PlaywrightFormComponent = () => {
  const {
    form, setNotifierSheetOpen, isPending, mode, createMonitorMutation,
    editMonitorMutation, monitorId, monitor,
  } = useMonitorFormContext();

  useEffect(() => {
    if (mode === "create" && form.getValues("browser_ws_endpoint") === undefined) {
      const name = form.getValues("name");
      form.reset({ ...playwrightDefaultValues, name: name || playwrightDefaultValues.name });
    } else if (monitor?.data) {
      form.reset(deserialize(monitor.data));
    }
  }, [form, mode, monitor]);

  const onSubmit = (data: PlaywrightForm) => {
    const body = { ...serialize(data), active: mode === "create" ? true : monitor?.data?.active };
    if (mode === "create") {
      createMonitorMutation.mutate({ body });
    } else {
      editMonitorMutation.mutate({ path: { id: monitorId! }, body });
    }
  };

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(data => onSubmit(data as PlaywrightForm))} className="space-y-6 max-w-[600px]">
        <Card><CardContent className="space-y-4"><General /></CardContent></Card>
        <Card>
          <CardContent className="space-y-4">
            <h4 className="text-lg font-semibold">Browser check</h4>
            <p className="text-sm text-muted-foreground">Connects to a remote Chromium DevTools Protocol WebSocket endpoint. The worker does not bundle a browser.</p>
            <FormField control={form.control} name="browser_ws_endpoint" render={({ field }) => (
              <FormItem><FormLabel>Chromium DevTools WebSocket URL</FormLabel><FormControl><Input placeholder="ws://browser:9222/devtools/browser/…" {...field} /></FormControl><FormMessage /></FormItem>
            )} />
            <FormField control={form.control} name="url" render={({ field }) => (
              <FormItem><FormLabel>Page URL</FormLabel><FormControl><Input placeholder="https://" {...field} /></FormControl><FormMessage /></FormItem>
            )} />
            <FormField control={form.control} name="selector" render={({ field }) => (
              <FormItem><FormLabel>CSS selector to require (optional)</FormLabel><FormControl><Input placeholder="main h1" {...field} /></FormControl><FormMessage /></FormItem>
            )} />
            <FormField control={form.control} name="keyword" render={({ field }) => (
              <FormItem><FormLabel>Text to require in page (optional)</FormLabel><FormControl><Input placeholder="Welcome" {...field} /></FormControl><FormMessage /></FormItem>
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

export default PlaywrightFormComponent;
