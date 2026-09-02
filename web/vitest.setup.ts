import '@testing-library/jest-dom/vitest';
import { afterEach } from 'vitest';
import { cleanup } from '@testing-library/react';

// jsdom implements neither ResizeObserver nor the pointer-capture APIs, both of
// which Radix primitives call while opening a popup. Without these stubs every
// Select/DropdownMenu interaction throws before the menu ever renders.
globalThis.ResizeObserver ??= class {
  observe() {}
  unobserve() {}
  disconnect() {}
};
Element.prototype.hasPointerCapture ??= () => false;
Element.prototype.setPointerCapture ??= () => {};
Element.prototype.releasePointerCapture ??= () => {};
Element.prototype.scrollIntoView ??= () => {};

// React Testing Library auto-cleans between tests when using its `cleanup`
// hook explicitly — without this each test would leak DOM into the next.
afterEach(() => {
  cleanup();
});
