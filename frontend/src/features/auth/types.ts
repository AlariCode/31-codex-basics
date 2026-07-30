export type AuthUser = {
  id: string;
  email: string;
  name: string;
  avatar_url: string | null;
};

export type AuthResponse = {
  access_token: string;
  token_type: "Bearer";
  expires_in: number;
  user: AuthUser;
};

export type LoginInput = {
  email: string;
  password: string;
};

export type RegisterInput = LoginInput & {
  name: string;
};
