export interface Connection {
  id: string;
  instanceUid: string;
  namespace: string;
  type: string;
  lastCommunicatedAt: string;
  alive: boolean;
}
