import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { AuthModel } from "@/api/types.gen";

interface AuthState {
  accessToken: string | null;
  refreshToken: string | null;
  user: AuthModel | null;
  activeGroupID: string | null;
  setTokens: (accessToken: string, refreshToken: string) => void;
  setUser: (user: AuthModel | null) => void;
  setActiveGroupID: (groupID: string | null) => void;
  clearTokens: () => void;
  clearUser: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      accessToken: null,
      refreshToken: null,
      user: null,
      activeGroupID: null,
      setTokens: (accessToken: string, refreshToken: string) =>
        set({ accessToken, refreshToken }),
      setUser: (user: AuthModel | null) => set({ user }),
      setActiveGroupID: (activeGroupID: string | null) => set({ activeGroupID }),
      clearTokens: () => set({ accessToken: null, refreshToken: null, user: null, activeGroupID: null }),
      clearUser: () => set({ user: null }),
    }),
    {
      name: 'auth-storage',
    }
  )
);
