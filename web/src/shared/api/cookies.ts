// httpOnly session cookie names. Shared by the /api/session route handler
// (which writes them) and the server-side API client (which reads the token).
// Kept here so the server client doesn't have to reach back into the app/ route
// layer.
export const TOKEN_COOKIE = 'opamp_token';
export const REFRESH_COOKIE = 'opamp_refresh_token';
export const EXPIRES_COOKIE = 'opamp_expires_at';
