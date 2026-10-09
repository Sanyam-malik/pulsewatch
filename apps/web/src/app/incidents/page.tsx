import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import Layout from "@/layout";
import { client } from "@/api/client.gen";
import { useAuthStore } from "@/store/auth";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";

type StatusPage = { id: string; title: string; slug: string };
type IncidentUpdate = { id: string; status: string; message: string; created_at: string };
type Incident = {
  id: string;
  status_page_id: string;
  status_page_title: string;
  status_page_slug: string;
  title: string;
  status: "investigating" | "identified" | "monitoring" | "resolved";
  created_at: string;
  updated_at: string;
  resolved_at?: string;
  updates: IncidentUpdate[];
};
type Response<T> = { data: T };
const statuses: Incident["status"][] = ["investigating", "identified", "monitoring", "resolved"];

const IncidentsPage = () => {
  const queryClient = useQueryClient();
  const role = useAuthStore((state) => state.user?.role);
  const readOnly = role === "viewer";
  const [statusPageID, setStatusPageID] = useState("");
  const [title, setTitle] = useState("");
  const [message, setMessage] = useState("");
  const [updates, setUpdates] = useState<Record<string, { status: Incident["status"]; message: string }>>({});

  const pagesQuery = useQuery({
    queryKey: ["incident-status-pages"],
    queryFn: async () => {
      const response = await client.instance.get<Response<StatusPage[]>>("/status-pages", {
        params: { page: 0, limit: 100 },
      });
      return response.data.data;
    },
  });
  const incidentsQuery = useQuery({
    queryKey: ["incidents"],
    queryFn: async () => {
      const response = await client.instance.get<Response<Incident[]>>("/incidents");
      return response.data.data;
    },
  });

  const createIncident = useMutation({
    mutationFn: async () => client.instance.post("/incidents", {
      status_page_id: statusPageID,
      title,
      message,
    }),
    onSuccess: async () => {
      setTitle("");
      setMessage("");
      await queryClient.invalidateQueries({ queryKey: ["incidents"] });
      toast.success("Incident created");
    },
    onError: () => toast.error("Could not create incident"),
  });

  const addUpdate = useMutation({
    mutationFn: async ({ id, status, message: updateMessage }: { id: string; status: Incident["status"]; message: string }) =>
      client.instance.post(`/incidents/${id}/updates`, { status, message: updateMessage }),
    onSuccess: async (_, variables) => {
      setUpdates((current) => ({
        ...current,
        [variables.id]: { status: variables.status, message: "" },
      }));
      await queryClient.invalidateQueries({ queryKey: ["incidents"] });
      toast.success("Incident update posted");
    },
    onError: () => toast.error("Could not post incident update"),
  });

  return (
    <Layout pageName="Incidents">
      <div className="space-y-6">
        {!readOnly && (
          <Card>
            <CardHeader>
              <CardTitle>Report an incident</CardTitle>
              <CardDescription>
                New incidents start as investigating and appear on the selected published status page.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <form
                className="space-y-3"
                onSubmit={(event) => {
                  event.preventDefault();
                  if (statusPageID && title.trim() && message.trim()) createIncident.mutate();
                }}
              >
                <select
                  className="h-9 w-full rounded-md border bg-background px-3 text-sm"
                  value={statusPageID}
                  onChange={(event) => setStatusPageID(event.target.value)}
                  required
                >
                  <option value="">Select a status page</option>
                  {(pagesQuery.data ?? []).map((page) => (
                    <option key={page.id} value={page.id}>{page.title} ({page.slug})</option>
                  ))}
                </select>
                <Input
                  value={title}
                  onChange={(event) => setTitle(event.target.value)}
                  placeholder="Incident title"
                  maxLength={255}
                  required
                />
                <Textarea
                  value={message}
                  onChange={(event) => setMessage(event.target.value)}
                  placeholder="Describe what is happening"
                  maxLength={4000}
                  required
                />
                <Button type="submit" disabled={createIncident.isPending || pagesQuery.isLoading}>
                  Create incident
                </Button>
              </form>
            </CardContent>
          </Card>
        )}

        {incidentsQuery.isLoading ? (
          <p className="text-sm text-muted-foreground">Loading incidents…</p>
        ) : incidentsQuery.isError ? (
          <p className="text-sm text-destructive">Could not load incidents.</p>
        ) : (incidentsQuery.data ?? []).length === 0 ? (
          <Card><CardContent className="p-6 text-sm text-muted-foreground">No incidents have been reported.</CardContent></Card>
        ) : (
          <div className="space-y-4">
            {(incidentsQuery.data ?? []).map((incident) => {
              const draft = updates[incident.id] ?? { status: incident.status, message: "" };
              return (
                <Card key={incident.id}>
                  <CardHeader>
                    <div className="flex flex-wrap items-start justify-between gap-3">
                      <div>
                        <CardTitle>{incident.title}</CardTitle>
                        <CardDescription>
                          {incident.status_page_title} · {incident.status_page_slug} · {incident.status}
                        </CardDescription>
                      </div>
                      <span className="text-xs text-muted-foreground">
                        Updated {new Date(incident.updated_at).toLocaleString()}
                      </span>
                    </div>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <ol className="space-y-3 border-l pl-4">
                      {incident.updates.map((update) => (
                        <li key={update.id}>
                          <p className="font-medium capitalize">{update.status}</p>
                          <p className="text-sm">{update.message}</p>
                          <time className="text-xs text-muted-foreground">{new Date(update.created_at).toLocaleString()}</time>
                        </li>
                      ))}
                    </ol>
                    {!readOnly && incident.status !== "resolved" && (
                      <form
                        className="space-y-3 border-t pt-4"
                        onSubmit={(event) => {
                          event.preventDefault();
                          if (draft.message.trim()) {
                            addUpdate.mutate({ id: incident.id, ...draft });
                          }
                        }}
                      >
                        <select
                          className="h-9 rounded-md border bg-background px-3 text-sm"
                          value={draft.status}
                          onChange={(event) => setUpdates((current) => ({
                            ...current,
                            [incident.id]: { ...draft, status: event.target.value as Incident["status"] },
                          }))}
                        >
                          {statuses.map((status) => <option key={status} value={status}>{status}</option>)}
                        </select>
                        <Textarea
                          value={draft.message}
                          onChange={(event) => setUpdates((current) => ({
                            ...current,
                            [incident.id]: { ...draft, message: event.target.value },
                          }))}
                          placeholder="Post an update"
                          maxLength={4000}
                          required
                        />
                        <Button type="submit" disabled={addUpdate.isPending}>Post update</Button>
                      </form>
                    )}
                  </CardContent>
                </Card>
              );
            })}
          </div>
        )}
      </div>
    </Layout>
  );
};

export default IncidentsPage;
