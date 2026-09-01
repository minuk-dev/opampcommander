// Client-side checks for the agent-package editor.

// PACKAGE_TYPES mirrors the OpAMP package types the server maps onto the
// protobuf enum (an unrecognized value silently becomes TopLevel server-side,
// so the UI offers only the two that mean something).
export const PACKAGE_TYPES = ['TopLevel', 'AddOn'] as const;

// spec.contentHash / spec.signature / spec.hash are []byte server-side, so JSON
// carries them base64-encoded. Catch the common mistake of pasting a hex digest.
export function validateBase64(value: string, label: string): string | null {
  if (value === '') return null;
  if (!/^[A-Za-z0-9+/]+={0,2}$/.test(value) || value.length % 4 !== 0) {
    return `${label} must be base64 (a hex digest needs converting first).`;
  }
  return null;
}

export function validateDownloadUrl(value: string): string | null {
  if (value.trim() === '') return 'Download URL is required.';
  let url: URL;
  try {
    url = new URL(value);
  } catch {
    return 'Download URL must be an absolute URL.';
  }
  if (url.protocol !== 'http:' && url.protocol !== 'https:') {
    return 'Download URL must use http or https.';
  }
  return null;
}
