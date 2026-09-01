import { describe, expect, it } from 'vitest';
import { languageFor, tokenizeLines } from './highlight';

// The highlight layer is painted underneath a transparent textarea, so any
// character the tokenizer drops or duplicates shifts the visible text out from
// under the caret. Exact round-tripping is the property that matters most.
function roundTrip(text: string, language: 'yaml' | 'json' | 'text') {
  return tokenizeLines(text, language)
    .map((tokens) => tokens.map((t) => t.text).join(''))
    .join('\n');
}

const COLLECTOR_CONFIG = `# a collector config
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
processors:
  batch: {}
  memory_limiter:
    limit_mib: 512
    check_interval: 1s
exporters:
  otlphttp:
    endpoint: https://otlp.example.com # trailing comment
    headers:
      authorization: "Bearer \${env:TOKEN}"
service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [otlphttp]
`;

describe('tokenizeLines', () => {
  it('round-trips a collector config exactly', () => {
    expect(roundTrip(COLLECTOR_CONFIG, 'yaml')).toBe(COLLECTOR_CONFIG);
  });

  it('round-trips JSON exactly', () => {
    const json = '{\n  "a": 1,\n  "b": [true, null],\n  "c": "x"\n}\n';
    expect(roundTrip(json, 'json')).toBe(json);
  });

  it('round-trips arbitrary text', () => {
    const text = 'no: [structure\n\there';
    expect(roundTrip(text, 'text')).toBe(text);
  });

  it('marks map keys, values and comments', () => {
    const [line] = tokenizeLines('  endpoint: 0.0.0.0:4317 # note', 'yaml');
    expect(line.find((t) => t.kind === 'key')?.text).toBe('endpoint');
    expect(line.find((t) => t.kind === 'comment')?.text).toBe('# note');
  });

  it('does not treat a "#" inside a quoted value as a comment', () => {
    const [line] = tokenizeLines('key: "a # b"', 'yaml');
    expect(line.some((t) => t.kind === 'comment')).toBe(false);
  });

  it('keeps a dotted version whole instead of splitting off a leading number', () => {
    const [line] = tokenizeLines('version: 0.110.0', 'yaml');
    // A bare "0.110" number token would leave ".0" coloured differently.
    expect(line.some((t) => t.kind === 'number')).toBe(false);
    expect(line.some((t) => t.text.includes('0.110.0'))).toBe(true);
  });

  it('still recognises a plain number value', () => {
    const [line] = tokenizeLines('limit_mib: 512', 'yaml');
    expect(line.find((t) => t.kind === 'number')?.text).toBe('512');
  });

  it('highlights sequence items', () => {
    const [line] = tokenizeLines('  - name: otlp', 'yaml');
    expect(line.find((t) => t.kind === 'key')?.text).toBe('name');
  });
});

describe('languageFor', () => {
  it.each([
    ['text/yaml', 'yaml'],
    ['application/x-yaml', 'yaml'],
    ['text/yml', 'yaml'],
    ['application/json', 'json'],
    ['text/plain', 'text'],
    [undefined, 'text'],
  ])('maps %s to %s', (contentType, expected) => {
    expect(languageFor(contentType)).toBe(expected);
  });
});
