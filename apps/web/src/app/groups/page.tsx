import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import Layout from "@/layout";
import { client } from "@/api/client.gen";
import { useAuthStore } from "@/store/auth";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import type { AuthModel } from "@/api/types.gen";

type Group = {
  id: string;
  name: string;
  role: "owner" | "admin" | "member" | "viewer";
};
type Member = {
  user: Pick<AuthModel, "id" | "email" | "active">;
  role: "owner" | "admin" | "member" | "viewer";
};
type Response<T> = { data: T };

const GroupsPage = () => {
  const queryClient = useQueryClient();
  const user = useAuthStore((state) => state.user);
  const activeGroupID = useAuthStore((state) => state.activeGroupID);
  const setActiveGroupID = useAuthStore((state) => state.setActiveGroupID);
  const setUser = useAuthStore((state) => state.setUser);
  const [groupName, setGroupName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [role, setRole] = useState<"admin" | "member" | "viewer">("member");

  const groupsQuery = useQuery({
    queryKey: ["groups"],
    queryFn: async () => {
      const response = await client.instance.get<Response<Group[]>>("/groups");
      return response.data.data;
    },
  });
  const groups = groupsQuery.data ?? [];
  const selectedGroup =
    groups.find((group) => group.id === activeGroupID) ??
    groups.find((group) => group.id === user?.groupId) ??
    groups[0];
  const canManage = selectedGroup?.role === "owner" || selectedGroup?.role === "admin";

  useEffect(() => {
    if (!selectedGroup || activeGroupID === selectedGroup.id) return;
    setActiveGroupID(selectedGroup.id);
    setUser({ ...user, id: user?.id ?? "", email: user?.email ?? "", groupId: selectedGroup.id, role: selectedGroup.role });
  }, [activeGroupID, selectedGroup, setActiveGroupID, setUser, user]);

  const membersQuery = useQuery({
    queryKey: ["group-members", selectedGroup?.id],
    enabled: Boolean(selectedGroup && canManage),
    queryFn: async () => {
      const response = await client.instance.get<Response<Member[]>>(
        `/groups/${selectedGroup!.id}/members`,
      );
      return response.data.data;
    },
  });

  const createGroup = useMutation({
    mutationFn: async () => {
      const response = await client.instance.post<Response<Group>>("/groups", { name: groupName });
      return response.data.data;
    },
    onSuccess: async (group) => {
      setGroupName("");
      setActiveGroupID(group.id);
      setUser({ ...user, id: user?.id ?? "", email: user?.email ?? "", groupId: group.id, role: group.role });
      await queryClient.invalidateQueries({ queryKey: ["groups"] });
      toast.success("Group created");
    },
    onError: () => toast.error("Could not create group"),
  });

  const addMember = useMutation({
    mutationFn: async () => {
      const response = await client.instance.post<Response<Member>>(
        `/groups/${selectedGroup!.id}/members`,
        { email, password: password || undefined, role },
      );
      return response.data.data;
    },
    onSuccess: async () => {
      setEmail("");
      setPassword("");
      await queryClient.invalidateQueries({ queryKey: ["group-members", selectedGroup?.id] });
      toast.success("Member added");
    },
    onError: () => toast.error("Could not add member. Check the email, role, and password."),
  });

  const changeRole = useMutation({
    mutationFn: ({ id, role: nextRole }: { id: string; role: "admin" | "member" | "viewer" }) =>
      client.instance.patch(`/groups/${selectedGroup!.id}/members/${id}`, { role: nextRole }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["group-members", selectedGroup?.id] });
      toast.success("Access updated");
    },
    onError: () => toast.error("Could not update access"),
  });

  const removeMember = useMutation({
    mutationFn: (id: string) => client.instance.delete(`/groups/${selectedGroup!.id}/members/${id}`),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["group-members", selectedGroup?.id] });
      toast.success("Member removed");
    },
    onError: () => toast.error("Could not remove member. A group must retain an owner."),
  });

  return (
    <Layout pageName="Groups and access">
      <div className="space-y-6">
        <Card>
          <CardHeader>
            <CardTitle>Groups</CardTitle>
            <CardDescription>
              Groups are workspaces. Your access level is assigned separately in each group.
            </CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-4 sm:flex-row sm:items-end">
            <label className="flex flex-1 flex-col gap-2 text-sm">
              Active group
              <select
                className="h-9 rounded-md border bg-background px-3"
                value={selectedGroup?.id ?? ""}
                onChange={(event) => {
                  const group = groups.find((item) => item.id === event.target.value);
                  if (!group) return;
                  setActiveGroupID(group.id);
                  setUser({ ...user, id: user?.id ?? "", email: user?.email ?? "", groupId: group.id, role: group.role });
                }}
                disabled={groups.length === 0}
              >
                {groups.map((group) => (
                  <option key={group.id} value={group.id}>{group.name} — {group.role}</option>
                ))}
              </select>
            </label>
            <form
              className="flex flex-1 gap-2"
              onSubmit={(event) => {
                event.preventDefault();
                if (groupName.trim()) createGroup.mutate();
              }}
            >
              <Input value={groupName} onChange={(event) => setGroupName(event.target.value)} placeholder="New group name" maxLength={255} required />
              <Button type="submit" disabled={createGroup.isPending}>Create group</Button>
            </form>
          </CardContent>
        </Card>

        {selectedGroup && (
          <Card>
            <CardHeader>
              <CardTitle>Members of {selectedGroup.name}</CardTitle>
              <CardDescription>
                Owner: full group control. Admin: manage members. Member: manage monitors. Viewer: read-only.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-5">
              {!canManage ? (
                <p className="text-sm text-muted-foreground">Only group owners and admins can view or manage the member list.</p>
              ) : (
                <>
                  <form
                    className="grid gap-3 md:grid-cols-4"
                    onSubmit={(event) => {
                      event.preventDefault();
                      addMember.mutate();
                    }}
                  >
                    <Input type="email" value={email} onChange={(event) => setEmail(event.target.value)} placeholder="Member email" required />
                    <Input type="password" value={password} onChange={(event) => setPassword(event.target.value)} placeholder="Initial password (new account)" />
                    <select className="h-9 rounded-md border bg-background px-3" value={role} onChange={(event) => setRole(event.target.value as typeof role)}>
                      <option value="member">Member</option>
                      <option value="viewer">Viewer</option>
                      {selectedGroup.role === "owner" && <option value="admin">Admin</option>}
                    </select>
                    <Button type="submit" disabled={addMember.isPending}>Add member</Button>
                  </form>
                  <p className="text-xs text-muted-foreground">
                    Existing accounts can be added without a password. For a new account, set a strong initial password and share it securely.
                  </p>
                  {membersQuery.isLoading ? (
                    <p className="text-sm text-muted-foreground">Loading members…</p>
                  ) : (
                    <div className="divide-y rounded-md border">
                      {(membersQuery.data ?? []).map((member) => (
                        <div key={member.user.id} className="flex flex-wrap items-center justify-between gap-3 p-3">
                          <div>
                            <p className="font-medium">{member.user.email}</p>
                            <p className="text-xs text-muted-foreground">{member.user.active ? "Active account" : "Disabled account"}</p>
                          </div>
                          <div className="flex items-center gap-2">
                            {member.role === "owner" ? (
                              <span className="text-sm font-medium">Owner</span>
                            ) : (
                              <>
                                <select
                                  aria-label={`Role for ${member.user.email}`}
                                  className="h-9 rounded-md border bg-background px-3"
                                  value={member.role}
                                  onChange={(event) => changeRole.mutate({ id: member.user.id!, role: event.target.value as "admin" | "member" | "viewer" })}
                                >
                                  {selectedGroup.role === "owner" && <option value="admin">Admin</option>}
                                  <option value="member">Member</option>
                                  <option value="viewer">Viewer</option>
                                </select>
                                <Button variant="destructive" size="sm" onClick={() => removeMember.mutate(member.user.id!)}>Remove</Button>
                              </>
                            )}
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </>
              )}
            </CardContent>
          </Card>
        )}
      </div>
    </Layout>
  );
};

export default GroupsPage;
