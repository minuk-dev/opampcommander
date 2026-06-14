'use client';

import { useSyncExternalStore } from 'react';

// A single shared "current time" that ticks on one interval for the whole page.
// Relative timestamps ("5 minutes ago") need a periodically-refreshed reference
// time, but a list can render hundreds of them — so instead of each component
// owning a timer, they all subscribe to this one store via useSyncExternalStore.
// The interval only runs while there is at least one subscriber.

// 30s is fine-grained enough for minute/hour/day relative phrasings without
// re-rendering every second.
const REFRESH_MS = 30_000;

let now = Date.now();
const listeners = new Set<() => void>();
let timer: ReturnType<typeof setInterval> | null = null;

function subscribe(onChange: () => void): () => void {
  listeners.add(onChange);
  if (!timer) {
    // The interval is the only thing that advances `now`, so while no one was
    // subscribed the clock sat frozen at the last tick. Refresh it as the timer
    // (re)starts — otherwise the first render after a remount would format
    // against a stale time and could even read a past event as a future one.
    now = Date.now();
    timer = setInterval(() => {
      now = Date.now();
      listeners.forEach((listener) => listener());
    }, REFRESH_MS);
  }
  return () => {
    listeners.delete(onChange);
    if (listeners.size === 0 && timer) {
      clearInterval(timer);
      timer = null;
    }
  };
}

function getSnapshot(): number {
  return now;
}

// Deterministic on the server so the markup matches the first client render;
// the real clock takes over after hydration. Relative output is only shown
// post-hydration anyway, so this value is never formatted.
function getServerSnapshot(): number {
  return 0;
}

// Current epoch-ms, refreshed every REFRESH_MS. Re-renders subscribers on tick.
export function useNow(): number {
  return useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
}
