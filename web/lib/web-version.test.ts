import { describe, expect, it } from 'vitest';
import { getWebVersionInfo, parseMajorMinor } from './web-version';

describe('parseMajorMinor', () => {
  it('parses a snapshot version', () => {
    expect(parseMajorMinor('0.1.41-SNAPSHOT-14bc29e')).toEqual({ major: '0', minor: '1' });
  });

  it('parses a plain semver', () => {
    expect(parseMajorMinor('1.20.3')).toEqual({ major: '1', minor: '20' });
  });

  it('tolerates a leading v', () => {
    expect(parseMajorMinor('v2.5.0')).toEqual({ major: '2', minor: '5' });
  });

  it('returns empty parts for an unrecognized version', () => {
    expect(parseMajorMinor('unknown')).toEqual({ major: '', minor: '' });
  });
});

describe('getWebVersionInfo', () => {
  it('reports Node runtime fields', () => {
    const info = getWebVersionInfo();
    expect(info.nodeVersion).toBe(process.version);
    expect(info.platform).toBe(`${process.platform}/${process.arch}`);
    expect(info.nextVersion).toMatch(/\d+\.\d+/);
  });
});
