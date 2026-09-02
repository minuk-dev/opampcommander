export { default as CodeBlock, type CodeFormat } from './CodeBlock';
export { default as JsonBlock } from './JsonBlock';
export { default as ConfirmDialog } from './ConfirmDialog';
export { default as CodeEditorDialog, type CodeSample } from './CodeEditorDialog';
export { default as JsonEditorDialog } from './JsonEditorDialog';
export { default as PaginationFooter } from './PaginationFooter';
export { default as ColumnPicker } from './ColumnPicker';
export { default as RowActionsMenu, type RowAction } from './RowActionsMenu';
export { default as PageHeader } from './PageHeader';
// CodeTextEditor and DiffView are intentionally NOT re-exported: they are
// only ever needed once an editor dialog opens, so call sites import them
// lazily (next/dynamic on '@shared/ui/CodeTextEditor') to keep them out of the
// initial route bundles.
