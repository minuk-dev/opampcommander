import { describe, expect, it } from 'vitest';
import type { AgentGroup, AgentGroupRemoteConfig } from '../model/types';
import {
  describeRemoteConfigSources,
  hasRemoteConfig,
  inlineRemoteConfigs,
  remoteConfigRefs,
  withRemoteConfigRefs,
} from './remote-config';

function group(configs?: AgentGroupRemoteConfig[]): AgentGroup {
  return {
    metadata: { namespace: 'default', name: 'g', attributes: {}, createdAt: '' },
    spec: {
      priority: 0,
      selector: {},
      agentConfig: configs ? { agentRemoteConfigs: configs } : undefined,
    },
    status: {
      numAgents: 0,
      numConnectedAgents: 0,
      numHealthyAgents: 0,
      numUnhealthyAgents: 0,
      numNotConnectedAgents: 0,
    },
  };
}

describe('remote-config helpers', () => {
  it('treats a group with no agentConfig as empty', () => {
    const g = group();
    expect(remoteConfigRefs(g)).toEqual([]);
    expect(hasRemoteConfig(g)).toBe(false);
    expect(describeRemoteConfigSources(g)).toBe('none');
  });

  it('lists refs in declared order and ignores inline entries', () => {
    const g = group([
      { agentRemoteConfigRef: 'a' },
      { agentRemoteConfigName: 'inline-1', agentRemoteConfigSpec: { value: 'x', contentType: 'text/yaml' } },
      { agentRemoteConfigRef: 'b' },
    ]);
    expect(remoteConfigRefs(g)).toEqual(['a', 'b']);
    expect(inlineRemoteConfigs(g)).toHaveLength(1);
    expect(hasRemoteConfig(g)).toBe(true);
    expect(describeRemoteConfigSources(g)).toBe('ref → a, inline (inline-1), ref → b');
  });

  it('rewrites refs while preserving inline entries and collapsing duplicates', () => {
    const inline = {
      agentRemoteConfigName: 'inline-1',
      agentRemoteConfigSpec: { value: 'x', contentType: 'text/yaml' },
    };
    const g = group([{ agentRemoteConfigRef: 'a' }, inline]);

    // Replace the ref set with b + c (a removed, duplicate c collapsed); inline kept.
    expect(withRemoteConfigRefs(g, ['b', 'c', 'c'])).toEqual([
      inline,
      { agentRemoteConfigRef: 'b' },
      { agentRemoteConfigRef: 'c' },
    ]);
  });

  it('clears refs but keeps inline entries when given an empty list', () => {
    const inline = {
      agentRemoteConfigName: 'inline-1',
      agentRemoteConfigSpec: { value: 'x', contentType: 'text/yaml' },
    };
    const g = group([{ agentRemoteConfigRef: 'a' }, inline]);
    expect(withRemoteConfigRefs(g, [])).toEqual([inline]);
  });
});
