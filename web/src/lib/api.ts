const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:9080';

class ApiClient {
  private baseUrl: string;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

  private getHeaders(): HeadersInit {
    const headers: HeadersInit = { 'Content-Type': 'application/json' };
    if (typeof window !== 'undefined') {
      const token = localStorage.getItem('access_token');
      if (token) headers['Authorization'] = `Bearer ${token}`;
    }
    return headers;
  }

  private getAuthHeader(): HeadersInit {
    const headers: HeadersInit = {};
    if (typeof window !== 'undefined') {
      const token = localStorage.getItem('access_token');
      if (token) headers['Authorization'] = `Bearer ${token}`;
    }
    return headers;
  }

  async get<T>(path: string): Promise<T> {
    const res = await fetch(`${this.baseUrl}${path}`, { headers: this.getHeaders() });
    if (!res.ok) throw new Error(`API error: ${res.status}`);
    return res.json();
  }

  async post<T>(path: string, body?: any): Promise<T> {
    const res = await fetch(`${this.baseUrl}${path}`, {
      method: 'POST',
      headers: this.getHeaders(),
      body: body ? JSON.stringify(body) : undefined,
    });
    if (!res.ok) throw new Error(`API error: ${res.status}`);
    return res.json();
  }

  async put<T>(path: string, body?: any): Promise<T> {
    const res = await fetch(`${this.baseUrl}${path}`, {
      method: 'PUT',
      headers: this.getHeaders(),
      body: body ? JSON.stringify(body) : undefined,
    });
    if (!res.ok) throw new Error(`API error: ${res.status}`);
    return res.json();
  }

  async patch<T>(path: string, body?: any): Promise<T> {
    const res = await fetch(`${this.baseUrl}${path}`, {
      method: 'PATCH',
      headers: this.getHeaders(),
      body: body ? JSON.stringify(body) : undefined,
    });
    if (!res.ok) throw new Error(`API error: ${res.status}`);
    return res.json();
  }

  async delete<T>(path: string): Promise<T> {
    const res = await fetch(`${this.baseUrl}${path}`, {
      method: 'DELETE',
      headers: this.getHeaders(),
    });
    if (!res.ok) throw new Error(`API error: ${res.status}`);
    return res.json();
  }

  async uploadFile(path: string, file: File, extraFields?: Record<string, string>): Promise<any> {
    const formData = new FormData();
    formData.append('file', file);
    if (extraFields) {
      Object.entries(extraFields).forEach(([k, v]) => formData.append(k, v));
    }
    const res = await fetch(`${this.baseUrl}${path}`, {
      method: 'POST',
      headers: this.getAuthHeader(),
      body: formData,
    });
    if (!res.ok) throw new Error(`Upload error: ${res.status}`);
    return res.json();
  }
}

export const api = new ApiClient(API_BASE);

// ── Field-shape adapters (v2 proto → v1 frontend expectations) ──────────────

function normalizeUser(u: any): any {
  if (!u) return u;
  const parts = (u.full_name || '').trim().split(/\s+/);
  return {
    ...u,
    first_name: parts[0] || '',
    last_name: parts.slice(1).join(' ') || '',
    role_name: u.role || 'student',
    language: u.locale || 'en',
    theme: 'light',
  };
}

function normalizeCourse(c: any): any {
  if (!c) return c;
  return {
    ...c,
    title_en: c.title_en || c.title || '',
    description_en: c.description_en || c.description || '',
    code: c.code || '',
    credits: c.credits || 0,
    is_published: c.is_published !== undefined ? c.is_published : true,
  };
}

// ── Auth ─────────────────────────────────────────────────────────────────────

export const authApi = {
  login: async (email: string, password: string) => {
    const tokens = await api.post<any>('/api/v1/auth/login', { email, password });
    // Store token early so the /me call below can authenticate
    if (typeof window !== 'undefined') {
      localStorage.setItem('access_token', tokens.access_token);
      if (tokens.refresh_token) localStorage.setItem('refresh_token', tokens.refresh_token);
    }
    const me = await api.get<any>('/api/v1/auth/me');
    const user = normalizeUser(me.user);
    return {
      user,
      tokens: { access_token: tokens.access_token, refresh_token: tokens.refresh_token },
      permissions: [],
    };
  },

  register: (data: { email: string; password: string; full_name?: string; first_name?: string; last_name?: string }) => {
    const payload: any = { ...data };
    if (!payload.full_name && (payload.first_name || payload.last_name)) {
      payload.full_name = [payload.first_name, payload.last_name].filter(Boolean).join(' ');
    }
    return api.post('/api/v1/auth/register', payload);
  },

  me: async () => {
    const res = await api.get<any>('/api/v1/auth/me');
    return { user: normalizeUser(res.user) };
  },

  refresh: (refresh_token: string) => api.post('/api/v1/auth/refresh', { refresh_token }),

  logout: () => {
    const rt = typeof window !== 'undefined' ? localStorage.getItem('refresh_token') : '';
    return api.post('/api/v1/auth/logout', { refresh_token: rt || '' });
  },
};

// ── Courses ───────────────────────────────────────────────────────────────────

export const courseApi = {
  list: async (_userId?: string) => {
    const res = await api.get<any>('/api/v1/courses');
    const courses = (res.courses || []).map(normalizeCourse);
    return { courses };
  },

  get: async (id: string, _userId?: string) => {
    const res = await api.get<any>(`/api/v1/courses/${id}`);
    return normalizeCourse(res.course || res);
  },

  create: (data: any) => {
    const payload: any = {
      title: data.title_en || data.title || '',
      description: data.description_en || data.description || '',
    };
    return api.post('/api/v1/courses', payload);
  },

  update: (id: string, data: any) => {
    const payload: any = {
      title: data.title_en || data.title || '',
      description: data.description_en || data.description || '',
    };
    return api.patch(`/api/v1/courses/${id}`, payload);
  },

  delete: (id: string) => api.delete(`/api/v1/courses/${id}`),

  sections: async (id: string) => {
    const res = await api.get<any>(`/api/v1/courses/${id}/sections`);
    return { sections: res.sections || [] };
  },

  createSection: (courseId: string, data: any) =>
    api.post(`/api/v1/courses/${courseId}/sections`, {
      title: data.title_en || data.title || '',
      position: data.position || 0,
    }),

  updateSection: (_id: string, _data: any) => Promise.resolve({}),

  deleteSection: (_id: string) => Promise.resolve({}),

  materials: async (sectionId: string) => {
    const res = await api.get<any>(`/api/v1/sections/${sectionId}/materials`);
    return { materials: res.materials || [] };
  },

  createMaterial: (sectionId: string, data: any) =>
    api.post(`/api/v1/sections/${sectionId}/materials`, {
      title: data.title || '',
      url: data.url || '',
      kind: data.kind || 'link',
    }),

  deleteMaterial: (_id: string) => Promise.resolve({}),

  enrollments: async (id: string) => {
    const res = await api.get<any>(`/api/v1/courses/${id}/enrollments`);
    return { enrollments: res.enrollments || [] };
  },

  enroll: (courseId: string, _userId?: string, _role?: string) =>
    api.post(`/api/v1/courses/${courseId}/enroll`, {}),

  unenroll: (courseId: string, _userId?: string) =>
    api.delete(`/api/v1/courses/${courseId}/enroll`),
};

// ── Quizzes (v2 assessment service) ──────────────────────────────────────────
// These endpoints are new in v2 and don't have a v1 counterpart.

export const quizApi = {
  list: (courseId: string) =>
    api.get<any>(`/api/v1/assessments/quizzes?course_id=${courseId}`),

  get: (id: string) => api.get<any>(`/api/v1/assessments/quizzes/${id}`),

  create: (data: { course_id: string; title: string; time_limit_sec?: number; shuffle?: boolean }) =>
    api.post<any>('/api/v1/assessments/quizzes', data),

  update: (id: string, data: any) => api.patch<any>(`/api/v1/assessments/quizzes/${id}`, data),

  delete: (id: string) => api.delete<any>(`/api/v1/assessments/quizzes/${id}`),

  startAttempt: (quizId: string) =>
    api.post<any>('/api/v1/assessments/attempts', { quiz_id: quizId }),

  submitAttempt: (attemptId: string, answers: { question_id: string; chosen_key: string }[]) =>
    api.post<any>(`/api/v1/assessments/attempts/${attemptId}/submit`, { answers }),

  getAttempt: (id: string) => api.get<any>(`/api/v1/assessments/attempts/${id}`),

  listAttempts: (quizId?: string, studentId?: string) => {
    const p = new URLSearchParams();
    if (quizId) p.set('quiz_id', quizId);
    if (studentId) p.set('student_id', studentId);
    return api.get<any>(`/api/v1/assessments/attempts?${p.toString()}`);
  },

  gradebook: (courseId: string) =>
    api.get<any>(`/api/v1/assessments/gradebook?course_id=${courseId}`),

  exportGrades: (courseId: string) =>
    `${API_BASE}/api/v1/assessments/gradebook/export?course_id=${courseId}`,
};

// ── Stubs for v1-only services (not in v2 gateway, fail gracefully) ──────────

export const gradeApi = {
  gradebook: (courseId: string) => quizApi.gradebook(courseId),
  progress: (_courseId: string) => Promise.reject(new Error('not implemented in v2')),
  advancedProgress: (_courseId: string) => Promise.reject(new Error('not implemented in v2')),
  create: (_data: any) => Promise.reject(new Error('not implemented in v2')),
  update: (_id: string, _data: any) => Promise.reject(new Error('not implemented in v2')),
};

export const formulaApi = {
  get: (_courseId: string) => Promise.reject(new Error('not implemented in v2')),
  create: (_data: any) => Promise.reject(new Error('not implemented in v2')),
  update: (_id: string, _data: any) => Promise.reject(new Error('not implemented in v2')),
};

export const attendanceApi = {
  course: (_courseId: string, _date?: string) => Promise.reject(new Error('not implemented in v2')),
  mark: (_data: any) => Promise.reject(new Error('not implemented in v2')),
  stats: (_courseId: string) => Promise.reject(new Error('not implemented in v2')),
};

export const mediaApi = {
  upload: (_file: File, _userId?: string) => Promise.reject(new Error('not implemented in v2')),
  getFileUrl: (id: string) => `${API_BASE}/api/media/files/${id}/download`,
};

export const analyticsApi = {
  overview: () => Promise.reject(new Error('not implemented in v2')),
};

export const userApi = {
  list: (_filters?: any) => Promise.resolve({ users: [] }),
  get: (_id: string) => Promise.reject(new Error('not implemented in v2')),
  roles: () => Promise.resolve({ roles: [] }),
  updateRole: (_userId: string, _roleName: string) => Promise.reject(new Error('not implemented in v2')),
  groups: () => Promise.resolve({ groups: [] }),
  createGroup: (_data: any) => Promise.reject(new Error('not implemented in v2')),
  deleteGroup: (_id: string) => Promise.reject(new Error('not implemented in v2')),
  setUserGroup: (_userId: string, _groupId: string | null) => Promise.reject(new Error('not implemented in v2')),
  search: (_query: string) => Promise.resolve({ users: [] }),
  updateProfile: (_userId: string, _data: any) => Promise.reject(new Error('not implemented in v2')),
  uploadAvatar: (_userId: string, _file: File) => Promise.reject(new Error('not implemented in v2')),
  listPermissions: () => Promise.resolve({ permissions: [] }),
  getRolePermissions: (_roleId: string) => Promise.resolve({ permissions: [] }),
  updateRolePermissions: (_roleId: string, _permissionIds: string[]) => Promise.reject(new Error('not implemented in v2')),
};

export const notificationApi = {
  list: (_userId: string) => Promise.resolve({ notifications: [], unread_count: 0 }),
  markRead: (_id: string) => Promise.reject(new Error('not implemented in v2')),
  markAllRead: (_userId: string) => Promise.reject(new Error('not implemented in v2')),
  delete: (_id: string) => Promise.reject(new Error('not implemented in v2')),
  create: (_data: any) => Promise.reject(new Error('not implemented in v2')),
  createBulk: (_data: any) => Promise.reject(new Error('not implemented in v2')),
};

export const newsApi = {
  list: () => Promise.resolve({ news: [], total: 0 }),
  create: (_data: any) => Promise.reject(new Error('not implemented in v2')),
  delete: (_id: string) => Promise.reject(new Error('not implemented in v2')),
};

export const scheduleApi = {
  list: (_courseId?: string) => Promise.resolve({ schedule: [] }),
  create: (_data: any) => Promise.reject(new Error('not implemented in v2')),
  update: (_id: string, _data: any) => Promise.reject(new Error('not implemented in v2')),
  delete: (_id: string) => Promise.reject(new Error('not implemented in v2')),
  user: (_userId: string) => Promise.resolve({ schedule: [] }),
};

export const assignmentApi = {
  list: (_courseId?: string) => Promise.resolve({ assignments: [] }),
  get: (_id: string) => Promise.reject(new Error('not implemented in v2')),
  create: (_data: any) => Promise.reject(new Error('not implemented in v2')),
  update: (_id: string, _data: any) => Promise.reject(new Error('not implemented in v2')),
  delete: (_id: string) => Promise.reject(new Error('not implemented in v2')),
  submit: (_id: string, _data: any) => Promise.reject(new Error('not implemented in v2')),
  submissions: (_id: string) => Promise.resolve({ submissions: [] }),
  deleteSubmission: (_id: string, _userId: string) => Promise.reject(new Error('not implemented in v2')),
  gradeSubmission: (_id: string, _data: any) => Promise.reject(new Error('not implemented in v2')),
};

export const sessionApi = {
  list: (_courseId?: string, _from?: string, _to?: string) => Promise.resolve({ sessions: [] }),
  create: (_data: any) => Promise.reject(new Error('not implemented in v2')),
  delete: (_id: string) => Promise.reject(new Error('not implemented in v2')),
  user: (_userId: string) => Promise.resolve({ sessions: [] }),
};
