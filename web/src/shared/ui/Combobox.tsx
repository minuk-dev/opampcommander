'use client';

import * as PopoverPrimitive from '@radix-ui/react-popover';
import { Command, CommandEmpty, CommandInput, CommandItem, CommandList } from 'cmdk';
import { Check, ChevronsUpDown } from 'lucide-react';
import { useRef, useState } from 'react';
import { cn } from '@shared/lib';
import Button from './Button';

export interface ComboboxOption {
  value: string;
  label: string;
  description?: string;
}

interface Props {
  value: string;
  onChange: (value: string) => void;
  options: ComboboxOption[];
  placeholder?: string;
  searchPlaceholder?: string;
  emptyMessage?: string;
  id?: string;
  className?: string;
  disabled?: boolean;
  onSearchChange?: (search: string) => void;
  onOpenChange?: (open: boolean) => void;
  loading?: boolean;
  error?: string;
  onRetry?: () => void;
  onNextPage?: () => void;
  'aria-describedby'?: string;
}

// Searchable single-select for long option lists (timezones, resource picks).
export default function Combobox({
  value,
  onChange,
  options,
  placeholder = 'Select…',
  searchPlaceholder = 'Search…',
  emptyMessage = 'No match',
  id,
  className,
  disabled,
  onSearchChange,
  onOpenChange,
  loading,
  error,
  onRetry,
  onNextPage,
  ...rest
}: Props) {
  const [open, setOpen] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const selected = options.find((o) => o.value === value);

  return (
    <PopoverPrimitive.Root
      open={open}
      onOpenChange={(next) => {
        setOpen(next);
        onOpenChange?.(next);
      }}
    >
      <PopoverPrimitive.Trigger
        id={id}
        role="combobox"
        type="button"
        aria-expanded={open}
        disabled={disabled}
        className={cn(
          'flex h-8 w-full items-center justify-between gap-2 rounded-md border border-input bg-card px-2.5 text-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60 disabled:cursor-not-allowed disabled:opacity-60',
          className,
        )}
        {...rest}
      >
        <span className={cn('truncate', !selected && 'text-muted-foreground')}>
          {selected?.label ?? (value || placeholder)}
        </span>
        <ChevronsUpDown className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
      </PopoverPrimitive.Trigger>
      <PopoverPrimitive.Portal>
        <PopoverPrimitive.Content
          align="start"
          sideOffset={4}
          className="z-50 w-(--radix-popover-trigger-width) overflow-hidden rounded-md border border-border bg-popover text-popover-foreground shadow-lg data-[state=open]:animate-in data-[state=open]:fade-in-0"
        >
          <Command
            shouldFilter={!onSearchChange}
            filter={(itemValue, search) =>
              itemValue.toLowerCase().includes(search.toLowerCase()) ? 1 : 0
            }
          >
            <div className="border-b border-border px-2.5">
              <CommandInput
                ref={inputRef}
                onValueChange={onSearchChange}
                aria-label={searchPlaceholder}
                placeholder={searchPlaceholder}
                className="h-8 w-full bg-transparent text-sm outline-none placeholder:text-muted-foreground"
              />
            </div>
            <CommandList aria-busy={loading} className="max-h-60 overflow-y-auto p-1">
              {!loading && !error && (
                <CommandEmpty className="px-2 py-3 text-center text-xs text-muted-foreground">
                  {emptyMessage}
                </CommandEmpty>
              )}
              {loading && (
                <p role="status" className="px-2 py-3 text-xs">
                  Loading…
                </p>
              )}
              {error && (
                <p role="alert" className="px-2 py-3 text-xs text-destructive">
                  {error}
                </p>
              )}
              {options.map((option) => (
                <CommandItem
                  key={option.value}
                  value={option.value}
                  onSelect={(next) => {
                    onChange(next);
                    setOpen(false);
                    onOpenChange?.(false);
                  }}
                  className="flex cursor-pointer items-center gap-2 rounded-sm px-2 py-1.5 text-sm select-none data-[selected=true]:bg-accent data-[selected=true]:text-accent-foreground"
                >
                  <Check
                    className={cn(
                      'size-3.5 shrink-0',
                      option.value === value ? 'opacity-100' : 'opacity-0',
                    )}
                    aria-hidden
                  />
                  <span className="truncate">{option.label}</span>
                  {option.description && (
                    <span className="ml-auto truncate text-xs text-muted-foreground">
                      {option.description}
                    </span>
                  )}
                </CommandItem>
              ))}
            </CommandList>
          </Command>
          {error && onRetry && (
            <Button
              type="button"
              variant="ghost"
              onClick={() => {
                onRetry();
                inputRef.current?.focus();
              }}
              className="w-full rounded-none"
            >
              Retry
            </Button>
          )}
          {onNextPage && !loading && !error && (
            <Button
              type="button"
              variant="ghost"
              onClick={() => {
                onNextPage();
                inputRef.current?.focus();
              }}
              className="w-full rounded-none border-t border-border"
            >
              Next results
            </Button>
          )}
        </PopoverPrimitive.Content>
      </PopoverPrimitive.Portal>
    </PopoverPrimitive.Root>
  );
}
