// Client-safe data-access surface. NOTE: the server-only RSC client lives in
// ./server and is intentionally NOT re-exported here, so importing this barrel
// from a Client Component never pulls `server-only` into the browser bundle.
export * from './client';
export * from './swr';
export * from './session';
export * from './auth-storage';
export * from './cookies';
export * from './model/types';
