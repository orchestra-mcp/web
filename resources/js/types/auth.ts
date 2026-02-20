export type Subscription = {
    id: number;
    user_id: number;
    plan: 'free' | 'sponsor' | 'team_sponsor';
    status: 'active' | 'expired' | 'cancelled' | 'past_due';
    start_date: string | null;
    end_date: string | null;
    amount_cents: number;
    created_at: string;
    updated_at: string;
};

export type Role = {
    id: number;
    name: string;
    permissions_count?: number;
    users_count?: number;
    permissions?: Permission[];
};

export type Permission = {
    id: number;
    name: string;
};

export type UserMeta = {
    id: number;
    key: string;
    value: string | null;
};

export type Notification = {
    id: string;
    type: string;
    data: Record<string, string>;
    read_at: string | null;
    created_at: string;
};

export type User = {
    id: number;
    name: string;
    email: string;
    status: 'active' | 'blocked';
    password_set: boolean;
    avatar_url?: string;
    email_verified_at: string | null;
    two_factor_enabled?: boolean;
    roles?: Role[];
    subscription?: Subscription | null;
    metas?: UserMeta[];
    created_at: string;
    updated_at: string;
    [key: string]: unknown;
};

export type Auth = {
    user: User;
    notifications?: Notification[];
    isAdmin?: boolean;
    hasActiveSubscription?: boolean;
};

export type PageData = {
    id: number;
    slug: string;
    title: string;
    content: string;
    meta: Record<string, string> | null;
    is_published: boolean;
    created_at: string;
    updated_at: string;
};

export type TwoFactorSetupData = {
    svg: string;
    url: string;
};

export type TwoFactorSecretKey = {
    secretKey: string;
};
