export interface ServerCondition {
  type: 'Registered' | 'Alive' | string;
  lastTransitionTime: string;
  status: 'True' | 'False' | 'Unknown';
  reason: string;
  message?: string;
}

export interface Server {
  id: string;
  lastHeartbeatAt: string;
  conditions?: ServerCondition[];
}
