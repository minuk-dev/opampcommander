'use client';

// Backwards-compatible alias. Existing call sites now get YAML-by-default
// editing with a JSON toggle, via CodeEditorDialog. New code should prefer
// CodeEditorDialog directly.
import CodeEditorDialog from './CodeEditorDialog';

export default CodeEditorDialog;
