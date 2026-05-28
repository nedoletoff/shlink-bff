import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from 'react';
import { api } from '../api/client';
import type { MeResponse } from '../types/api';

interface AuthState {
  user:    MeResponse | null;
  loading: boolean;
  error:   string | null;
  refetch: () => void;
}

const AuthContext = createContext<AuthState>({
  user: null, loading: true, error: null, refetch: () => {},
});

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user,    setUser]    = useState<MeResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error,   setError]   = useState<string | null>(null);

  const fetchMe = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const me = await api.get<MeResponse>('/api/me');
      setUser(me);
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Auth error';
      setError(msg);
      setUser(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchMe(); }, [fetchMe]);

  const value = useMemo(
    () => ({ user, loading, error, refetch: fetchMe }),
    [user, loading, error, fetchMe],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthState {
  return useContext(AuthContext);
}

export function useIsAdmin(): boolean {
  const { user } = useAuth();
  return user?.role === 'admin';
}
