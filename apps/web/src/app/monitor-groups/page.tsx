import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import Layout from "@/layout";
import { client } from "@/api/client.gen";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";

type Monitor = { id: string; name: string; type: string };
type MonitorGroup = {
  id: string;
  name: string;
  description: string;
  monitor_ids: string[];
};
type Response<T> = { data: T };

const MonitorGroupsPage = () => {
  const queryClient = useQueryClient();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [selectedGroupID, setSelectedGroupID] = useState("");
  const [selectedMonitorIDs, setSelectedMonitorIDs] = useState<string[]>([]);

  const groupsQuery = useQuery({
    queryKey: ["monitor-groups"],
    queryFn: async () => {
      const response = await client.instance.get<Response<MonitorGroup[]>>("/monitor-groups");
      return response.data.data;
    },
  });
  const monitorsQuery = useQuery({
    queryKey: ["monitors", "monitor-groups"],
    queryFn: async () => {
      const response = await client.instance.get<Response<Monitor[]>>("/monitors", {
        params: { page: 0, limit: 100 },
      });
      return response.data.data;
    },
  });

  const groups = groupsQuery.data ?? [];
  const selectedGroup =
    groups.find((group) => group.id === selectedGroupID) ?? groups[0];

  useEffect(() => {
    if (selectedGroup) {
      setSelectedMonitorIDs(selectedGroup.monitor_ids);
    }
  }, [selectedGroup]);

  const createGroup = useMutation({
    mutationFn: async () => {
      const response = await client.instance.post<Response<MonitorGroup>>("/monitor-groups", {
        name,
        description,
        monitor_ids: [],
      });
      return response.data.data;
    },
    onSuccess: async () => {
      setName("");
      setDescription("");
      await queryClient.invalidateQueries({ queryKey: ["monitor-groups"] });
      toast.success("Monitor group created");
    },
    onError: () => toast.error("Could not create monitor group"),
  });

  const saveMembership = useMutation({
    mutationFn: async () => {
      if (!selectedGroup) return;
      await client.instance.put(`/monitor-groups/${selectedGroup.id}/monitors`, {
        monitor_ids: selectedMonitorIDs,
      });
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["monitor-groups"] });
      toast.success("Monitor membership saved");
    },
    onError: () => toast.error("Could not save monitor membership"),
  });

  const deleteGroup = useMutation({
    mutationFn: async (id: string) => client.instance.delete(`/monitor-groups/${id}`),
    onSuccess: async () => {
      setSelectedGroupID("");
      await queryClient.invalidateQueries({ queryKey: ["monitor-groups"] });
      toast.success("Monitor group deleted");
    },
    onError: () => toast.error("Could not delete monitor group"),
  });

  const toggleMonitor = (id: string) => {
    setSelectedMonitorIDs((current) =>
      current.includes(id) ? current.filter((item) => item !== id) : [...current, id],
    );
  };

  return (
    <Layout pageName="Monitor groups">
      <div className="space-y-6">
        <Card>
          <CardHeader>
            <CardTitle>Monitor groups</CardTitle>
            <CardDescription>
              Organize monitors into reusable sets. These are separate from workspace access groups.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form
              className="grid gap-3 md:grid-cols-[1fr_2fr_auto]"
              onSubmit={(event) => {
                event.preventDefault();
                if (name.trim()) createGroup.mutate();
              }}
            >
              <Input
                value={name}
                onChange={(event) => setName(event.target.value)}
                placeholder="Group name"
                maxLength={255}
                required
              />
              <Textarea
                value={description}
                onChange={(event) => setDescription(event.target.value)}
                placeholder="Description (optional)"
                maxLength={2000}
                rows={1}
              />
              <Button type="submit" disabled={createGroup.isPending}>Create group</Button>
            </form>
          </CardContent>
        </Card>

        {groupsQuery.isLoading ? (
          <p className="text-sm text-muted-foreground">Loading monitor groups…</p>
        ) : groups.length === 0 ? (
          <Card>
            <CardContent className="p-6 text-sm text-muted-foreground">
              No monitor groups yet. Create one to organize related monitors.
            </CardContent>
          </Card>
        ) : (
          <div className="grid gap-6 lg:grid-cols-[minmax(220px,1fr)_2fr]">
            <Card>
              <CardHeader><CardTitle>Groups</CardTitle></CardHeader>
              <CardContent className="space-y-2">
                {groups.map((group) => (
                  <Button
                    key={group.id}
                    type="button"
                    variant={group.id === selectedGroup?.id ? "default" : "outline"}
                    className="w-full justify-start"
                    onClick={() => {
                      setSelectedGroupID(group.id);
                      setSelectedMonitorIDs(group.monitor_ids);
                    }}
                  >
                    {group.name} ({group.monitor_ids.length})
                  </Button>
                ))}
              </CardContent>
            </Card>

            {selectedGroup && (
              <Card>
                <CardHeader>
                  <CardTitle>{selectedGroup.name}</CardTitle>
                  {selectedGroup.description && (
                    <CardDescription>{selectedGroup.description}</CardDescription>
                  )}
                </CardHeader>
                <CardContent className="space-y-4">
                  {monitorsQuery.isLoading ? (
                    <p className="text-sm text-muted-foreground">Loading monitors…</p>
                  ) : monitorsQuery.isError ? (
                    <p className="text-sm text-destructive">Could not load monitors.</p>
                  ) : (monitorsQuery.data ?? []).length === 0 ? (
                    <p className="text-sm text-muted-foreground">Create monitors before assigning them to a group.</p>
                  ) : (
                    <div className="divide-y rounded-md border">
                      {(monitorsQuery.data ?? []).map((monitor) => (
                        <label key={monitor.id} className="flex cursor-pointer items-center gap-3 p-3">
                          <input
                            type="checkbox"
                            checked={selectedMonitorIDs.includes(monitor.id)}
                            onChange={() => toggleMonitor(monitor.id)}
                          />
                          <span className="flex-1 font-medium">{monitor.name}</span>
                          <span className="text-xs text-muted-foreground">{monitor.type}</span>
                        </label>
                      ))}
                    </div>
                  )}
                  <div className="flex flex-wrap justify-between gap-2">
                    <Button
                      variant="destructive"
                      type="button"
                      disabled={deleteGroup.isPending}
                      onClick={() => deleteGroup.mutate(selectedGroup.id)}
                    >
                      Delete group
                    </Button>
                    <Button
                      type="button"
                      disabled={saveMembership.isPending || monitorsQuery.isLoading}
                      onClick={() => saveMembership.mutate()}
                    >
                      Save monitors
                    </Button>
                  </div>
                </CardContent>
              </Card>
            )}
          </div>
        )}
      </div>
    </Layout>
  );
};

export default MonitorGroupsPage;
