import '@testing-library/jest-dom/vitest';
import { afterEach } from 'vitest';
import { cleanup } from '@testing-library/react';

// React Testing Library auto-cleans between tests when using its `cleanup`
// hook explicitly — without this each test would leak DOM into the next.
afterEach(() => {
  cleanup();
});
