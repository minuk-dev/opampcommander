// Tiny YAML/JSON tokenizer for the config editor's highlight layer.
//
// Deliberately approximate: it is a display aid, not a parser — actual
// validation goes through js-yaml (see fromYAML). It works line by line so the
// highlight layer can be rendered as one <span> per token with no lookbehind
// state, which keeps it cheap to re-run on every keystroke. (Block scalars are
// highlighted as if they were YAML; harmless, since it only affects colour.)

export type TokenKind =
  | 'plain'
  | 'comment'
  | 'key'
  | 'string'
  | 'number'
  | 'literal'
  | 'punctuation';

export interface Token {
  text: string;
  kind: TokenKind;
}

export type HighlightLanguage = 'yaml' | 'json' | 'text';

// languageFor maps a resource content type onto a highlighter.
export function languageFor(contentType: string | undefined): HighlightLanguage {
  const ct = (contentType ?? '').toLowerCase();
  if (ct.includes('yaml') || ct.includes('yml')) return 'yaml';
  if (ct.includes('json')) return 'json';
  return 'text';
}

const YAML_LITERALS = new Set([
  'true',
  'false',
  'null',
  'yes',
  'no',
  'on',
  'off',
  '~',
  'True',
  'False',
  'Null',
]);

function push(tokens: Token[], text: string, kind: TokenKind) {
  if (text === '') return;
  const last = tokens[tokens.length - 1];
  if (last && last.kind === kind) {
    last.text += text;
    return;
  }
  tokens.push({ text, kind });
}

// splitTrailingComment returns [code, comment] for a YAML line. A `#` only
// starts a comment when it follows whitespace and sits outside quotes.
function splitTrailingComment(line: string): [string, string] {
  let quote: string | null = null;
  for (let i = 0; i < line.length; i++) {
    const ch = line[i];
    if (quote) {
      if (ch === '\\' && quote === '"') i++;
      else if (ch === quote) quote = null;
      continue;
    }
    if (ch === '"' || ch === "'") {
      quote = ch;
      continue;
    }
    if (ch === '#' && (i === 0 || /\s/.test(line[i - 1]))) {
      return [line.slice(0, i), line.slice(i)];
    }
  }
  return [line, ''];
}

function tokenizeYamlValue(value: string, tokens: Token[]) {
  // The number alternative refuses to match a prefix of a longer bare word
  // (e.g. the "0.110" in "0.110.0"), which would otherwise split a version
  // string into two differently coloured pieces.
  const re =
    /("(?:[^"\\]|\\.)*"|'(?:[^']|'')*'|-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?(?![^\s,}\]])|[{}[\],]|\S+|\s+)/g;
  for (const [chunk] of value.matchAll(re)) {
    if (/^\s+$/.test(chunk)) push(tokens, chunk, 'plain');
    else if (chunk.startsWith('"') || chunk.startsWith("'")) push(tokens, chunk, 'string');
    else if (/^-?\d/.test(chunk) && /^-?\d+(\.\d+)?([eE][+-]?\d+)?$/.test(chunk))
      push(tokens, chunk, 'number');
    else if (YAML_LITERALS.has(chunk)) push(tokens, chunk, 'literal');
    else if (/^[{}[\],]$/.test(chunk)) push(tokens, chunk, 'punctuation');
    else push(tokens, chunk, 'plain');
  }
}

function tokenizeYamlLine(line: string): Token[] {
  const tokens: Token[] = [];
  const [code, comment] = splitTrailingComment(line);

  // Leading indentation and any number of "- " sequence markers.
  const lead = /^(\s*)((?:-\s+)*)(-\s*$)?/.exec(code);
  let rest = code;
  if (lead) {
    push(tokens, lead[1], 'plain');
    push(tokens, lead[2], 'punctuation');
    if (lead[3]) push(tokens, lead[3], 'punctuation');
    rest = code.slice(lead[0].length);
  }

  const keyMatch = /^([^\s:][^:]*|"(?:[^"\\]|\\.)*"|'(?:[^']|'')*')(:)(\s|$)/.exec(rest);
  if (keyMatch) {
    push(tokens, keyMatch[1], 'key');
    push(tokens, keyMatch[2], 'punctuation');
    push(tokens, keyMatch[3], 'plain');
    tokenizeYamlValue(rest.slice(keyMatch[0].length), tokens);
  } else {
    tokenizeYamlValue(rest, tokens);
  }

  push(tokens, comment, 'comment');
  return tokens;
}

function tokenizeJsonLine(line: string): Token[] {
  const tokens: Token[] = [];
  const re =
    /("(?:[^"\\]|\\.)*"\s*:|"(?:[^"\\]|\\.)*"|-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?|true|false|null|[{}[\],:]|\s+|[^\s{}[\],:"]+)/g;
  for (const [chunk] of line.matchAll(re)) {
    if (/^\s+$/.test(chunk)) push(tokens, chunk, 'plain');
    else if (chunk.startsWith('"') && chunk.trimEnd().endsWith(':')) push(tokens, chunk, 'key');
    else if (chunk.startsWith('"')) push(tokens, chunk, 'string');
    else if (/^-?\d/.test(chunk)) push(tokens, chunk, 'number');
    else if (chunk === 'true' || chunk === 'false' || chunk === 'null')
      push(tokens, chunk, 'literal');
    else if (/^[{}[\],:]$/.test(chunk)) push(tokens, chunk, 'punctuation');
    else push(tokens, chunk, 'plain');
  }
  return tokens;
}

// tokenizeLines returns one token list per line of `text`.
export function tokenizeLines(text: string, language: HighlightLanguage): Token[][] {
  const lines = text.split('\n');
  if (language === 'text') return lines.map((line) => [{ text: line, kind: 'plain' as const }]);
  const tokenize = language === 'yaml' ? tokenizeYamlLine : tokenizeJsonLine;
  return lines.map((line) => (line === '' ? [] : tokenize(line)));
}
