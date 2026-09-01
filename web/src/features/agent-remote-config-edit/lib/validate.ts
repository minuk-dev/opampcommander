// Client-side checks for the remote-config editor. Kept deliberately loose:
// the server is the authority, so these only catch what would certainly fail
// (an unparseable config body, a name that cannot form a URL path).

import { fromYAML, languageFor } from '@shared/lib';

export interface ParseProblem {
  message: string;
  // 1-based line the parser pointed at, when it reported one.
  line?: number;
}

interface YamlMark {
  line?: number;
}

function markLine(err: unknown): number | undefined {
  if (!err || typeof err !== 'object' || !('mark' in err)) return undefined;
  const mark = (err as { mark?: YamlMark }).mark;
  // js-yaml marks are 0-based.
  return typeof mark?.line === 'number' ? mark.line + 1 : undefined;
}

function jsonErrorLine(text: string, err: unknown): number | undefined {
  const message = err instanceof Error ? err.message : '';
  const at = /position (\d+)/.exec(message);
  if (!at) return undefined;
  const offset = Number(at[1]);
  return text.slice(0, offset).split('\n').length;
}

// validateConfigBody parses the config body with the parser its content type
// implies. Content types we cannot parse (text/plain, anything custom) are
// accepted as-is.
export function validateConfigBody(body: string, contentType: string): ParseProblem | null {
  const language = languageFor(contentType);
  if (language === 'text' || body.trim() === '') return null;
  try {
    if (language === 'yaml') fromYAML(body);
    else JSON.parse(body);
    return null;
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    const line = language === 'yaml' ? markLine(err) : jsonErrorLine(body, err);
    return line ? { message, line } : { message };
  }
}
