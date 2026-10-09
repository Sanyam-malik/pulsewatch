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
import { deserialize, gameDigDefaultValues, serialize, type GameDigForm } from "./schema";

const GameDigFormComponent = () => {
  const {
    form, setNotifierSheetOpen, isPending, mode, createMonitorMutation,
    editMonitorMutation, monitorId, monitor,
  } = useMonitorFormContext();

  useEffect(() => {
    if (mode === "create" && form.getValues("game") === undefined) {
      const name = form.getValues("name");
      form.reset({ ...gameDigDefaultValues, name: name || gameDigDefaultValues.name });
    } else if (monitor?.data) {
      form.reset(deserialize(monitor.data));
    }
  }, [form, mode, monitor]);

  const onSubmit = (data: GameDigForm) => {
    const body = { ...serialize(data), active: mode === "create" ? true : monitor?.data?.active };
    if (mode === "create") {
      createMonitorMutation.mutate({ body });
    } else {
      editMonitorMutation.mutate({ path: { id: monitorId! }, body });
    }
  };

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(data => onSubmit(data as GameDigForm))} className="space-y-6 max-w-[600px]">
        <Card><CardContent className="space-y-4"><General /></CardContent></Card>
        <Card>
          <CardContent className="space-y-4">
            <h4 className="text-lg font-semibold">GameDig server query</h4>
            <p className="text-sm text-muted-foreground">Queries Source A2S-compatible game servers (including many Steam games).</p>
            <FormField control={form.control} name="game" render={({ field }) => (
              <FormItem><FormLabel>Game ID</FormLabel><FormControl><Input placeholder="source, cs2, tf2, rust…" {...field} /></FormControl><FormMessage /></FormItem>
            )} />
            <FormField control={form.control} name="host" render={({ field }) => (
              <FormItem><FormLabel>Host</FormLabel><FormControl><Input placeholder="game.example.com" {...field} /></FormControl><FormMessage /></FormItem>
            )} />
            <FormField control={form.control} name="port" render={({ field }) => (
              <FormItem><FormLabel>Query port</FormLabel><FormControl><Input type="number" {...field} onChange={event => field.onChange(event.target.value)} /></FormControl><FormMessage /></FormItem>
            )} />
            <FormField control={form.control} name="expected_name" render={({ field }) => (
              <FormItem><FormLabel>Expected server name (optional)</FormLabel><FormControl><Input {...field} /></FormControl><FormMessage /></FormItem>
            )} />
            <FormField control={form.control} name="expected_map" render={({ field }) => (
              <FormItem><FormLabel>Expected map (optional)</FormLabel><FormControl><Input {...field} /></FormControl><FormMessage /></FormItem>
            )} />
            <div className="grid grid-cols-2 gap-4">
              <FormField control={form.control} name="min_players" render={({ field }) => (
                <FormItem><FormLabel>Minimum players</FormLabel><FormControl><Input type="number" {...field} onChange={event => field.onChange(event.target.value)} /></FormControl><FormMessage /></FormItem>
              )} />
              <FormField control={form.control} name="max_players" render={({ field }) => (
                <FormItem><FormLabel>Maximum players (0 = no limit)</FormLabel><FormControl><Input type="number" {...field} onChange={event => field.onChange(event.target.value)} /></FormControl><FormMessage /></FormItem>
              )} />
            </div>
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

export default GameDigFormComponent;
