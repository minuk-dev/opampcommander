// Class-name merge used by every component in shared/ui: clsx resolves
// conditionals, tailwind-merge drops earlier classes a later one overrides so
// a caller's `className` always wins over a component's defaults.

import { type ClassValue, clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs));
}
