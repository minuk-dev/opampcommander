export interface AuthnTokenResponse {
  token: string;
  refreshToken?: string;
  expiresAt?: string;
}

export interface AuthInfo {
  authenticated: boolean;
  email?: string | null;
}

export interface OAuth2AuthCodeURLResponse {
  url: string;
}
