// Primitives (Radix + Tailwind, styled here so the app never imports a
// component library's own classes).
export { default as Alert } from './Alert';
export { default as Badge, badgeVariants } from './Badge';
export { default as Button, buttonVariants } from './Button';
export { default as Checkbox } from './Checkbox';
export { default as Collapsible } from './Collapsible';
export { default as Combobox, type ComboboxOption } from './Combobox';
export { default as Field } from './Field';
export { default as Input, fieldBase } from './Input';
export { default as Label } from './Label';
export { default as Progress } from './Progress';
export { default as Separator } from './Separator';
export { default as Skeleton } from './Skeleton';
export { default as Spinner } from './Spinner';
export { default as Switch } from './Switch';
export { default as Textarea } from './Textarea';
export { default as Tooltip, TooltipProvider } from './Tooltip';
export * from './Card';
export * from './Dialog';
export * from './DropdownMenu';
export * from './SegmentedControl';
export * from './Select';
export * from './Sheet';
export * from './Table';
export * from './Tabs';
export * from './Toast';

// Composites built on the primitives.
export { default as CodeBlock, type CodeFormat } from './CodeBlock';
export { default as JsonBlock } from './JsonBlock';
export { default as ConfirmDialog } from './ConfirmDialog';
export { default as CodeEditorDialog, type CodeSample } from './CodeEditorDialog';
export { default as JsonEditorDialog } from './JsonEditorDialog';
export { default as PaginationFooter } from './PaginationFooter';
export { default as ColumnPicker } from './ColumnPicker';
export { default as ConditionBadges } from './ConditionBadges';
export { default as RowActionsMenu, type RowAction } from './RowActionsMenu';
export { default as PageHeader } from './PageHeader';
export { default as SampleMenu } from './SampleMenu';
// CodeTextEditor and DiffView are intentionally NOT re-exported: they are
// only ever needed once an editor dialog opens, so call sites import them
// lazily (next/dynamic on '@shared/ui/CodeTextEditor') to keep them out of the
// initial route bundles.
